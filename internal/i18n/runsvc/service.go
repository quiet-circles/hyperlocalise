package runsvc

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/lockfile"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/pathresolver"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translationfileparser"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

const (
	tokenSource = "{{source}}"
	tokenTarget = "{{target}}"
	tokenInput  = "{{input}}"
)

type Input struct {
	ConfigPath string
	DryRun     bool
	Prune      bool
	PruneLimit int
	PruneForce bool
	LockPath   string
	Workers    int
	OnEvent    func(Event)
}

const defaultPruneLimit = 100

type EventKind string

const (
	EventPhase     EventKind = "phase"
	EventPlanned   EventKind = "planned"
	EventTaskDone  EventKind = "task_done"
	EventPersisted EventKind = "persisted"
	EventCompleted EventKind = "completed"
)

const (
	PhasePlanning         = "planning"
	PhaseScanningPrune    = "scanning_prune"
	PhaseExecuting        = "executing"
	PhaseFinalizingOutput = "finalizing_output"
)

type Event struct {
	Kind            EventKind `json:"kind"`
	Phase           string    `json:"phase,omitempty"`
	PlannedTotal    int       `json:"plannedTotal,omitempty"`
	SkippedByLock   int       `json:"skippedByLock,omitempty"`
	ExecutableTotal int       `json:"executableTotal,omitempty"`
	Succeeded       int       `json:"succeeded,omitempty"`
	Failed          int       `json:"failed,omitempty"`
	PersistedToLock int       `json:"persistedToLock,omitempty"`
	PruneCandidates int       `json:"pruneCandidates,omitempty"`
	PruneApplied    int       `json:"pruneApplied,omitempty"`
	TaskSucceeded   bool      `json:"taskSucceeded,omitempty"`
	TargetPath      string    `json:"targetPath,omitempty"`
	EntryKey        string    `json:"entryKey,omitempty"`
	FailureReason   string    `json:"failureReason,omitempty"`
}

type Task struct {
	SourceLocale string `json:"sourceLocale"`
	TargetLocale string `json:"targetLocale"`
	SourcePath   string `json:"sourcePath"`
	TargetPath   string `json:"targetPath"`
	EntryKey     string `json:"entryKey"`
	SourceText   string `json:"sourceText"`
	ProfileName  string `json:"profileName"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
}

type Failure struct {
	TargetPath string `json:"targetPath"`
	EntryKey   string `json:"entryKey"`
	Reason     string `json:"reason"`
}

type Report struct {
	PlannedTotal    int              `json:"plannedTotal"`
	SkippedByLock   int              `json:"skippedByLock"`
	ExecutableTotal int              `json:"executableTotal"`
	Succeeded       int              `json:"succeeded"`
	Failed          int              `json:"failed"`
	PersistedToLock int              `json:"persistedToLock"`
	Failures        []Failure        `json:"failures,omitempty"`
	Executable      []Task           `json:"executable,omitempty"`
	Skipped         []Task           `json:"skipped,omitempty"`
	PruneCandidates []PruneCandidate `json:"pruneCandidates,omitempty"`
	PruneApplied    int              `json:"pruneApplied"`
}

type PruneCandidate struct {
	TargetPath string `json:"targetPath"`
	EntryKey   string `json:"entryKey"`
}

type Service struct {
	loadConfig func(path string) (*config.I18NConfig, error)
	loadLock   func(path string) (*lockfile.File, error)
	saveLock   func(path string, f lockfile.File) error
	readFile   func(path string) ([]byte, error)
	writeFile  func(path string, content []byte) error
	translate  func(ctx context.Context, req translator.Request) (string, error)
	newParser  func() *translationfileparser.Strategy
	now        func() time.Time
	numCPU     func() int
}

func New() *Service {
	return &Service{
		loadConfig: config.Load,
		loadLock:   lockfile.Load,
		saveLock:   lockfile.Save,
		readFile:   os.ReadFile,
		writeFile: func(path string, content []byte) error {
			return writeBytesAtomic(path, content)
		},
		translate: translator.Translate,
		newParser: translationfileparser.NewDefaultStrategy,
		now:       func() time.Time { return time.Now().UTC() },
		numCPU:    runtime.NumCPU,
	}
}

func Run(ctx context.Context, in Input) (Report, error) {
	return New().Run(ctx, in)
}

func (s *Service) Run(ctx context.Context, in Input) (Report, error) {
	emitter := newEventEmitter(in.OnEvent)
	emitter.emit(Event{Kind: EventPhase, Phase: PhasePlanning})

	cfg, err := s.loadConfig(in.ConfigPath)
	if err != nil {
		return Report{}, fmt.Errorf("load config: %w", err)
	}

	planned, err := s.planTasks(cfg)
	if err != nil {
		return Report{}, err
	}

	state, err := s.loadLock(in.LockPath)
	if err != nil {
		return Report{}, fmt.Errorf("load lock state: %w", err)
	}

	if state.RunCompleted == nil {
		state.RunCompleted = map[string]lockfile.RunCompletion{}
	}

	report, executable := applyLockFilter(planned, state.RunCompleted)
	emitter.emit(Event{
		Kind:            EventPlanned,
		PlannedTotal:    report.PlannedTotal,
		SkippedByLock:   report.SkippedByLock,
		ExecutableTotal: report.ExecutableTotal,
	})

	pruneTargets := map[string]map[string]struct{}{}
	if in.Prune {
		emitter.emit(Event{Kind: EventPhase, Phase: PhaseScanningPrune})
		pruneTargets = buildPlannedTargetKeySet(planned)
		report.PruneCandidates, err = s.planPruneCandidates(pruneTargets)
		if err != nil {
			return report, err
		}
		if err := validatePruneLimit(in, len(report.PruneCandidates)); err != nil {
			return report, err
		}
	}

	if in.DryRun {
		emitter.emit(Event{
			Kind:            EventCompleted,
			PlannedTotal:    report.PlannedTotal,
			SkippedByLock:   report.SkippedByLock,
			ExecutableTotal: report.ExecutableTotal,
			PruneCandidates: len(report.PruneCandidates),
		})
		return report, nil
	}
	if len(executable) == 0 && len(report.PruneCandidates) == 0 {
		emitter.emit(Event{
			Kind:            EventCompleted,
			PlannedTotal:    report.PlannedTotal,
			SkippedByLock:   report.SkippedByLock,
			ExecutableTotal: report.ExecutableTotal,
			PruneCandidates: len(report.PruneCandidates),
		})
		return report, nil
	}

	emitter.emit(Event{Kind: EventPhase, Phase: PhaseExecuting})
	staged, flushedTargets, execReport, err := s.executePool(ctx, executable, in.LockPath, state, in.Workers, pruneTargets, emitter)
	report.Succeeded = execReport.Succeeded
	report.Failed = execReport.Failed
	report.PersistedToLock = execReport.PersistedToLock
	report.Failures = append(report.Failures, execReport.Failures...)
	if err != nil {
		emitter.emit(Event{
			Kind:            EventCompleted,
			PlannedTotal:    report.PlannedTotal,
			SkippedByLock:   report.SkippedByLock,
			ExecutableTotal: report.ExecutableTotal,
			Succeeded:       report.Succeeded,
			Failed:          report.Failed,
			PersistedToLock: report.PersistedToLock,
			PruneCandidates: len(report.PruneCandidates),
		})
		return report, err
	}

	emitter.emit(Event{Kind: EventPhase, Phase: PhaseFinalizingOutput})
	remainingPruneTargets := map[string]map[string]struct{}{}
	for path, keep := range pruneTargets {
		if _, alreadyFlushed := flushedTargets[path]; alreadyFlushed {
			continue
		}
		remainingPruneTargets[path] = keep
	}

	if err := s.flushOutputs(staged, remainingPruneTargets); err != nil {
		emitter.emit(Event{
			Kind:            EventCompleted,
			PlannedTotal:    report.PlannedTotal,
			SkippedByLock:   report.SkippedByLock,
			ExecutableTotal: report.ExecutableTotal,
			Succeeded:       report.Succeeded,
			Failed:          report.Failed,
			PersistedToLock: report.PersistedToLock,
			PruneCandidates: len(report.PruneCandidates),
		})
		return report, err
	}
	report.PruneApplied = len(report.PruneCandidates)
	emitter.emit(Event{
		Kind:            EventCompleted,
		PlannedTotal:    report.PlannedTotal,
		SkippedByLock:   report.SkippedByLock,
		ExecutableTotal: report.ExecutableTotal,
		Succeeded:       report.Succeeded,
		Failed:          report.Failed,
		PersistedToLock: report.PersistedToLock,
		PruneCandidates: len(report.PruneCandidates),
		PruneApplied:    report.PruneApplied,
	})

	return report, nil
}

func (s *Service) planTasks(cfg *config.I18NConfig) ([]Task, error) {
	parser := s.newParser()
	groups := sortedGroupNames(cfg.Groups)
	buckets := sortedBucketNames(cfg.Buckets)

	tasks := make([]Task, 0)

	for _, groupName := range groups {
		group := cfg.Groups[groupName]
		profileName, profile, err := resolveProfile(cfg, groupName)
		if err != nil {
			return nil, err
		}

		targets := group.Targets
		if len(targets) == 0 {
			targets = append([]string(nil), cfg.Locales.Targets...)
		}
		slices.Sort(targets)

		selectedBuckets := group.Buckets
		if len(selectedBuckets) == 0 {
			selectedBuckets = append([]string(nil), buckets...)
		}

		for _, bucketName := range selectedBuckets {
			bucket, ok := cfg.Buckets[bucketName]
			if !ok {
				return nil, fmt.Errorf("planning tasks: group %q references unknown bucket %q", groupName, bucketName)
			}

			for _, file := range bucket.Files {
				sourcePattern := pathresolver.ResolveSourcePath(file.From, cfg.Locales.Source)
				sources, err := resolveSourcePaths(sourcePattern)
				if err != nil {
					return nil, fmt.Errorf("planning tasks: resolve source paths for %q: %w", sourcePattern, err)
				}
				if len(sources) == 0 {
					return nil, fmt.Errorf("planning tasks: source pattern %q matched no files", sourcePattern)
				}

				for _, sourcePath := range sources {
					if shouldIgnoreSourcePath(sourcePath, cfg.Locales.Targets) {
						continue
					}
					sourceEntries, err := s.loadSourceEntries(parser, sourcePath)
					if err != nil {
						return nil, err
					}
					keys := sortedEntryKeys(sourceEntries)
					for _, target := range targets {
						resolvedTargetPattern := pathresolver.ResolveTargetPath(file.To, cfg.Locales.Source, target)
						targetPath, err := resolveTargetPath(sourcePattern, resolvedTargetPattern, sourcePath)
						if err != nil {
							return nil, fmt.Errorf("planning tasks: resolve target path for source %q: %w", sourcePath, err)
						}
						for _, key := range keys {
							sourceText := sourceEntries[key]
							tasks = append(tasks, Task{
								SourceLocale: cfg.Locales.Source,
								TargetLocale: target,
								SourcePath:   sourcePath,
								TargetPath:   targetPath,
								EntryKey:     key,
								SourceText:   sourceText,
								ProfileName:  profileName,
								Provider:     profile.Provider,
								Model:        profile.Model,
								Prompt:       renderPrompt(profile.Prompt, cfg.Locales.Source, target, sourceText),
							})
						}
					}
				}
			}
		}
	}

	return tasks, nil
}

func (s *Service) loadSourceEntries(parser *translationfileparser.Strategy, sourcePath string) (map[string]string, error) {
	content, err := s.readFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("planning tasks: source file %q does not exist", sourcePath)
		}
		return nil, fmt.Errorf("planning tasks: read source file %q: %w", sourcePath, err)
	}

	entries, err := parser.Parse(sourcePath, content)
	if err != nil {
		return nil, fmt.Errorf("planning tasks: parse source file %q: %w", sourcePath, err)
	}

	return entries, nil
}

func applyLockFilter(planned []Task, completed map[string]lockfile.RunCompletion) (Report, []Task) {
	report := Report{PlannedTotal: len(planned)}

	executable := make([]Task, 0, len(planned))
	for _, task := range planned {
		identity := taskIdentity(task.TargetPath, task.EntryKey)
		sourceHash := hashSourceText(task.SourceText)
		if c, ok := completed[identity]; ok && c.SourceHash == sourceHash {
			report.SkippedByLock++
			report.Skipped = append(report.Skipped, task)
			continue
		}

		report.Executable = append(report.Executable, task)
		executable = append(executable, task)
	}

	report.ExecutableTotal = len(executable)
	return report, executable
}

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
	entries    map[string]string
	sourcePath string
}

type eventEmitter struct {
	notify func(Event)
	mu     sync.Mutex
}

func newEventEmitter(onEvent func(Event)) *eventEmitter {
	if onEvent == nil {
		return nil
	}

	return &eventEmitter{notify: onEvent}
}

func (e *eventEmitter) emit(ev Event) {
	if e == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.notify(ev)
}

func (s *Service) executePool(
	ctx context.Context,
	tasks []Task,
	lockPath string,
	state *lockfile.File,
	workers int,
	pruneTargets map[string]map[string]struct{},
	emitter *eventEmitter,
) (map[string]stagedOutput, map[string]struct{}, executionReport, error) {
	report := executionReport{}
	staged := map[string]stagedOutput{}
	flushedTargets := map[string]struct{}{}
	pendingByTarget := map[string]int{}
	sourceByTarget := map[string]string{}
	for _, task := range tasks {
		pendingByTarget[task.TargetPath]++
		existing := sourceByTarget[task.TargetPath]
		if existing != "" && existing != task.SourcePath {
			return nil, nil, report, fmt.Errorf("output staging conflict: %s has conflicting source paths", task.TargetPath)
		}
		sourceByTarget[task.TargetPath] = task.SourcePath
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

	var (
		reportMu  sync.Mutex
		stageMu   sync.Mutex
		pendingMu sync.Mutex
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	lockWriterDone := make(chan struct{})
	go s.runLockWriter(
		completions,
		lockWriterDone,
		state,
		lockPath,
		fatalLockErr,
		cancel,
		staged,
		&stageMu,
		pendingByTarget,
		sourceByTarget,
		flushedTargets,
		pruneTargets,
		&pendingMu,
		&report,
		&reportMu,
		len(tasks),
		emitter,
	)

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go s.runWorker(ctx, jobs, completions, staged, &stageMu, pendingByTarget, sourceByTarget, flushedTargets, pruneTargets, &pendingMu, &report, &reportMu, len(tasks), emitter, &wg, cancel)
	}

	go s.feedJobs(ctx, jobs, tasks)

	wg.Wait()
	close(completions)
	<-lockWriterDone

	select {
	case err := <-fatalLockErr:
		return nil, nil, report, err
	default:
	}

	return staged, flushedTargets, report, nil
}

func (s *Service) runLockWriter(
	completions <-chan taskCompletion,
	lockWriterDone chan<- struct{},
	state *lockfile.File,
	lockPath string,
	fatalLockErr chan<- error,
	cancel context.CancelFunc,
	staged map[string]stagedOutput,
	stageMu *sync.Mutex,
	pendingByTarget map[string]int,
	sourceByTarget map[string]string,
	flushedTargets map[string]struct{},
	pruneTargets map[string]map[string]struct{},
	pendingMu *sync.Mutex,
	report *executionReport,
	reportMu *sync.Mutex,
	total int,
	emitter *eventEmitter,
) {
	defer close(lockWriterDone)
	for completion := range completions {
		state.RunCompleted[completion.identity] = lockfile.RunCompletion{CompletedAt: s.now(), SourceHash: completion.sourceHash}
		if err := s.saveLock(lockPath, *state); err != nil {
			select {
			case fatalLockErr <- fmt.Errorf("persist lock state: %w", err):
			default:
			}
			cancel()
			return
		}
		reportMu.Lock()
		report.PersistedToLock++
		persisted := report.PersistedToLock
		succeeded := report.Succeeded
		failed := report.Failed
		reportMu.Unlock()
		emitter.emit(Event{
			Kind:            EventPersisted,
			PersistedToLock: persisted,
			Succeeded:       succeeded,
			Failed:          failed,
		})
		if err := s.flushIfTargetCompleted(completion.targetPath, completion.sourcePath, staged, stageMu, pendingByTarget, sourceByTarget, flushedTargets, pruneTargets, pendingMu); err != nil {
			recordTaskFailure(report, reportMu, total, Task{TargetPath: completion.targetPath}, err, emitter)
			select {
			case fatalLockErr <- err:
			default:
			}
			cancel()
			return
		}
	}
}

func (s *Service) runWorker(
	ctx context.Context,
	jobs <-chan Task,
	completions chan<- taskCompletion,
	staged map[string]stagedOutput,
	stageMu *sync.Mutex,
	pendingByTarget map[string]int,
	sourceByTarget map[string]string,
	flushedTargets map[string]struct{},
	pruneTargets map[string]map[string]struct{},
	pendingMu *sync.Mutex,
	report *executionReport,
	reportMu *sync.Mutex,
	total int,
	emitter *eventEmitter,
	wg *sync.WaitGroup,
	cancel context.CancelFunc,
) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-jobs:
			if !ok {
				return
			}
			if s.processTask(ctx, task, staged, stageMu, completions, report, reportMu, total, emitter) {
				continue
			}
			if err := s.flushIfTargetCompleted(task.TargetPath, task.SourcePath, staged, stageMu, pendingByTarget, sourceByTarget, flushedTargets, pruneTargets, pendingMu); err != nil {
				recordTaskFailure(report, reportMu, total, task, err, emitter)
				cancel()
				return
			}
		}
	}
}

func (s *Service) flushIfTargetCompleted(
	targetPath string,
	sourcePath string,
	staged map[string]stagedOutput,
	stageMu *sync.Mutex,
	pendingByTarget map[string]int,
	sourceByTarget map[string]string,
	flushedTargets map[string]struct{},
	pruneTargets map[string]map[string]struct{},
	pendingMu *sync.Mutex,
) error {
	shouldFlush := false
	expectedSourcePath := sourcePath
	pendingMu.Lock()
	remaining := pendingByTarget[targetPath]
	if remaining > 0 {
		remaining--
		pendingByTarget[targetPath] = remaining
	}
	if remaining == 0 {
		if _, done := flushedTargets[targetPath]; !done {
			shouldFlush = true
			flushedTargets[targetPath] = struct{}{}
			if knownSourcePath := sourceByTarget[targetPath]; knownSourcePath != "" {
				expectedSourcePath = knownSourcePath
			}
		}
	}
	pendingMu.Unlock()

	if !shouldFlush {
		return nil
	}

	stageMu.Lock()
	output, ok := staged[targetPath]
	if ok {
		delete(staged, targetPath)
	}
	stageMu.Unlock()

	if !ok {
		output = stagedOutput{
			entries:    map[string]string{},
			sourcePath: expectedSourcePath,
		}
	} else if output.sourcePath == "" {
		output.sourcePath = expectedSourcePath
	}

	return s.flushOutputForTarget(targetPath, output, pruneTargets[targetPath])
}

func (s *Service) processTask(
	ctx context.Context,
	task Task,
	staged map[string]stagedOutput,
	stageMu *sync.Mutex,
	completions chan<- taskCompletion,
	report *executionReport,
	reportMu *sync.Mutex,
	total int,
	emitter *eventEmitter,
) bool {
	translated, err := s.translate(ctx, translator.Request{
		Source:         task.SourceText,
		TargetLanguage: task.TargetLocale,
		Context:        task.EntryKey,
		ModelProvider:  task.Provider,
		Model:          task.Model,
		Prompt:         task.Prompt,
	})
	if err != nil {
		recordTaskFailure(report, reportMu, total, task, err, emitter)
		return false
	}

	if err := stageTaskOutput(staged, task.TargetPath, task.SourcePath, task.EntryKey, translated, stageMu); err != nil {
		recordTaskFailure(report, reportMu, total, task, err, emitter)
		return false
	}

	select {
	case completions <- taskCompletion{
		identity:   taskIdentity(task.TargetPath, task.EntryKey),
		sourceHash: hashSourceText(task.SourceText),
		targetPath: task.TargetPath,
		sourcePath: task.SourcePath,
	}:
		reportMu.Lock()
		report.Succeeded++
		succeeded := report.Succeeded
		failed := report.Failed
		reportMu.Unlock()
		emitter.emit(Event{
			Kind:            EventTaskDone,
			TaskSucceeded:   true,
			TargetPath:      task.TargetPath,
			EntryKey:        task.EntryKey,
			Succeeded:       succeeded,
			Failed:          failed,
			ExecutableTotal: total,
		})
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

	emitter.emit(Event{
		Kind:            EventTaskDone,
		TaskSucceeded:   false,
		TargetPath:      task.TargetPath,
		EntryKey:        task.EntryKey,
		FailureReason:   err.Error(),
		Succeeded:       succeeded,
		Failed:          failed,
		ExecutableTotal: total,
	})
}

func stageTaskOutput(staged map[string]stagedOutput, targetPath, sourcePath, entryKey, value string, stageMu *sync.Mutex) error {
	stageMu.Lock()
	defer stageMu.Unlock()

	bucket, ok := staged[targetPath]
	if !ok {
		bucket = stagedOutput{entries: map[string]string{}, sourcePath: sourcePath}
		staged[targetPath] = bucket
	} else if bucket.sourcePath != sourcePath {
		return fmt.Errorf("output staging conflict: %s has conflicting source paths", targetPath)
	}

	if existing, exists := bucket.entries[entryKey]; exists && existing != value {
		return fmt.Errorf("output staging conflict: %s already staged with different value", taskIdentity(targetPath, entryKey))
	}

	bucket.entries[entryKey] = value
	staged[targetPath] = bucket
	return nil
}

func (s *Service) flushOutputs(staged map[string]stagedOutput, pruneTargets map[string]map[string]struct{}) error {
	targetPaths := make([]string, 0, len(staged))
	for path := range staged {
		targetPaths = append(targetPaths, path)
	}
	for path := range pruneTargets {
		if _, ok := staged[path]; ok {
			continue
		}
		targetPaths = append(targetPaths, path)
	}
	slices.Sort(targetPaths)
	targetPaths = slices.Compact(targetPaths)

	for _, targetPath := range targetPaths {
		output := staged[targetPath]
		if err := s.flushOutputForTarget(targetPath, output, pruneTargets[targetPath]); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) flushOutputForTarget(targetPath string, output stagedOutput, keep map[string]struct{}) error {
	values, err := s.loadExistingTarget(targetPath)
	if err != nil {
		return err
	}

	if keep != nil {
		for key := range values {
			if _, ok := keep[key]; ok {
				continue
			}
			delete(values, key)
		}
	}

	maps.Copy(values, output.entries)

	content, err := s.marshalTargetFile(targetPath, output.sourcePath, values)
	if err != nil {
		return err
	}
	if err := s.writeFile(targetPath, content); err != nil {
		return fmt.Errorf("flush outputs: write %q: %w", targetPath, err)
	}
	return nil
}

func buildPlannedTargetKeySet(planned []Task) map[string]map[string]struct{} {
	keep := map[string]map[string]struct{}{}
	for _, task := range planned {
		bucket := keep[task.TargetPath]
		if bucket == nil {
			bucket = map[string]struct{}{}
			keep[task.TargetPath] = bucket
		}
		bucket[task.EntryKey] = struct{}{}
	}
	return keep
}

func (s *Service) planPruneCandidates(pruneTargets map[string]map[string]struct{}) ([]PruneCandidate, error) {
	candidates := make([]PruneCandidate, 0)
	targetPaths := make([]string, 0, len(pruneTargets))
	for path := range pruneTargets {
		targetPaths = append(targetPaths, path)
	}
	slices.Sort(targetPaths)

	for _, targetPath := range targetPaths {
		existing, err := s.loadExistingTarget(targetPath)
		if err != nil {
			return nil, err
		}
		keys := sortedEntryKeys(existing)
		for _, key := range keys {
			if _, ok := pruneTargets[targetPath][key]; ok {
				continue
			}
			candidates = append(candidates, PruneCandidate{TargetPath: targetPath, EntryKey: key})
		}
	}

	return candidates, nil
}

func validatePruneLimit(in Input, candidates int) error {
	if !in.Prune || in.DryRun || in.PruneForce {
		return nil
	}
	limit := in.PruneLimit
	if limit <= 0 {
		limit = defaultPruneLimit
	}
	if candidates <= limit {
		return nil
	}
	return fmt.Errorf("prune safety limit exceeded: %d keys scheduled for deletion (limit %d). rerun with --prune-max-deletions %d or --prune-force", candidates, limit, candidates)
}

func (s *Service) loadExistingTarget(path string) (map[string]string, error) {
	content, err := s.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("flush outputs: read target file %q: %w", path, err)
	}

	entries, err := s.newParser().Parse(path, content)
	if err != nil {
		return nil, fmt.Errorf("flush outputs: parse target file %q: %w", path, err)
	}

	return entries, nil
}

func (s *Service) marshalTargetFile(path, sourcePath string, values map[string]string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".md" || ext == ".mdx" {
		template, err := s.loadTemplateFallback(path, sourcePath)
		if err != nil {
			return nil, err
		}
		return translationfileparser.MarshalMarkdown(template, values), nil
	}

	if ext == ".strings" {
		template, err := s.loadTemplateFallback(path, sourcePath)
		if err != nil {
			return nil, err
		}
		content, err := translationfileparser.MarshalAppleStrings(template, values)
		if err != nil {
			return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
		}
		return content, nil
	}

	if ext == ".stringsdict" {
		template, err := s.loadTemplateFallback(path, sourcePath)
		if err != nil {
			return nil, err
		}
		content, err := translationfileparser.MarshalAppleStringsdict(template, values)
		if err != nil {
			return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
		}
		return content, nil
	}

	if ext == ".csv" {
		template, err := s.loadTemplateFallback(path, sourcePath)
		if err != nil {
			return nil, err
		}
		content, err := translationfileparser.MarshalCSV(template, values, translationfileparser.CSVParser{})
		if err != nil {
			return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
		}
		return content, nil
	}

	if ext != ".json" {
		return nil, fmt.Errorf("flush outputs: unsupported target file extension %q for %q", ext, path)
	}

	payload := map[string]any{}
	keys := sortedEntryKeys(values)
	for _, key := range keys {
		setNestedValue(payload, key, values[key])
	}

	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
	}
	return append(content, '\n'), nil
}

func (s *Service) loadTemplateFallback(targetPath, sourcePath string) ([]byte, error) {
	content, err := s.readFile(targetPath)
	if err == nil {
		return content, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("flush outputs: read target file %q: %w", targetPath, err)
	}

	template, srcErr := s.readFile(sourcePath)
	if srcErr != nil {
		return nil, fmt.Errorf("flush outputs: read template source %q: %w", sourcePath, srcErr)
	}
	return template, nil
}

func setNestedValue(payload map[string]any, dottedKey, value string) {
	parts := strings.Split(dottedKey, ".")
	current := payload
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}

		next, ok := current[part]
		if !ok {
			nested := map[string]any{}
			current[part] = nested
			current = nested
			continue
		}

		nested, ok := next.(map[string]any)
		if !ok {
			nested = map[string]any{}
			current[part] = nested
		}
		current = nested
	}
}

func writeBytesAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return nil
}

func resolveProfile(cfg *config.I18NConfig, groupName string) (string, config.LLMProfile, error) {
	bestPriority := -1
	bestProfile := ""

	for _, rule := range cfg.LLM.Rules {
		if rule.Group != groupName {
			continue
		}
		if rule.Priority > bestPriority {
			bestPriority = rule.Priority
			bestProfile = rule.Profile
		}
	}

	if strings.TrimSpace(bestProfile) == "" {
		bestProfile = "default"
	}

	profile, ok := cfg.LLM.Profiles[bestProfile]
	if !ok {
		return "", config.LLMProfile{}, fmt.Errorf("planning tasks: unresolvable profile %q for group %q", bestProfile, groupName)
	}

	return bestProfile, profile, nil
}

func sortedGroupNames(groups map[string]config.GroupConfig) []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func sortedBucketNames(buckets map[string]config.BucketConfig) []string {
	names := make([]string, 0, len(buckets))
	for name := range buckets {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func sortedEntryKeys(entries map[string]string) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func shouldIgnoreSourcePath(sourcePath string, targetLocales []string) bool {
	normalized := filepath.ToSlash(sourcePath)
	segments := strings.Split(normalized, "/")
	if len(segments) < 2 {
		return false
	}

	targets := make(map[string]struct{}, len(targetLocales))
	for _, locale := range targetLocales {
		targets[locale] = struct{}{}
	}

	for i := 1; i < len(segments)-1; i++ {
		if _, ok := targets[segments[i]]; ok {
			return true
		}
	}
	return false
}

func resolveSourcePaths(sourcePattern string) ([]string, error) {
	if !strings.ContainsAny(sourcePattern, "*?[") {
		return []string{sourcePattern}, nil
	}

	if !strings.Contains(sourcePattern, "**") {
		matches, err := filepath.Glob(sourcePattern)
		if err != nil {
			return nil, err
		}
		slices.Sort(matches)
		return matches, nil
	}

	normalizedPattern := filepath.ToSlash(sourcePattern)
	re, err := globToRegex(normalizedPattern)
	if err != nil {
		return nil, err
	}

	baseDir := baseDirForDoublestar(sourcePattern)
	matches := make([]string, 0)
	err = filepath.WalkDir(baseDir, func(candidate string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		normalizedCandidate := filepath.ToSlash(candidate)
		if re.MatchString(normalizedCandidate) {
			matches = append(matches, candidate)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.Sort(matches)
	return matches, nil
}

func resolveTargetPath(sourcePattern, targetPattern, sourcePath string) (string, error) {
	if !strings.ContainsAny(sourcePattern, "*?[") {
		return targetPattern, nil
	}
	if !strings.ContainsAny(targetPattern, "*?[") {
		return "", fmt.Errorf("target pattern %q must include glob tokens when source pattern %q includes globs", targetPattern, sourcePattern)
	}
	sourceBase := globBaseDir(sourcePattern)
	targetBase := globBaseDir(targetPattern)
	relative, err := filepath.Rel(sourceBase, sourcePath)
	if err != nil {
		return "", err
	}
	return filepath.Join(targetBase, relative), nil
}

func baseDirForDoublestar(pattern string) string {
	normalized := filepath.ToSlash(pattern)
	idx := strings.Index(normalized, "**")
	if idx == -1 {
		return filepath.Dir(pattern)
	}
	prefix := strings.TrimSuffix(normalized[:idx], "/")
	if prefix == "" {
		return "."
	}
	return filepath.FromSlash(prefix)
}

func globBaseDir(pattern string) string {
	idx := strings.IndexAny(filepath.ToSlash(pattern), "*?[")
	if idx == -1 {
		return filepath.Dir(pattern)
	}
	prefix := filepath.ToSlash(pattern)[:idx]
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return "."
	}
	return filepath.FromSlash(prefix)
}

func globToRegex(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					b.WriteString("(?:.*/)?")
					i += 3
					continue
				}
				b.WriteString(".*")
				i += 2
				continue
			}
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
		}
		i++
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

func renderPrompt(prompt, sourceLocale, targetLocale, sourceText string) string {
	rendered := strings.ReplaceAll(prompt, tokenSource, sourceLocale)
	rendered = strings.ReplaceAll(rendered, tokenTarget, targetLocale)
	rendered = strings.ReplaceAll(rendered, tokenInput, sourceText)
	return rendered
}

func taskIdentity(targetPath, entryKey string) string {
	return targetPath + "::" + entryKey
}

func hashSourceText(source string) string {
	sum := sha512.Sum512([]byte(source))
	return fmt.Sprintf("%x", sum)
}
