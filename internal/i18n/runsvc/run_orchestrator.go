package runsvc

import (
	"context"
	"fmt"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/lockfile"
)

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
	initializeLockState(state)

	report, executable := applyLockFilter(planned, state.RunCompleted)
	emitter.emit(Event{Kind: EventPlanned, PlannedTotal: report.PlannedTotal, SkippedByLock: report.SkippedByLock, ExecutableTotal: report.ExecutableTotal})

	pruneTargets, err := s.collectPruneTargets(in, planned, &report, emitter)
	if err != nil {
		return report, err
	}

	if in.DryRun || (len(executable) == 0 && len(report.PruneCandidates) == 0) {
		emitter.emit(completedEvent(report))
		return report, nil
	}

	emitter.emit(Event{Kind: EventPhase, Phase: PhaseExecuting})
	staged, flushedTargets, execReport, err := s.executePool(ctx, executable, in.LockPath, state, in.Workers, pruneTargets, emitter)
	report.Succeeded = execReport.Succeeded
	report.Failed = execReport.Failed
	report.PersistedToLock = execReport.PersistedToLock
	report.Failures = append(report.Failures, execReport.Failures...)
	if err != nil {
		emitter.emit(completedEvent(report))
		return report, err
	}

	emitter.emit(Event{Kind: EventPhase, Phase: PhaseFinalizingOutput})
	if err := s.flushOutputs(staged, remainingPruneTargets(pruneTargets, flushedTargets)); err != nil {
		emitter.emit(completedEvent(report))
		return report, err
	}

	report.PruneApplied = len(report.PruneCandidates)
	emitter.emit(completedEvent(report))
	return report, nil
}

func initializeLockState(state *lockfile.File) {
	if state.RunCompleted == nil {
		state.RunCompleted = map[string]lockfile.RunCompletion{}
	}
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

func (s *Service) collectPruneTargets(in Input, planned []Task, report *Report, emitter *eventEmitter) (map[string]map[string]struct{}, error) {
	pruneTargets := map[string]map[string]struct{}{}
	if !in.Prune {
		return pruneTargets, nil
	}

	emitter.emit(Event{Kind: EventPhase, Phase: PhaseScanningPrune})
	pruneTargets = buildPlannedTargetKeySet(planned)
	candidates, err := s.planPruneCandidates(pruneTargets)
	if err != nil {
		return nil, err
	}
	report.PruneCandidates = candidates
	if err := validatePruneLimit(in, len(report.PruneCandidates)); err != nil {
		return nil, err
	}
	return pruneTargets, nil
}

func remainingPruneTargets(pruneTargets map[string]map[string]struct{}, flushedTargets map[string]struct{}) map[string]map[string]struct{} {
	remaining := map[string]map[string]struct{}{}
	for path, keep := range pruneTargets {
		if _, alreadyFlushed := flushedTargets[path]; alreadyFlushed {
			continue
		}
		remaining[path] = keep
	}
	return remaining
}

func completedEvent(report Report) Event {
	return Event{
		Kind:            EventCompleted,
		PlannedTotal:    report.PlannedTotal,
		SkippedByLock:   report.SkippedByLock,
		ExecutableTotal: report.ExecutableTotal,
		Succeeded:       report.Succeeded,
		Failed:          report.Failed,
		PersistedToLock: report.PersistedToLock,
		PruneCandidates: len(report.PruneCandidates),
		PruneApplied:    report.PruneApplied,
	}
}
