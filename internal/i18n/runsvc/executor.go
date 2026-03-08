package runsvc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/cache"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/lockfile"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

const (
	lockPersistBatchSize     = 32
	lockPersistFlushInterval = 250 * time.Millisecond
)

type executionReport struct {
	Succeeded       int
	Failed          int
	PersistedToLock int
	TokenUsage
	LocaleUsage                 map[string]TokenUsage
	Batches                     []BatchUsage
	Failures                    []Failure
	ContextMemoryGenerated      int
	ContextMemoryFallbackGroups int
	Warnings                    []string
}

type taskCompletion struct {
	identity     string
	entryKey     string
	value        string
	sourceHash   string
	targetPath   string
	sourcePath   string
	targetLocale string
}

type stagedOutput struct {
	entries      map[string]string
	sourcePath   string
	sourceLocale string
	targetLocale string
}

type executorState struct {
	total                int
	staged               map[string]stagedOutput
	flushedTargets       map[string]struct{}
	failedTargets        map[string]struct{}
	idsByTarget          map[string][]string
	pendingByTarget      map[string]int
	sourceByTarget       map[string]string
	sourceLocaleByTarget map[string]string
	localeByTarget       map[string]string
	pruneTargets         map[string]map[string]struct{}
	contextPlan          contextMemoryPlan
	contextSlots         map[string]*contextMemorySlot
	report               executionReport

	stageMu   sync.Mutex
	pendingMu sync.Mutex
	reportMu  sync.Mutex
	contextMu sync.Mutex
}

type contextMemorySlot struct {
	done   chan struct{}
	memory string
}

func newExecutorState(tasks []Task, initialStaged map[string]stagedOutput, pruneTargets map[string]map[string]struct{}, contextPlan contextMemoryPlan) (*executorState, error) {
	staged := map[string]stagedOutput{}
	for targetPath, output := range initialStaged {
		entries := map[string]string{}
		for key, value := range output.entries {
			entries[key] = value
		}
		staged[targetPath] = stagedOutput{entries: entries, sourcePath: output.sourcePath, sourceLocale: output.sourceLocale, targetLocale: output.targetLocale}
	}

	state := &executorState{
		total:                len(tasks),
		staged:               staged,
		flushedTargets:       map[string]struct{}{},
		failedTargets:        map[string]struct{}{},
		idsByTarget:          map[string][]string{},
		pendingByTarget:      map[string]int{},
		sourceByTarget:       map[string]string{},
		sourceLocaleByTarget: map[string]string{},
		localeByTarget:       map[string]string{},
		pruneTargets:         pruneTargets,
		contextPlan:          contextPlan,
		contextSlots:         map[string]*contextMemorySlot{},
		report:               executionReport{LocaleUsage: map[string]TokenUsage{}},
	}
	for _, task := range tasks {
		state.pendingByTarget[task.TargetPath]++
		state.idsByTarget[task.TargetPath] = append(state.idsByTarget[task.TargetPath], taskIdentity(task.TargetPath, task.EntryKey))
		existing := state.sourceByTarget[task.TargetPath]
		if existing != "" && existing != task.SourcePath {
			return nil, fmt.Errorf("output staging conflict: %s has conflicting source paths", task.TargetPath)
		}
		state.sourceByTarget[task.TargetPath] = task.SourcePath
		existingSourceLocale := state.sourceLocaleByTarget[task.TargetPath]
		if existingSourceLocale != "" && existingSourceLocale != task.SourceLocale {
			return nil, fmt.Errorf("output staging conflict: %s has conflicting source locales", task.TargetPath)
		}
		state.sourceLocaleByTarget[task.TargetPath] = task.SourceLocale
		existingLocale := state.localeByTarget[task.TargetPath]
		if existingLocale != "" && existingLocale != task.TargetLocale {
			return nil, fmt.Errorf("output staging conflict: %s has conflicting target locales", task.TargetPath)
		}
		state.localeByTarget[task.TargetPath] = task.TargetLocale
	}
	return state, nil
}

func (s *Service) executePool(ctx context.Context, tasks []Task, initialStaged map[string]stagedOutput, lockPath string, lockState *lockfile.File, workers int, activeRunID string, pruneTargets map[string]map[string]struct{}, contextPlan contextMemoryPlan, l1 cache.ExactCache, emitter *eventEmitter) (map[string]stagedOutput, map[string]struct{}, executionReport, error) {
	scheduledTasks := tasks
	if contextPlan.Enabled {
		scheduledTasks = interleaveTasksByContextKey(tasks)
	}

	state, err := newExecutorState(scheduledTasks, initialStaged, pruneTargets, contextPlan)
	if err != nil {
		return nil, nil, executionReport{}, err
	}

	if contextPlan.Enabled {
		if err := s.precomputeContextMemory(ctx, state, emitter, workers); err != nil {
			return nil, nil, state.report, err
		}
	}

	workerCount := workers
	if workerCount == 0 {
		workerCount = s.numCPU()
	}
	if workerCount < 1 {
		workerCount = 1
	}

	jobs := make(chan Task)
	completions := make(chan taskCompletion)
	targetFailures := make(chan string)
	fatalLockErr := make(chan error, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	lockWriterDone := make(chan struct{})
	go s.runLockWriter(ctx, completions, targetFailures, lockWriterDone, lockState, lockPath, activeRunID, fatalLockErr, cancel, state, emitter)

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go s.runWorker(ctx, jobs, completions, targetFailures, state, l1, emitter, &wg, cancel)
	}

	go s.feedJobs(ctx, jobs, scheduledTasks)

	wg.Wait()
	close(completions)
	close(targetFailures)
	<-lockWriterDone

	select {
	case err := <-fatalLockErr:
		return nil, nil, state.report, err
	default:
	}

	return state.staged, state.flushedTargets, state.report, nil
}

func (s *Service) precomputeContextMemory(ctx context.Context, state *executorState, emitter *eventEmitter, workers int) error {
	if !state.contextPlan.Enabled || len(state.contextPlan.Groups) == 0 {
		return nil
	}

	workerCount := workers
	if workerCount == 0 {
		workerCount = s.numCPU()
	}
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(state.contextPlan.Groups) {
		workerCount = len(state.contextPlan.Groups)
	}

	jobs := make(chan string)
	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case key, ok := <-jobs:
					if !ok {
						return
					}
					_ = s.resolveTaskContextMemory(ctx, Task{ContextKey: key}, state, emitter)
				}
			}
		}()
	}

	for key := range state.contextPlan.Groups {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		case jobs <- key:
		}
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *Service) runLockWriter(ctx context.Context, completions <-chan taskCompletion, targetFailures <-chan string, done chan<- struct{}, lockState *lockfile.File, lockPath string, activeRunID string, fatalLockErr chan<- error, cancel context.CancelFunc, state *executorState, emitter *eventEmitter) {
	defer close(done)
	flushInterval := s.lockPersistFlushInterval
	if flushInterval <= 0 {
		flushInterval = lockPersistFlushInterval
	}
	batchSize := s.lockPersistBatchSize
	if batchSize <= 0 {
		batchSize = lockPersistBatchSize
	}

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	dirty := false
	pendingPersisted := map[string]struct{}{}

	flushPending := func(errPrefix string, removedPersisted int) error {
		if !dirty {
			return nil
		}
		if err := s.saveLock(lockPath, *lockState); err != nil {
			return fmt.Errorf("%s: %w", errPrefix, err)
		}
		addedPersisted := len(pendingPersisted)
		pendingPersisted = map[string]struct{}{}
		dirty = false

		if addedPersisted == 0 && removedPersisted == 0 {
			return nil
		}

		state.reportMu.Lock()
		state.report.PersistedToLock += addedPersisted
		state.report.PersistedToLock -= removedPersisted
		if state.report.PersistedToLock < 0 {
			state.report.PersistedToLock = 0
		}
		persisted := state.report.PersistedToLock
		succeeded := state.report.Succeeded
		failed := state.report.Failed
		state.reportMu.Unlock()
		emitter.emit(Event{Kind: EventPersisted, PersistedToLock: persisted, Succeeded: succeeded, Failed: failed})
		return nil
	}

	reportFatal := func(err error) {
		select {
		case fatalLockErr <- err:
		default:
		}
	}

	willTargetFlush := func(targetPath string) bool {
		state.pendingMu.Lock()
		remaining := state.pendingByTarget[targetPath]
		state.pendingMu.Unlock()
		return remaining <= 1
	}

	completionCh := completions
	failureCh := targetFailures
	for {
		if completionCh == nil && failureCh == nil {
			if err := flushPending("persist lock state", 0); err != nil {
				reportFatal(err)
			}
			return
		}
		select {
		case <-ctx.Done():
			if err := flushPending("persist lock state", 0); err != nil {
				reportFatal(err)
			}
			return
		case <-ticker.C:
			if err := flushPending("persist lock state", 0); err != nil {
				reportFatal(err)
				cancel()
				return
			}
		case completion, ok := <-completionCh:
			if !ok {
				completionCh = nil
				continue
			}
			if isTargetFailed(completion.targetPath, &state.pendingMu, state.failedTargets) {
				if err := s.flushIfTargetCompleted(completion.targetPath, completion.sourcePath, state); err != nil {
					recordTaskFailure(&state.report, &state.reportMu, state.total, Task{TargetPath: completion.targetPath}, err, emitter)
				}
				continue
			}
			lockState.RunCompleted[completion.identity] = lockfile.RunCompletion{SourceHash: completion.sourceHash}
			lockState.RunCheckpoint[completion.identity] = lockfile.RunCheckpoint{
				RunID:        activeRunID,
				TargetPath:   completion.targetPath,
				SourcePath:   completion.sourcePath,
				TargetLocale: completion.targetLocale,
				EntryKey:     completion.entryKey,
				Value:        completion.value,
				SourceHash:   completion.sourceHash,
				UpdatedAt:    s.now(),
			}
			pendingPersisted[completion.identity] = struct{}{}
			dirty = true
			if len(pendingPersisted) >= batchSize {
				if err := flushPending("persist lock state", 0); err != nil {
					reportFatal(err)
					cancel()
					return
				}
			}
			if willTargetFlush(completion.targetPath) {
				if err := flushPending("persist lock state", 0); err != nil {
					reportFatal(err)
					cancel()
					return
				}
			}

			if err := s.flushIfTargetCompleted(completion.targetPath, completion.sourcePath, state); err != nil {
				recordTaskFailure(&state.report, &state.reportMu, state.total, Task{TargetPath: completion.targetPath}, err, emitter)
				removedPersisted, changed := s.rollbackLockForTarget(lockState, completion.targetPath, pendingPersisted, state)
				if !changed {
					continue
				}
				dirty = true
				if err := flushPending(fmt.Sprintf("persist lock rollback for %q", completion.targetPath), removedPersisted); err != nil {
					reportFatal(err)
					cancel()
					return
				}
				continue
			}
		case targetPath, ok := <-failureCh:
			if !ok {
				failureCh = nil
				continue
			}
			removedPersisted, changed := s.rollbackLockForTarget(lockState, targetPath, pendingPersisted, state)
			if !changed {
				continue
			}
			dirty = true
			if err := flushPending(fmt.Sprintf("persist lock rollback for %q", targetPath), removedPersisted); err != nil {
				reportFatal(err)
				cancel()
				return
			}
		}
	}
}

func (s *Service) runWorker(ctx context.Context, jobs <-chan Task, completions chan<- taskCompletion, targetFailures chan<- string, state *executorState, l1 cache.ExactCache, emitter *eventEmitter, wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-jobs:
			if !ok {
				return
			}
			if s.processTask(ctx, task, completions, targetFailures, state, l1, emitter) {
				continue
			}
			if err := s.flushIfTargetCompleted(task.TargetPath, task.SourcePath, state); err != nil {
				recordTaskFailure(&state.report, &state.reportMu, state.total, task, err, emitter)
				continue
			}
		}
	}
}

func (s *Service) flushIfTargetCompleted(targetPath, sourcePath string, state *executorState) error {
	shouldFlush := false
	expectedSourcePath := sourcePath
	expectedSourceLocale := ""
	expectedTargetLocale := ""
	state.pendingMu.Lock()
	remaining := state.pendingByTarget[targetPath]
	if remaining > 0 {
		remaining--
		state.pendingByTarget[targetPath] = remaining
	}
	if remaining == 0 {
		if _, done := state.flushedTargets[targetPath]; !done {
			shouldFlush = true
			state.flushedTargets[targetPath] = struct{}{}
			if knownSourcePath := state.sourceByTarget[targetPath]; knownSourcePath != "" {
				expectedSourcePath = knownSourcePath
			}
			expectedSourceLocale = state.sourceLocaleByTarget[targetPath]
			expectedTargetLocale = state.localeByTarget[targetPath]
		}
	}
	state.pendingMu.Unlock()
	if !shouldFlush {
		return nil
	}
	if isTargetFailed(targetPath, &state.pendingMu, state.failedTargets) {
		return nil
	}

	state.stageMu.Lock()
	output, ok := state.staged[targetPath]
	if ok {
		delete(state.staged, targetPath)
	}
	state.stageMu.Unlock()

	if !ok {
		output = stagedOutput{entries: map[string]string{}, sourcePath: expectedSourcePath, sourceLocale: expectedSourceLocale, targetLocale: expectedTargetLocale}
	} else if output.sourcePath == "" {
		output.sourcePath = expectedSourcePath
	}
	if output.sourceLocale == "" {
		output.sourceLocale = expectedSourceLocale
	}
	if output.targetLocale == "" {
		output.targetLocale = expectedTargetLocale
	}

	warnings, err := s.flushOutputForTarget(targetPath, output, state.pruneTargets[targetPath])
	if err != nil {
		return err
	}
	if len(warnings) > 0 {
		state.reportMu.Lock()
		state.report.Warnings = append(state.report.Warnings, warnings...)
		state.reportMu.Unlock()
	}
	return nil
}

func (s *Service) processTask(ctx context.Context, task Task, completions chan<- taskCompletion, targetFailures chan<- string, state *executorState, l1 cache.ExactCache, emitter *eventEmitter) bool {
	state.reportMu.Lock()
	startedSucceeded := state.report.Succeeded
	startedFailed := state.report.Failed
	state.reportMu.Unlock()
	emitter.emit(Event{
		Kind:            EventTaskStart,
		TargetPath:      task.TargetPath,
		EntryKey:        task.EntryKey,
		Succeeded:       startedSucceeded,
		Failed:          startedFailed,
		ExecutableTotal: state.total,
	})

	usage := translator.Usage{}
	if state.contextPlan.Enabled {
		task.ContextMemory = s.resolveTaskContextMemory(ctx, task, state, emitter)
	}
	cacheKey := exactCacheKey(task)
	if l1 != nil {
		cachedValue, hit, err := l1.Get(ctx, cacheKey)
		if err != nil {
			state.reportMu.Lock()
			state.report.Warnings = append(state.report.Warnings, fmt.Sprintf(`cache_l1_get_failed target=%q key=%q error=%q`, task.TargetPath, task.EntryKey, err.Error()))
			state.reportMu.Unlock()
		} else if hit {
			if err := stageTaskOutput(state.staged, task.TargetPath, task.SourcePath, task.SourceLocale, task.TargetLocale, task.EntryKey, cachedValue, &state.stageMu); err != nil {
				recordTaskFailure(&state.report, &state.reportMu, state.total, task, err, emitter)
				markTargetFailed(task.TargetPath, &state.pendingMu, state.failedTargets, targetFailures, ctx)
				return false
			}
			select {
			case completions <- taskCompletion{identity: taskIdentity(task.TargetPath, task.EntryKey), entryKey: task.EntryKey, value: cachedValue, sourceHash: hashSourceText(task.SourceText), targetPath: task.TargetPath, sourcePath: task.SourcePath, targetLocale: task.TargetLocale}:
				state.reportMu.Lock()
				state.report.Succeeded++
				succeeded := state.report.Succeeded
				failed := state.report.Failed
				tokenUsage := state.report.TokenUsage
				state.reportMu.Unlock()
				emitter.emit(Event{
					Kind:             EventTaskDone,
					TaskSucceeded:    true,
					TargetPath:       task.TargetPath,
					EntryKey:         task.EntryKey,
					Succeeded:        succeeded,
					Failed:           failed,
					ExecutableTotal:  state.total,
					PromptTokens:     tokenUsage.PromptTokens,
					CompletionTokens: tokenUsage.CompletionTokens,
					TotalTokens:      tokenUsage.TotalTokens,
				})
				return true
			case <-ctx.Done():
				return false
			}
		}
	}
	translated, err := s.translateWithRetry(translator.WithUsageCollector(ctx, &usage), task)
	if err != nil {
		recordTaskFailure(&state.report, &state.reportMu, state.total, task, err, emitter)
		markTargetFailed(task.TargetPath, &state.pendingMu, state.failedTargets, targetFailures, ctx)
		return false
	}
	if l1 != nil {
		if err := l1.Put(ctx, cache.ExactCacheWrite{
			Key:          cacheKey,
			Value:        translated,
			SourceLocale: task.SourceLocale,
			TargetLocale: task.TargetLocale,
			Provider:     task.Provider,
			Model:        task.Model,
			SourceHash:   hashSourceText(task.SourceText),
		}); err != nil {
			state.reportMu.Lock()
			state.report.Warnings = append(state.report.Warnings, fmt.Sprintf(`cache_l1_put_failed target=%q key=%q error=%q`, task.TargetPath, task.EntryKey, err.Error()))
			state.reportMu.Unlock()
		}
	}
	if err := stageTaskOutput(state.staged, task.TargetPath, task.SourcePath, task.SourceLocale, task.TargetLocale, task.EntryKey, translated, &state.stageMu); err != nil {
		recordTaskFailure(&state.report, &state.reportMu, state.total, task, err, emitter)
		markTargetFailed(task.TargetPath, &state.pendingMu, state.failedTargets, targetFailures, ctx)
		return false
	}

	select {
	case completions <- taskCompletion{identity: taskIdentity(task.TargetPath, task.EntryKey), entryKey: task.EntryKey, value: translated, sourceHash: hashSourceText(task.SourceText), targetPath: task.TargetPath, sourcePath: task.SourcePath, targetLocale: task.TargetLocale}:
		state.reportMu.Lock()
		state.report.Succeeded++
		state.report.TokenUsage = addTokenUsage(state.report.TokenUsage, toRunTokenUsage(usage))
		localeUsage := state.report.LocaleUsage[task.TargetLocale]
		state.report.LocaleUsage[task.TargetLocale] = addTokenUsage(localeUsage, toRunTokenUsage(usage))
		state.report.Batches = append(state.report.Batches, BatchUsage{
			TargetLocale: task.TargetLocale,
			TargetPath:   task.TargetPath,
			EntryKey:     task.EntryKey,
			TokenUsage:   toRunTokenUsage(usage),
		})
		succeeded := state.report.Succeeded
		failed := state.report.Failed
		tokenUsage := state.report.TokenUsage
		state.reportMu.Unlock()
		emitter.emit(Event{
			Kind:             EventTaskDone,
			TaskSucceeded:    true,
			TargetPath:       task.TargetPath,
			EntryKey:         task.EntryKey,
			Succeeded:        succeeded,
			Failed:           failed,
			ExecutableTotal:  state.total,
			PromptTokens:     tokenUsage.PromptTokens,
			CompletionTokens: tokenUsage.CompletionTokens,
			TotalTokens:      tokenUsage.TotalTokens,
		})
		return true
	case <-ctx.Done():
		return false
	}
}

func toRunTokenUsage(usage translator.Usage) TokenUsage {
	return TokenUsage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func addTokenUsage(current TokenUsage, delta TokenUsage) TokenUsage {
	current.PromptTokens += delta.PromptTokens
	current.CompletionTokens += delta.CompletionTokens
	current.TotalTokens += delta.TotalTokens
	return current
}

func (s *Service) feedJobs(ctx context.Context, jobs chan<- Task, tasks []Task) {
	defer close(jobs)
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return
		case jobs <- task:
		}
	}
}

func recordTaskFailure(report *executionReport, reportMu *sync.Mutex, total int, task Task, err error, emitter *eventEmitter) {
	reportMu.Lock()
	report.Failed++
	report.Failures = append(report.Failures, Failure{TargetPath: task.TargetPath, EntryKey: task.EntryKey, Reason: err.Error()})
	succeeded := report.Succeeded
	failed := report.Failed
	tokenUsage := report.TokenUsage
	reportMu.Unlock()
	emitter.emit(Event{
		Kind:             EventTaskDone,
		TaskSucceeded:    false,
		TargetPath:       task.TargetPath,
		EntryKey:         task.EntryKey,
		FailureReason:    err.Error(),
		Succeeded:        succeeded,
		Failed:           failed,
		ExecutableTotal:  total,
		PromptTokens:     tokenUsage.PromptTokens,
		CompletionTokens: tokenUsage.CompletionTokens,
		TotalTokens:      tokenUsage.TotalTokens,
	})
}

func stageTaskOutput(staged map[string]stagedOutput, targetPath, sourcePath, sourceLocale, targetLocale, entryKey, value string, stageMu *sync.Mutex) error {
	if stageMu != nil {
		stageMu.Lock()
		defer stageMu.Unlock()
	}

	bucket, ok := staged[targetPath]
	if !ok {
		bucket = stagedOutput{entries: map[string]string{}, sourcePath: sourcePath, sourceLocale: sourceLocale, targetLocale: targetLocale}
		staged[targetPath] = bucket
	} else if bucket.sourcePath != sourcePath {
		return fmt.Errorf("output staging conflict: %s has conflicting source paths", targetPath)
	} else if bucket.sourceLocale != "" && bucket.sourceLocale != sourceLocale {
		return fmt.Errorf("output staging conflict: %s has conflicting source locales", targetPath)
	} else if bucket.targetLocale != "" && bucket.targetLocale != targetLocale {
		return fmt.Errorf("output staging conflict: %s has conflicting target locales", targetPath)
	}

	if existing, exists := bucket.entries[entryKey]; exists && existing != value {
		return fmt.Errorf("output staging conflict: %s already staged with different value", taskIdentity(targetPath, entryKey))
	}
	bucket.entries[entryKey] = value
	staged[targetPath] = bucket
	return nil
}

func markTargetFailed(targetPath string, mu *sync.Mutex, failedTargets map[string]struct{}, targetFailures chan<- string, ctx context.Context) {
	newFailure := false
	mu.Lock()
	if _, failed := failedTargets[targetPath]; !failed {
		newFailure = true
	}
	failedTargets[targetPath] = struct{}{}
	mu.Unlock()

	if !newFailure {
		return
	}

	select {
	case targetFailures <- targetPath:
	case <-ctx.Done():
	}
}

func isTargetFailed(targetPath string, mu *sync.Mutex, failedTargets map[string]struct{}) bool {
	mu.Lock()
	_, failed := failedTargets[targetPath]
	mu.Unlock()
	return failed
}

func (s *Service) rollbackLockForTarget(lockState *lockfile.File, targetPath string, pendingPersisted map[string]struct{}, state *executorState) (int, bool) {
	ids := state.idsByTarget[targetPath]
	if len(ids) == 0 {
		return 0, false
	}

	removedPersisted := 0
	checkpointRemoved := 0
	changed := false
	for _, id := range ids {
		if _, ok := lockState.RunCompleted[id]; ok {
			delete(lockState.RunCompleted, id)
			changed = true
			if _, pending := pendingPersisted[id]; pending {
				delete(pendingPersisted, id)
			} else {
				removedPersisted++
			}
		}
		if _, ok := lockState.RunCheckpoint[id]; ok {
			delete(lockState.RunCheckpoint, id)
			checkpointRemoved++
			changed = true
		}
	}
	if !changed || (removedPersisted == 0 && checkpointRemoved == 0) {
		return 0, changed
	}
	return removedPersisted, true
}
