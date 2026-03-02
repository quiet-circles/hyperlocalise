package runsvc

import (
	"context"
	"fmt"
	"sync"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/lockfile"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

type executionReport struct {
	Succeeded       int
	Failed          int
	PersistedToLock int
	Failures        []Failure
}

type taskCompletion struct {
	identity   string
	sourceHash string
	targetPath string
	sourcePath string
}

type stagedOutput struct {
	entries       map[string]string
	sourcePath    string
	targetLocale  string
}

type executorState struct {
	total           int
	staged          map[string]stagedOutput
	flushedTargets  map[string]struct{}
	failedTargets   map[string]struct{}
	idsByTarget     map[string][]string
	pendingByTarget map[string]int
	sourceByTarget  map[string]string
	localeByTarget  map[string]string
	pruneTargets    map[string]map[string]struct{}
	report          executionReport

	stageMu   sync.Mutex
	pendingMu sync.Mutex
	reportMu  sync.Mutex
}

func newExecutorState(tasks []Task, pruneTargets map[string]map[string]struct{}) (*executorState, error) {
	state := &executorState{
		total:           len(tasks),
		staged:          map[string]stagedOutput{},
		flushedTargets:  map[string]struct{}{},
		failedTargets:   map[string]struct{}{},
		idsByTarget:     map[string][]string{},
		pendingByTarget: map[string]int{},
		sourceByTarget:  map[string]string{},
		localeByTarget:  map[string]string{},
		pruneTargets:    pruneTargets,
	}
	for _, task := range tasks {
		state.pendingByTarget[task.TargetPath]++
		state.idsByTarget[task.TargetPath] = append(state.idsByTarget[task.TargetPath], taskIdentity(task.TargetPath, task.EntryKey))
		existing := state.sourceByTarget[task.TargetPath]
		if existing != "" && existing != task.SourcePath {
			return nil, fmt.Errorf("output staging conflict: %s has conflicting source paths", task.TargetPath)
		}
		state.sourceByTarget[task.TargetPath] = task.SourcePath
		existingLocale := state.localeByTarget[task.TargetPath]
		if existingLocale != "" && existingLocale != task.TargetLocale {
			return nil, fmt.Errorf("output staging conflict: %s has conflicting target locales", task.TargetPath)
		}
		state.localeByTarget[task.TargetPath] = task.TargetLocale
	}
	return state, nil
}

func (s *Service) executePool(ctx context.Context, tasks []Task, lockPath string, lockState *lockfile.File, workers int, pruneTargets map[string]map[string]struct{}, emitter *eventEmitter) (map[string]stagedOutput, map[string]struct{}, executionReport, error) {
	state, err := newExecutorState(tasks, pruneTargets)
	if err != nil {
		return nil, nil, executionReport{}, err
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
	fatalLockErr := make(chan error, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	lockWriterDone := make(chan struct{})
	go s.runLockWriter(ctx, completions, lockWriterDone, lockState, lockPath, fatalLockErr, cancel, state, emitter)

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go s.runWorker(ctx, jobs, completions, state, emitter, &wg, cancel)
	}

	go s.feedJobs(ctx, jobs, tasks)

	wg.Wait()
	close(completions)
	<-lockWriterDone

	select {
	case err := <-fatalLockErr:
		return nil, nil, state.report, err
	default:
	}

	return state.staged, state.flushedTargets, state.report, nil
}

func (s *Service) runLockWriter(ctx context.Context, completions <-chan taskCompletion, done chan<- struct{}, lockState *lockfile.File, lockPath string, fatalLockErr chan<- error, cancel context.CancelFunc, state *executorState, emitter *eventEmitter) {
	defer close(done)
	for {
		select {
		case <-ctx.Done():
			return
		case completion, ok := <-completions:
			if !ok {
				return
			}
			lockState.RunCompleted[completion.identity] = lockfile.RunCompletion{CompletedAt: s.now(), SourceHash: completion.sourceHash}
			if err := s.saveLock(lockPath, *lockState); err != nil {
				select {
				case fatalLockErr <- fmt.Errorf("persist lock state: %w", err):
				default:
				}
				cancel()
				return
			}

			state.reportMu.Lock()
			state.report.PersistedToLock++
			persisted := state.report.PersistedToLock
			succeeded := state.report.Succeeded
			failed := state.report.Failed
			state.reportMu.Unlock()
			emitter.emit(Event{Kind: EventPersisted, PersistedToLock: persisted, Succeeded: succeeded, Failed: failed})

			if err := s.flushIfTargetCompleted(completion.targetPath, completion.sourcePath, state); err != nil {
				recordTaskFailure(&state.report, &state.reportMu, state.total, Task{TargetPath: completion.targetPath}, err, emitter)
				if rollbackErr := s.rollbackLockForTarget(lockState, lockPath, completion.targetPath, state, emitter); rollbackErr != nil {
					select {
					case fatalLockErr <- rollbackErr:
					default:
					}
					cancel()
					return
				}
				continue
			}
		}
	}
}

func (s *Service) runWorker(ctx context.Context, jobs <-chan Task, completions chan<- taskCompletion, state *executorState, emitter *eventEmitter, wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-jobs:
			if !ok {
				return
			}
			if s.processTask(ctx, task, completions, state, emitter) {
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
		output = stagedOutput{entries: map[string]string{}, sourcePath: expectedSourcePath, targetLocale: expectedTargetLocale}
	} else if output.sourcePath == "" {
		output.sourcePath = expectedSourcePath
	}
	if output.targetLocale == "" {
		output.targetLocale = expectedTargetLocale
	}

	return s.flushOutputForTarget(targetPath, output, state.pruneTargets[targetPath])
}

func (s *Service) processTask(ctx context.Context, task Task, completions chan<- taskCompletion, state *executorState, emitter *eventEmitter) bool {
	translated, err := s.translate(ctx, translator.Request{Source: task.SourceText, TargetLanguage: task.TargetLocale, Context: task.EntryKey, ModelProvider: task.Provider, Model: task.Model, Prompt: task.Prompt})
	if err != nil {
		recordTaskFailure(&state.report, &state.reportMu, state.total, task, err, emitter)
		markTargetFailed(task.TargetPath, &state.pendingMu, state.failedTargets)
		return false
	}
	if err := stageTaskOutput(state.staged, task.TargetPath, task.SourcePath, task.TargetLocale, task.EntryKey, translated, &state.stageMu); err != nil {
		recordTaskFailure(&state.report, &state.reportMu, state.total, task, err, emitter)
		markTargetFailed(task.TargetPath, &state.pendingMu, state.failedTargets)
		return false
	}

	select {
	case completions <- taskCompletion{identity: taskIdentity(task.TargetPath, task.EntryKey), sourceHash: hashSourceText(task.SourceText), targetPath: task.TargetPath, sourcePath: task.SourcePath}:
		state.reportMu.Lock()
		state.report.Succeeded++
		succeeded := state.report.Succeeded
		failed := state.report.Failed
		state.reportMu.Unlock()
		emitter.emit(Event{Kind: EventTaskDone, TaskSucceeded: true, TargetPath: task.TargetPath, EntryKey: task.EntryKey, Succeeded: succeeded, Failed: failed, ExecutableTotal: state.total})
		return true
	case <-ctx.Done():
		return false
	}
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
	reportMu.Unlock()
	emitter.emit(Event{Kind: EventTaskDone, TaskSucceeded: false, TargetPath: task.TargetPath, EntryKey: task.EntryKey, FailureReason: err.Error(), Succeeded: succeeded, Failed: failed, ExecutableTotal: total})
}

func stageTaskOutput(staged map[string]stagedOutput, targetPath, sourcePath, targetLocale, entryKey, value string, stageMu *sync.Mutex) error {
	stageMu.Lock()
	defer stageMu.Unlock()

	bucket, ok := staged[targetPath]
	if !ok {
		bucket = stagedOutput{entries: map[string]string{}, sourcePath: sourcePath, targetLocale: targetLocale}
		staged[targetPath] = bucket
	} else if bucket.sourcePath != sourcePath {
		return fmt.Errorf("output staging conflict: %s has conflicting source paths", targetPath)
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

func markTargetFailed(targetPath string, mu *sync.Mutex, failedTargets map[string]struct{}) {
	mu.Lock()
	failedTargets[targetPath] = struct{}{}
	mu.Unlock()
}

func isTargetFailed(targetPath string, mu *sync.Mutex, failedTargets map[string]struct{}) bool {
	mu.Lock()
	_, failed := failedTargets[targetPath]
	mu.Unlock()
	return failed
}

func (s *Service) rollbackLockForTarget(lockState *lockfile.File, lockPath, targetPath string, state *executorState, emitter *eventEmitter) error {
	ids := state.idsByTarget[targetPath]
	if len(ids) == 0 {
		return nil
	}

	removed := 0
	for _, id := range ids {
		if _, ok := lockState.RunCompleted[id]; ok {
			delete(lockState.RunCompleted, id)
			removed++
		}
	}
	if removed == 0 {
		return nil
	}
	if err := s.saveLock(lockPath, *lockState); err != nil {
		return fmt.Errorf("persist lock rollback for %q: %w", targetPath, err)
	}

	state.reportMu.Lock()
	state.report.PersistedToLock -= removed
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
