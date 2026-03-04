package runsvc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/lockfile"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translationfileparser"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

func TestRunUsesConfiguredWorkersWhenProvided(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B","c":"C"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.numCPU = func() int { return 1 }

	var mu sync.Mutex
	active := 0
	maxActive := 0
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()
		return strings.ToUpper(req.Source), nil
	}

	_, err := svc.Run(context.Background(), Input{Workers: 3})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if maxActive < 2 {
		t.Fatalf("expected parallel execution with explicit workers, max active=%d", maxActive)
	}
}

func TestRunDefaultsWorkersToNumCPUWhenUnset(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B","c":"C"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.numCPU = func() int { return 1 }

	var mu sync.Mutex
	active := 0
	maxActive := 0
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()
		return strings.ToUpper(req.Source), nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if maxActive != 1 {
		t.Fatalf("expected single worker from numCPU default, max active=%d", maxActive)
	}
}

func TestRunFailsWhenSourceFileMissing(t *testing.T) {
	svc := newTestService()
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig("/tmp/missing.json", "/tmp/out.json")
		return &cfg, nil
	}
	svc.readFile = func(_ string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	_, err := svc.Run(context.Background(), Input{})
	if err == nil {
		t.Fatalf("expected planning error")
	}
}

func TestRunFailsOnUnsupportedSourceFormat(t *testing.T) {
	svc := newTestService()
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig("/tmp/source.yaml", "/tmp/out.json")
		return &cfg, nil
	}
	svc.readFile = func(_ string) ([]byte, error) {
		return []byte("hello: world"), nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err == nil || !strings.Contains(err.Error(), "unsupported file extension") {
		t.Fatalf("expected unsupported extension error, got %v", err)
	}
}

func TestResolveProfileRulePriorityWithDefaultFallback(t *testing.T) {
	cfg := testConfig("/tmp/source.json", "/tmp/out.json")
	cfg.LLM.Profiles["fast"] = config.LLMProfile{Provider: "openai", Model: "fast-model", Prompt: "fast {{input}}"}
	cfg.LLM.Rules = []config.LLMRule{
		{Priority: 1, Group: "default", Profile: "default"},
		{Priority: 100, Group: "default", Profile: "fast"},
	}

	profileName, profile, err := resolveProfile(&cfg, "default")
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}
	if profileName != "fast" || profile.Model != "fast-model" {
		t.Fatalf("unexpected profile resolved: name=%s model=%s", profileName, profile.Model)
	}

	profileName, profile, err = resolveProfile(&cfg, "unknown")
	if err != nil {
		t.Fatalf("resolve fallback profile: %v", err)
	}
	if profileName != "default" || profile.Model != "gpt-4.1-mini" {
		t.Fatalf("expected default fallback profile, got name=%s model=%s", profileName, profile.Model)
	}
}

func TestRunAppliesLockFilterByTargetAndEntry(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{RunCompleted: map[string]lockfile.RunCompletion{taskIdentity(targetPath, "a"): {CompletedAt: time.Now(), SourceHash: hashSourceText("A")}}}, nil
	}

	report, err := svc.Run(context.Background(), Input{DryRun: true})
	if err != nil {
		t.Fatalf("run dry-run: %v", err)
	}

	if report.PlannedTotal != 2 || report.SkippedByLock != 1 || report.ExecutableTotal != 1 {
		t.Fatalf("unexpected plan totals: %+v", report)
	}
	if len(report.Executable) != 1 || report.Executable[0].EntryKey != "b" {
		t.Fatalf("unexpected executable tasks: %+v", report.Executable)
	}
}

func TestRunDoesNotSkipWhenSourceTextChanges(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello World"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{RunCompleted: map[string]lockfile.RunCompletion{taskIdentity(targetPath, "hello"): {CompletedAt: time.Now(), SourceHash: hashSourceText("Hello")}}}, nil
	}

	report, err := svc.Run(context.Background(), Input{DryRun: true})
	if err != nil {
		t.Fatalf("run dry-run: %v", err)
	}

	if report.SkippedByLock != 0 || report.ExecutableTotal != 1 {
		t.Fatalf("expected changed source to be executable, got %+v", report)
	}
}

func TestRunForceBypassesLockFilter(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{
			RunCompleted: map[string]lockfile.RunCompletion{
				taskIdentity(targetPath, "a"): {CompletedAt: time.Now(), SourceHash: hashSourceText("A")},
				taskIdentity(targetPath, "b"): {CompletedAt: time.Now(), SourceHash: hashSourceText("B")},
			},
		}, nil
	}

	report, err := svc.Run(context.Background(), Input{DryRun: true, Force: true})
	if err != nil {
		t.Fatalf("run dry-run force: %v", err)
	}

	if report.PlannedTotal != 2 || report.SkippedByLock != 0 || report.ExecutableTotal != 2 {
		t.Fatalf("expected force run to execute all tasks, got %+v", report)
	}
	if len(report.Skipped) != 0 {
		t.Fatalf("expected no skipped tasks with force, got %+v", report.Skipped)
	}
}

func TestRunDryRunSkipsWrites(t *testing.T) {
	writeCount := 0
	lockSaveCount := 0

	svc := newTestService()
	svc.writeFile = func(_ string, _ []byte) error {
		writeCount++
		return nil
	}
	svc.saveLock = func(_ string, _ lockfile.File) error {
		lockSaveCount++
		return nil
	}

	_, err := svc.Run(context.Background(), Input{DryRun: true})
	if err != nil {
		t.Fatalf("run dry-run: %v", err)
	}
	if writeCount != 0 {
		t.Fatalf("expected no writes in dry-run, got %d", writeCount)
	}
	if lockSaveCount != 0 {
		t.Fatalf("expected no lock writes in dry-run, got %d", lockSaveCount)
	}
}

func TestRunEmitsPlannedAndCompletedEventsInDryRun(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	events := make([]Event, 0, 4)
	report, err := svc.Run(context.Background(), Input{
		DryRun: true,
		OnEvent: func(event Event) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("run dry-run with events: %v", err)
	}
	if report.ExecutableTotal != 2 {
		t.Fatalf("unexpected executable total: %+v", report)
	}
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}
	if events[0].Kind != EventPhase || events[0].Phase != PhasePlanning {
		t.Fatalf("unexpected first event: %+v", events[0])
	}

	foundPlanned := false
	foundCompleted := false
	for _, event := range events {
		if event.Kind == EventPlanned {
			foundPlanned = true
			if event.PlannedTotal != 2 || event.ExecutableTotal != 2 || event.SkippedByLock != 0 {
				t.Fatalf("unexpected planned event payload: %+v", event)
			}
		}
		if event.Kind == EventCompleted {
			foundCompleted = true
		}
	}
	if !foundPlanned {
		t.Fatalf("expected planned event, got %+v", events)
	}
	if !foundCompleted {
		t.Fatalf("expected completed event, got %+v", events)
	}
}

func TestRunNoExecutableDoesNotEmitExecutingPhaseWithoutPruneCandidates(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{
			RunCompleted: map[string]lockfile.RunCompletion{
				taskIdentity(targetPath, "a"): {CompletedAt: time.Now(), SourceHash: hashSourceText("A")},
			},
		}, nil
	}
	svc.translate = func(_ context.Context, _ translator.Request) (string, error) {
		t.Fatalf("did not expect translation call with zero executable tasks")
		return "", nil
	}

	events := make([]Event, 0, 4)
	report, err := svc.Run(context.Background(), Input{
		OnEvent: func(event Event) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.ExecutableTotal != 0 {
		t.Fatalf("expected zero executable tasks, got %+v", report)
	}
	for _, event := range events {
		if event.Kind == EventPhase && event.Phase == PhaseExecuting {
			t.Fatalf("did not expect %q phase event, got %+v", PhaseExecuting, events)
		}
	}
}

func TestRunNoExecutableDoesNotEmitExecutingPhaseWithPruneCandidates(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello"}`), nil
		case targetPath:
			return []byte(`{"hello":"Bonjour","legacy":"Legacy"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{
			RunCompleted: map[string]lockfile.RunCompletion{
				taskIdentity(targetPath, "hello"): {CompletedAt: time.Now(), SourceHash: hashSourceText("Hello")},
			},
		}, nil
	}
	svc.translate = func(_ context.Context, _ translator.Request) (string, error) {
		t.Fatalf("did not expect translation call with zero executable tasks")
		return "", nil
	}

	events := make([]Event, 0, 8)
	report, err := svc.Run(context.Background(), Input{
		Prune: true,
		OnEvent: func(event Event) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.ExecutableTotal != 0 {
		t.Fatalf("expected zero executable tasks, got %+v", report)
	}
	if len(report.PruneCandidates) != 1 {
		t.Fatalf("expected one prune candidate, got %+v", report.PruneCandidates)
	}

	foundScanningPrune := false
	foundFinalizing := false
	for _, event := range events {
		if event.Kind != EventPhase {
			continue
		}
		if event.Phase == PhaseExecuting {
			t.Fatalf("did not expect %q phase event, got %+v", PhaseExecuting, events)
		}
		if event.Phase == PhaseScanningPrune {
			foundScanningPrune = true
		}
		if event.Phase == PhaseFinalizingOutput {
			foundFinalizing = true
		}
	}
	if !foundScanningPrune {
		t.Fatalf("expected %q phase event, got %+v", PhaseScanningPrune, events)
	}
	if !foundFinalizing {
		t.Fatalf("expected %q phase event, got %+v", PhaseFinalizingOutput, events)
	}
}

func TestRunEmitsTaskDoneEventsForSuccessAndFailure(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"ok":"hello","bad":"boom"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.translate = func(ctx context.Context, req translator.Request) (string, error) {
		if req.Source == "boom" {
			return "", errors.New("translation failed")
		}
		translator.SetUsage(ctx, translator.Usage{PromptTokens: 7, CompletionTokens: 2, TotalTokens: 9})
		return strings.ToUpper(req.Source), nil
	}

	events := make([]Event, 0, 8)
	report, err := svc.Run(context.Background(), Input{
		Workers: 1,
		OnEvent: func(event Event) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("run with events: %v", err)
	}
	if report.Succeeded != 1 || report.Failed != 1 {
		t.Fatalf("unexpected execution report: %+v", report)
	}

	success := 0
	failure := 0
	completedSeen := false
	maxTotalTokensSeen := 0
	for _, event := range events {
		if event.Kind == EventTaskDone {
			if event.TotalTokens > maxTotalTokensSeen {
				maxTotalTokensSeen = event.TotalTokens
			}
			if event.TaskSucceeded {
				success++
			} else {
				failure++
				if event.TotalTokens > 9 {
					t.Fatalf("unexpected cumulative token usage on failure event: %+v", event)
				}
			}
		}
		if event.Kind == EventCompleted {
			completedSeen = true
			if event.Succeeded != 1 || event.Failed != 1 {
				t.Fatalf("unexpected completed event: %+v", event)
			}
			if event.PromptTokens != 7 || event.CompletionTokens != 2 || event.TotalTokens != 9 {
				t.Fatalf("unexpected token totals on completed event: %+v", event)
			}
		}
	}
	if success != 1 || failure != 1 {
		t.Fatalf("unexpected task done events success=%d failure=%d events=%+v", success, failure, events)
	}
	if !completedSeen {
		t.Fatalf("expected completed event, got %+v", events)
	}
	if maxTotalTokensSeen != 9 {
		t.Fatalf("expected cumulative task event tokens to reach 9, got %d events=%+v", maxTotalTokensSeen, events)
	}
}

func TestRunContinueOnErrorReturnsPartialFailureReport(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	lockState := &lockfile.File{LocaleStates: map[string]lockfile.LocaleCheckpoint{}, RunCompleted: map[string]lockfile.RunCompletion{}}
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return lockState, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"ok":"hello","bad":"boom"}`), nil
		case targetPath:
			return []byte(`{"existing":"v"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source == "boom" {
			return "", errors.New("translation failed")
		}
		return strings.ToUpper(req.Source), nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	report, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.Succeeded != 1 || report.Failed != 1 {
		t.Fatalf("unexpected execution totals: %+v", report)
	}
	if report.PersistedToLock != 0 {
		t.Fatalf("expected lock rollback for failed target, got persisted=%d", report.PersistedToLock)
	}
	if len(report.Failures) != 1 || report.Failures[0].EntryKey != "bad" {
		t.Fatalf("unexpected failures: %+v", report.Failures)
	}
	if len(lockState.RunCompleted) != 0 {
		t.Fatalf("expected no completed lock entries for failed target, got %+v", lockState.RunCompleted)
	}

	var payload map[string]any
	if err := json.Unmarshal(written, &payload); err != nil {
		t.Fatalf("decode written payload: %v", err)
	}
	if payload["ok"] != "HELLO" {
		t.Fatalf("expected translated key to be written, got %+v", payload)
	}
	if payload["existing"] != "v" {
		t.Fatalf("expected existing key preserved, got %+v", payload)
	}
}

func TestRunLockWriterBatchesAndFlushesOnShutdown(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B","c":"C"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	lockWrites := 0
	seenSizes := []int{}
	lastCheckpointSize := -1
	lastActiveRunID := ""
	svc.saveLock = func(_ string, f lockfile.File) error {
		lockWrites++
		seenSizes = append(seenSizes, len(f.RunCompleted))
		lastCheckpointSize = len(f.RunCheckpoint)
		lastActiveRunID = f.ActiveRunID
		for identity, completion := range f.RunCompleted {
			if completion.SourceHash == "" {
				t.Fatalf("expected source hash persisted for %s", identity)
			}
		}
		return nil
	}

	report, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.Succeeded != 3 || report.PersistedToLock != 3 {
		t.Fatalf("unexpected lock persistence totals: %+v", report)
	}
	if lockWrites != 2 {
		t.Fatalf("expected one progress write plus one checkpoint cleanup write, got %d", lockWrites)
	}
	if seenSizes[len(seenSizes)-1] != 3 {
		t.Fatalf("expected final lock map size 3, got %v", seenSizes)
	}
	if lastCheckpointSize != 0 || lastActiveRunID != "" {
		t.Fatalf("expected checkpoints cleared after successful run, got run_checkpoint=%d active_run_id=%q", lastCheckpointSize, lastActiveRunID)
	}
}

func TestRunLockWriterFlushesPendingEntriesOnCancel(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	lockWrites := 0
	svc.saveLock = func(_ string, f lockfile.File) error {
		lockWrites++
		for identity, completion := range f.RunCompleted {
			if completion.SourceHash == "" {
				t.Fatalf("expected source hash persisted for %s", identity)
			}
		}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cancelOnce sync.Once
	report, err := svc.Run(ctx, Input{
		Workers: 1,
		OnEvent: func(e Event) {
			if e.Kind == EventTaskDone && e.TaskSucceeded {
				cancelOnce.Do(cancel)
			}
		},
	})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.PersistedToLock == 0 {
		t.Fatalf("expected cancel flush to persist at least one lock entry, got %+v", report)
	}
	if report.PersistedToLock != report.Succeeded {
		t.Fatalf("expected persisted entries to match successful tasks after cancel flush, got %+v", report)
	}
	if lockWrites != 2 {
		t.Fatalf("expected cancel flush plus checkpoint cleanup write, got %d writes", lockWrites)
	}
}

func TestRunLockWriterFlushesWhenBatchSizeReached(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.lockPersistBatchSize = 2
	svc.lockPersistFlushInterval = time.Hour
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B","c":"C","d":"D","e":"E"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	lockWrites := 0
	seenSizes := []int{}
	svc.saveLock = func(_ string, f lockfile.File) error {
		lockWrites++
		seenSizes = append(seenSizes, len(f.RunCompleted))
		return nil
	}

	report, err := svc.Run(context.Background(), Input{Workers: 1})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.Succeeded != 5 || report.PersistedToLock != 5 {
		t.Fatalf("unexpected lock persistence totals: %+v", report)
	}
	if lockWrites != 4 {
		t.Fatalf("expected flushes at batch 2,4 plus final target flush and checkpoint cleanup, got %d", lockWrites)
	}
	if len(seenSizes) != 4 || seenSizes[0] != 2 || seenSizes[1] != 4 || seenSizes[2] != 5 || seenSizes[3] != 5 {
		t.Fatalf("unexpected lock snapshot sizes: %v", seenSizes)
	}
}

func TestRunLockWriterFlushesOnTickerInterval(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.lockPersistBatchSize = 100
	svc.lockPersistFlushInterval = 5 * time.Millisecond
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B","c":"C"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		time.Sleep(20 * time.Millisecond)
		return strings.ToUpper(req.Source), nil
	}

	lockWrites := 0
	svc.saveLock = func(_ string, _ lockfile.File) error {
		lockWrites++
		return nil
	}

	report, err := svc.Run(context.Background(), Input{Workers: 1})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.Succeeded != 3 || report.PersistedToLock != 3 {
		t.Fatalf("unexpected lock persistence totals: %+v", report)
	}
	if lockWrites < 2 {
		t.Fatalf("expected at least one ticker-triggered flush plus final flush, got %d", lockWrites)
	}
}

func TestRunRetriesRetryableTranslateErrors(t *testing.T) {
	svc := newTestService()
	attempts := 0
	sleeps := 0
	originalSleep := sleepWithContext
	t.Cleanup(func() { sleepWithContext = originalSleep })
	sleepWithContext = func(ctx context.Context, _ time.Duration) error {
		sleeps++
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("status code 429")
		}
		return strings.ToUpper(req.Source), nil
	}

	report, err := svc.Run(context.Background(), Input{Workers: 1})
	if err != nil {
		t.Fatalf("run with retryable failures: %v", err)
	}
	if report.Succeeded != 1 || report.Failed != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 translate attempts, got %d", attempts)
	}
	if sleeps != 2 {
		t.Fatalf("expected 2 retry sleeps, got %d", sleeps)
	}
}

func TestRunResumesFromCheckpointAfterInterruptedRun(t *testing.T) {
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	aID := taskIdentity(targetPath, "a")
	now := time.Unix(1700000000, 0).UTC()
	lockState := &lockfile.File{
		LocaleStates: map[string]lockfile.LocaleCheckpoint{},
		ActiveRunID:  "run_1",
		RunCompleted: map[string]lockfile.RunCompletion{
			aID: {CompletedAt: now, SourceHash: hashSourceText("A")},
		},
		RunCheckpoint: map[string]lockfile.RunCheckpoint{
			aID: {RunID: "run_1", TargetPath: targetPath, SourcePath: sourcePath, TargetLocale: "fr", EntryKey: "a", Value: "a", SourceHash: hashSourceText("A"), UpdatedAt: now},
		},
	}

	svc := newTestService()
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) { return lockState, nil }
	svc.saveLock = func(_ string, f lockfile.File) error {
		*lockState = f
		return nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	translateCalls := 0
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		translateCalls++
		return strings.ToLower(req.Source), nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	report, err := svc.Run(context.Background(), Input{Workers: 1})
	if err != nil {
		t.Fatalf("resume run: %v", err)
	}
	if report.SkippedByLock != 1 || report.Succeeded != 1 {
		t.Fatalf("unexpected resume report: %+v", report)
	}
	if translateCalls != 1 {
		t.Fatalf("expected exactly one fresh translation after resume, got %d", translateCalls)
	}

	var payload map[string]any
	if err := json.Unmarshal(written, &payload); err != nil {
		t.Fatalf("decode resumed output: %v", err)
	}
	if payload["a"] != "a" || payload["b"] != "b" {
		t.Fatalf("expected resumed file to include checkpoint and new entries, got %+v", payload)
	}
}

func TestRunSkipsCompletedLocaleBatchAndFlushesFromCheckpoint(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	aID := taskIdentity(targetPath, "a")
	bID := taskIdentity(targetPath, "b")
	now := time.Unix(1700000000, 0).UTC()

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B"}`), nil
		case targetPath:
			return nil, os.ErrNotExist
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{
			LocaleStates: map[string]lockfile.LocaleCheckpoint{},
			ActiveRunID:  "run_1",
			RunCompleted: map[string]lockfile.RunCompletion{
				aID: {CompletedAt: now, SourceHash: hashSourceText("A")},
				bID: {CompletedAt: now, SourceHash: hashSourceText("B")},
			},
			RunCheckpoint: map[string]lockfile.RunCheckpoint{
				aID: {RunID: "run_1", TargetPath: targetPath, SourcePath: sourcePath, TargetLocale: "fr", EntryKey: "a", Value: "aa", SourceHash: hashSourceText("A"), UpdatedAt: now},
				bID: {RunID: "run_1", TargetPath: targetPath, SourcePath: sourcePath, TargetLocale: "fr", EntryKey: "b", Value: "bb", SourceHash: hashSourceText("B"), UpdatedAt: now},
			},
		}, nil
	}
	svc.translate = func(_ context.Context, _ translator.Request) (string, error) {
		return "", errors.New("translate should not be called for completed batch")
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	report, err := svc.Run(context.Background(), Input{Workers: 1})
	if err != nil {
		t.Fatalf("run from checkpoint batch: %v", err)
	}
	if report.ExecutableTotal != 0 || report.SkippedByLock != 2 {
		t.Fatalf("expected fully skipped executable set, got %+v", report)
	}

	var payload map[string]any
	if err := json.Unmarshal(written, &payload); err != nil {
		t.Fatalf("decode checkpoint output: %v", err)
	}
	if payload["a"] != "aa" || payload["b"] != "bb" {
		t.Fatalf("expected checkpoint values to be flushed, got %+v", payload)
	}
}

func TestRunFailsWhenCheckpointStagingConflicts(t *testing.T) {
	svc := newTestService()
	sourceAPath := "/tmp/source-a.json"
	sourceBPath := "/tmp/source-b.json"
	targetPath := "/tmp/out.json"
	aID := taskIdentity(targetPath, "a")
	bID := taskIdentity(targetPath, "b")
	now := time.Unix(1700000000, 0).UTC()

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := config.I18NConfig{
			Locales: config.LocaleConfig{Source: "en", Targets: []string{"fr"}},
			Buckets: map[string]config.BucketConfig{
				"ui": {
					Files: []config.BucketFileMapping{
						{From: sourceAPath, To: targetPath},
						{From: sourceBPath, To: targetPath},
					},
				},
			},
			Groups: map[string]config.GroupConfig{
				"default": {Targets: []string{"fr"}, Buckets: []string{"ui"}},
			},
			LLM: config.LLMConfig{Profiles: map[string]config.LLMProfile{
				"default": {
					Provider: "openai",
					Model:    "gpt-4.1-mini",
					Prompt:   "Translate {{source}} to {{target}}: {{input}}",
				},
			}},
		}
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourceAPath:
			return []byte(`{"a":"A"}`), nil
		case sourceBPath:
			return []byte(`{"b":"B"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{
			LocaleStates: map[string]lockfile.LocaleCheckpoint{},
			ActiveRunID:  "run_1",
			RunCompleted: map[string]lockfile.RunCompletion{
				aID: {CompletedAt: now, SourceHash: hashSourceText("A")},
				bID: {CompletedAt: now, SourceHash: hashSourceText("B")},
			},
			RunCheckpoint: map[string]lockfile.RunCheckpoint{
				aID: {RunID: "run_1", TargetPath: targetPath, SourcePath: sourceAPath, TargetLocale: "fr", EntryKey: "a", Value: "aa", SourceHash: hashSourceText("A"), UpdatedAt: now},
				bID: {RunID: "run_1", TargetPath: targetPath, SourcePath: sourceBPath, TargetLocale: "fr", EntryKey: "b", Value: "bb", SourceHash: hashSourceText("B"), UpdatedAt: now},
			},
		}, nil
	}
	svc.translate = func(_ context.Context, _ translator.Request) (string, error) {
		return "", errors.New("translate should not be called when everything is lock-skipped")
	}
	svc.writeFile = func(_ string, _ []byte) error {
		t.Fatalf("write should not be called on checkpoint staging conflict")
		return nil
	}

	_, err := svc.Run(context.Background(), Input{Workers: 1})
	if err == nil {
		t.Fatalf("expected conflict error")
	}
	if !strings.Contains(err.Error(), "stage checkpoint output") || !strings.Contains(err.Error(), "output staging conflict") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunWritesMarkdownUsingSourceTemplateWhenTargetMissing(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.md"
	targetPath := "/tmp/out.md"
	source := "---\ntitle: Welcome\n---\n\n# Heading\n\nHello `code` and [docs](https://example.com).\n\n```js\nconsole.log('x')\n```\n"

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "FR(" + req.Source + ")", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "title: Welcome") {
		t.Fatalf("expected frontmatter unchanged, got %q", out)
	}
	if !strings.Contains(out, "```js") || !strings.Contains(out, "console.log('x')") {
		t.Fatalf("expected code fence preserved, got %q", out)
	}
	if !strings.Contains(out, "https://example.com") {
		t.Fatalf("expected link destination preserved, got %q", out)
	}
	if !strings.Contains(out, "FR( Heading") || !strings.Contains(out, "FR(Hello )") {
		t.Fatalf("expected markdown text translated, got %q", out)
	}
}

func TestRunWritesMDXUsingSourceTemplateWhenTargetMissing(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.mdx"
	targetPath := "/tmp/out.mdx"
	source := "---\ntitle: Welcome\n---\n\nimport Tabs from '@theme/Tabs'\n\n<Tabs defaultValue=\"first\">\n  <Tab value=\"first\" label=\"First\">Run command.</Tab>\n</Tabs>\n"

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "FR(" + req.Source + ")", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "import Tabs") {
		t.Fatalf("expected import statement preserved, got %q", out)
	}
	if !strings.Contains(out, "defaultValue=\"first\"") || !strings.Contains(out, "label=\"First\"") {
		t.Fatalf("expected component attributes preserved, got %q", out)
	}
	if !strings.Contains(out, "FR(Run command.)") {
		t.Fatalf("expected prose translated, got %q", out)
	}
}

func TestRunWritesXLIFFWithInsertedUnitWhenExistingTargetPresent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.xlf"
	targetPath := "/tmp/out.xlf"
	source := `<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en" target-language="fr" datatype="plaintext" original="messages">
    <body>
      <trans-unit id="old">
        <source>Old text</source>
        <target>Old text</target>
      </trans-unit>
      <trans-unit id="new">
        <source>New text</source>
        <target>New text</target>
      </trans-unit>
    </body>
  </file>
</xliff>`
	target := `<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en" target-language="fr" datatype="plaintext" original="messages">
    <body>
      <trans-unit id="old">
        <source>Old text</source>
        <target>Ancien texte</target>
      </trans-unit>
    </body>
  </file>
</xliff>`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		case targetPath:
			return []byte(target), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{RunCompleted: map[string]lockfile.RunCompletion{
			taskIdentity(targetPath, "old"): {CompletedAt: time.Now(), SourceHash: hashSourceText("Old text")},
		}}, nil
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source != "New text" {
			t.Fatalf("unexpected translation request: %q", req.Source)
		}
		return "Nouveau texte", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "<trans-unit id=\"old\">") || !strings.Contains(out, "<target>Ancien texte</target>") {
		t.Fatalf("expected old translated unit preserved, got %q", out)
	}
	if !strings.Contains(out, "<trans-unit id=\"new\">") || !strings.Contains(out, "<target>Nouveau texte</target>") {
		t.Fatalf("expected inserted unit translated, got %q", out)
	}
}

func TestRunWritesPOWithInsertedEntryWhenExistingTargetPresent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.po"
	targetPath := "/tmp/out.po"
	source := `msgid ""
msgstr ""
"Project-Id-Version: test\n"

msgid "old"
msgstr "Old text"

msgid "new"
msgstr "New text"
`
	target := `msgid ""
msgstr ""
"Project-Id-Version: test\n"

msgid "old"
msgstr "Ancien texte"
`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		case targetPath:
			return []byte(target), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{RunCompleted: map[string]lockfile.RunCompletion{
			taskIdentity(targetPath, "old"): {CompletedAt: time.Now(), SourceHash: hashSourceText("Old text")},
		}}, nil
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source != "New text" {
			t.Fatalf("unexpected translation request: %q", req.Source)
		}
		return "Nouveau texte", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "msgid \"old\"\nmsgstr \"Ancien texte\"") {
		t.Fatalf("expected old translated entry preserved, got %q", out)
	}
	if !strings.Contains(out, "msgid \"new\"\nmsgstr \"Nouveau texte\"") {
		t.Fatalf("expected inserted entry translated, got %q", out)
	}
}

func TestRunWritesMarkdownWithInsertedSectionWhenExistingTargetPresent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.md"
	targetPath := "/tmp/out.md"
	source := "# Guide\n\nExisting intro.\n\nNew section added.\n\nExisting outro.\n"
	target := "# Guide\n\nIntro existant.\n\nConclusion existante.\n"

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		case targetPath:
			return []byte(target), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	sourceEntries, err := (translationfileparser.MarkdownParser{}).Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	var newSectionKey, newSectionText string
	for key, value := range sourceEntries {
		if strings.TrimSpace(value) == "New section added." {
			newSectionKey = key
			newSectionText = value
			break
		}
	}
	if newSectionKey == "" {
		t.Fatalf("expected key for inserted source section")
	}

	svc.loadLock = func(_ string) (*lockfile.File, error) {
		completed := map[string]lockfile.RunCompletion{}
		for key, value := range sourceEntries {
			if key == newSectionKey {
				continue
			}
			completed[taskIdentity(targetPath, key)] = lockfile.RunCompletion{
				CompletedAt: time.Now(),
				SourceHash:  hashSourceText(value),
			}
		}
		return &lockfile.File{RunCompleted: completed}, nil
	}

	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source != newSectionText {
			t.Fatalf("unexpected translation request for skipped entry: %q", req.Source)
		}
		return "Nouvelle section ajoutee.", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err = svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "Intro existant.") {
		t.Fatalf("expected existing translated intro preserved, got %q", out)
	}
	if !strings.Contains(out, "Nouvelle section ajoutee.") {
		t.Fatalf("expected inserted section translated, got %q", out)
	}
	if !strings.Contains(out, "Conclusion existante.") {
		t.Fatalf("expected existing translated outro preserved, got %q", out)
	}
}

func TestRunWritesAppleStringsUsingSourceTemplateWhenTargetMissing(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.strings"
	targetPath := "/tmp/out.strings"
	source := `/* Greeting */
"hello" = "Hello";
"multiline" = "First\nSecond";
`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source == "First\nSecond" {
			return "Premier\nDeuxieme", nil
		}
		return "FR(" + req.Source + ")", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "/* Greeting */") {
		t.Fatalf("expected comment preserved, got %q", out)
	}
	if !strings.Contains(out, `"hello" = "FR(Hello)";`) {
		t.Fatalf("expected greeting translated, got %q", out)
	}
	if !strings.Contains(out, `"multiline" = "Premier\nDeuxieme";`) {
		t.Fatalf("expected multiline translation escaped, got %q", out)
	}
}

func TestRunWritesAppleStringsWithInsertedKeyWhenExistingTargetPresent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.strings"
	targetPath := "/tmp/out.strings"
	source := `"old" = "Old text";
"new" = "New text";
`
	target := `"old" = "Ancien texte";
`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		case targetPath:
			return []byte(target), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{RunCompleted: map[string]lockfile.RunCompletion{
			taskIdentity(targetPath, "old"): {CompletedAt: time.Now(), SourceHash: hashSourceText("Old text")},
		}}, nil
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source != "New text" {
			t.Fatalf("unexpected translation request: %q", req.Source)
		}
		return "Nouveau texte", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, `"old" = "Ancien texte";`) {
		t.Fatalf("expected old translated key preserved, got %q", out)
	}
	if !strings.Contains(out, `"new" = "Nouveau texte";`) {
		t.Fatalf("expected inserted key translated, got %q", out)
	}
}

func TestRunWritesAppleStringsdictUsingSourceTemplateWhenTargetMissing(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.stringsdict"
	targetPath := "/tmp/out.stringsdict"
	source := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>item_count</key>
  <dict>
    <key>NSStringLocalizedFormatKey</key>
    <string>%#@items@</string>
    <key>items</key>
    <dict>
      <key>one</key>
      <string>%d item</string>
      <key>other</key>
      <string>%d items</string>
    </dict>
  </dict>
</dict>
</plist>
`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		switch req.Source {
		case "%d item":
			return "%d article", nil
		case "%d items":
			return "%d articles", nil
		default:
			return req.Source, nil
		}
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "<string>%#@items@</string>") {
		t.Fatalf("expected format placeholder preserved, got %q", out)
	}
	if !strings.Contains(out, "<string>%d article</string>") {
		t.Fatalf("expected one plural category translated, got %q", out)
	}
	if !strings.Contains(out, "<string>%d articles</string>") {
		t.Fatalf("expected other plural category translated, got %q", out)
	}
}

func TestRunWritesAppleStringsdictWithInsertedKeyWhenExistingTargetPresent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.stringsdict"
	targetPath := "/tmp/out.stringsdict"
	source := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>old</key>
  <string>Old text</string>
  <key>new</key>
  <string>New text</string>
</dict>
</plist>
`
	target := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>old</key>
  <string>Ancien texte</string>
</dict>
</plist>
`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		case targetPath:
			return []byte(target), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{RunCompleted: map[string]lockfile.RunCompletion{
			taskIdentity(targetPath, "old"): {CompletedAt: time.Now(), SourceHash: hashSourceText("Old text")},
		}}, nil
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source != "New text" {
			t.Fatalf("unexpected translation request: %q", req.Source)
		}
		return "Nouveau texte", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "<key>old</key>\n  <string>Ancien texte</string>") {
		t.Fatalf("expected old translated key preserved, got %q", out)
	}
	if !strings.Contains(out, "<key>new</key>\n  <string>Nouveau texte</string>") {
		t.Fatalf("expected inserted key translated, got %q", out)
	}
}

func TestRunWritesCSVUsingSourceTemplateWhenTargetMissing(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.csv"
	targetPath := "/tmp/out.csv"
	source := "key,source,target\nhello,Hello,Hello\n"

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "FR(" + req.Source + ")", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "key,source,target") {
		t.Fatalf("expected csv headers preserved, got %q", out)
	}
	if !strings.Contains(out, "hello,Hello,FR(Hello)") {
		t.Fatalf("expected csv translation written to target column, got %q", out)
	}
}

func TestRunReturnsFatalErrorWhenLockWriteFails(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.saveLock = func(_ string, _ lockfile.File) error {
		return errors.New("disk full")
	}

	writeCount := 0
	svc.writeFile = func(_ string, _ []byte) error {
		writeCount++
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err == nil || !strings.Contains(err.Error(), "persist lock state") {
		t.Fatalf("expected fatal lock persistence error, got %v", err)
	}
	if writeCount != 0 {
		t.Fatalf("expected no output flush on fatal lock error, got %d writes", writeCount)
	}
}

func TestRunDryRunReportsPruneCandidates(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello","nested.title":"Title"}`), nil
		case targetPath:
			return []byte(`{"hello":"Bonjour","nested":{"title":"Titre","old":"Ancien"},"legacy":"Legacy"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	report, err := svc.Run(context.Background(), Input{DryRun: true, Prune: true})
	if err != nil {
		t.Fatalf("run dry-run prune: %v", err)
	}
	if len(report.PruneCandidates) != 2 {
		t.Fatalf("expected 2 prune candidates, got %+v", report.PruneCandidates)
	}
	if report.PruneCandidates[0].EntryKey != "legacy" || report.PruneCandidates[1].EntryKey != "nested.old" {
		t.Fatalf("unexpected prune candidates ordering: %+v", report.PruneCandidates)
	}
}

func TestRunPruneRemovesStaleKeysForJSONAndNestedKeys(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello","nested.title":"Title"}`), nil
		case targetPath:
			return []byte(`{"hello":"Bonjour","nested":{"title":"Titre","old":"Ancien"},"legacy":"Legacy"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source == "Hello" {
			return "Salut", nil
		}
		return "Titre mis à jour", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	report, err := svc.Run(context.Background(), Input{Prune: true})
	if err != nil {
		t.Fatalf("run prune: %v", err)
	}
	if report.PruneApplied != 2 {
		t.Fatalf("expected 2 prune deletions applied, got %+v", report)
	}

	var payload map[string]any
	if err := json.Unmarshal(written, &payload); err != nil {
		t.Fatalf("decode written payload: %v", err)
	}
	if _, ok := payload["legacy"]; ok {
		t.Fatalf("expected legacy key pruned, got %+v", payload)
	}
	nested, ok := payload["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested object, got %+v", payload)
	}
	if _, ok := nested["old"]; ok {
		t.Fatalf("expected nested old key pruned, got %+v", nested)
	}
	if nested["title"] != "Titre mis à jour" {
		t.Fatalf("expected nested title preserved and updated, got %+v", nested)
	}
}

func TestRunPruneSafetyLimitBlocksMassDeletion(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello"}`), nil
		case targetPath:
			return []byte(`{"hello":"Bonjour","old":"ancien","older":"ancien2"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	_, err := svc.Run(context.Background(), Input{Prune: true, PruneLimit: 1})
	if err == nil || !strings.Contains(err.Error(), "prune safety limit exceeded") {
		t.Fatalf("expected prune safety limit error, got %v", err)
	}
}

func TestRunFiltersTasksByBucket(t *testing.T) {
	svc := newTestService()
	uiSource := "/tmp/ui_source.json"
	uiTarget := "/tmp/ui_out.json"
	marketingSource := "/tmp/marketing_source.json"
	marketingTarget := "/tmp/marketing_out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(uiSource, uiTarget)
		cfg.Buckets["marketing"] = config.BucketConfig{Files: []config.BucketFileMapping{{From: marketingSource, To: marketingTarget}}}
		cfg.Groups["default"] = config.GroupConfig{Targets: []string{"fr"}, Buckets: []string{"ui", "marketing"}}
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case uiSource:
			return []byte(`{"ui_key":"Hello"}`), nil
		case marketingSource:
			return []byte(`{"mkt_key":"Bonjour"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	report, err := svc.Run(context.Background(), Input{DryRun: true, Bucket: "marketing"})
	if err != nil {
		t.Fatalf("run dry-run: %v", err)
	}
	if report.PlannedTotal != 1 || report.ExecutableTotal != 1 {
		t.Fatalf("expected only one planned task for filtered bucket, got %+v", report)
	}
	if report.Executable[0].EntryKey != "mkt_key" {
		t.Fatalf("unexpected executable task: %+v", report.Executable)
	}
}

func TestRunReturnsErrorForUnknownBucketFilter(t *testing.T) {
	svc := newTestService()

	_, err := svc.Run(context.Background(), Input{DryRun: true, Bucket: "unknown"})
	if err == nil || !strings.Contains(err.Error(), `unknown bucket "unknown"`) {
		t.Fatalf("expected unknown bucket error, got %v", err)
	}
}

func TestRunFiltersTasksByGroup(t *testing.T) {
	svc := newTestService()
	sourceA := "/tmp/source_a.json"
	targetA := "/tmp/out_a.json"
	sourceB := "/tmp/source_b.json"
	targetB := "/tmp/out_b.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourceA, targetA)
		cfg.Buckets["marketing"] = config.BucketConfig{
			Files: []config.BucketFileMapping{{From: sourceB, To: targetB}},
		}
		cfg.Groups = map[string]config.GroupConfig{
			"default": {Targets: []string{"fr"}, Buckets: []string{"ui"}},
			"tests":   {Targets: []string{"fr"}, Buckets: []string{"marketing"}},
		}
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourceA:
			return []byte(`{"ui_key":"UI"}`), nil
		case sourceB:
			return []byte(`{"mkt_key":"MKT"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	report, err := svc.Run(context.Background(), Input{DryRun: true, Group: "tests"})
	if err != nil {
		t.Fatalf("run dry-run: %v", err)
	}
	if report.PlannedTotal != 1 || report.ExecutableTotal != 1 {
		t.Fatalf("expected one planned task for filtered group, got %+v", report)
	}
	if report.Executable[0].EntryKey != "mkt_key" {
		t.Fatalf("unexpected executable task: %+v", report.Executable)
	}
}

func TestRunReturnsErrorForUnknownGroupFilter(t *testing.T) {
	svc := newTestService()

	_, err := svc.Run(context.Background(), Input{DryRun: true, Group: "unknown"})
	if err == nil || !strings.Contains(err.Error(), `unknown group "unknown"`) {
		t.Fatalf("expected unknown group error, got %v", err)
	}
}

func TestRunAggregatesTokenUsageByLocaleAndBatch(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/fr_out.json"
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello"}`), nil
		case "/tmp/fr_out.json":
			return []byte(`{}`), nil
		case "/tmp/es_out.json":
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		cfg.Buckets["ui"] = config.BucketConfig{Files: []config.BucketFileMapping{{From: sourcePath, To: "/tmp/{{target}}_out.json"}}}
		cfg.Groups["default"] = config.GroupConfig{Targets: []string{"fr", "es"}, Buckets: []string{"ui"}}
		return &cfg, nil
	}
	svc.translate = func(ctx context.Context, req translator.Request) (string, error) {
		if req.TargetLanguage == "fr" {
			translator.SetUsage(ctx, translator.Usage{PromptTokens: 10, CompletionTokens: 4, TotalTokens: 14})
			return "Bonjour", nil
		}
		translator.SetUsage(ctx, translator.Usage{PromptTokens: 6, CompletionTokens: 3, TotalTokens: 9})
		return "Hola", nil
	}

	report, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.PromptTokens != 16 || report.CompletionTokens != 7 || report.TotalTokens != 23 {
		t.Fatalf("unexpected token totals: %+v", report.TokenUsage)
	}
	if len(report.LocaleUsage) != 2 {
		t.Fatalf("expected 2 locale usage entries, got %+v", report.LocaleUsage)
	}
	if got := report.LocaleUsage["fr"]; got.PromptTokens != 10 || got.CompletionTokens != 4 || got.TotalTokens != 14 {
		t.Fatalf("unexpected fr token usage: %+v", got)
	}
	if got := report.LocaleUsage["es"]; got.PromptTokens != 6 || got.CompletionTokens != 3 || got.TotalTokens != 9 {
		t.Fatalf("unexpected es token usage: %+v", got)
	}
	if len(report.Batches) != 2 {
		t.Fatalf("expected 2 batch usage entries, got %+v", report.Batches)
	}
}

func newTestService() *Service {
	now := time.Unix(1700000000, 0).UTC()
	sourcePath := "/tmp/source.json"

	return &Service{
		loadConfig: func(_ string) (*config.I18NConfig, error) {
			cfg := testConfig(sourcePath, "/tmp/out.json")
			return &cfg, nil
		},
		loadLock: func(_ string) (*lockfile.File, error) {
			return &lockfile.File{LocaleStates: map[string]lockfile.LocaleCheckpoint{}, RunCompleted: map[string]lockfile.RunCompletion{}}, nil
		},
		saveLock: func(_ string, _ lockfile.File) error { return nil },
		readFile: func(path string) ([]byte, error) {
			switch path {
			case sourcePath:
				return []byte(`{"hello":"Hello"}`), nil
			case "/tmp/out.json":
				return []byte(`{}`), nil
			default:
				return nil, filepath.ErrBadPattern
			}
		},
		writeFile: func(_ string, _ []byte) error { return nil },
		translate: func(_ context.Context, req translator.Request) (string, error) {
			return strings.ToUpper(req.Source), nil
		},
		newParser: translationfileparser.NewDefaultStrategy,
		now:       func() time.Time { return now },
		numCPU:    func() int { return 2 },

		lockPersistBatchSize:     32,
		lockPersistFlushInterval: 250 * time.Millisecond,
	}
}

func testConfig(sourcePath, targetPath string) config.I18NConfig {
	return config.I18NConfig{
		Locales: config.LocaleConfig{
			Source:  "en",
			Targets: []string{"fr"},
		},
		Buckets: map[string]config.BucketConfig{
			"ui": {
				Files: []config.BucketFileMapping{{
					From: sourcePath,
					To:   targetPath,
				}},
			},
		},
		Groups: map[string]config.GroupConfig{
			"default": {
				Targets: []string{"fr"},
				Buckets: []string{"ui"},
			},
		},
		LLM: config.LLMConfig{
			Profiles: map[string]config.LLMProfile{
				"default": {
					Provider: "openai",
					Model:    "gpt-4.1-mini",
					Prompt:   "Translate {{source}} to {{target}}: {{input}}",
				},
			},
		},
	}
}

func TestShouldIgnoreSourcePath(t *testing.T) {
	targets := []string{"fr", "es", "zh"}
	if !shouldIgnoreSourcePath("docs/fr/index.mdx", targets) {
		t.Fatalf("expected docs/fr/index.mdx to be ignored")
	}
	if !shouldIgnoreSourcePath("docs/es/guides/quickstart.mdx", targets) {
		t.Fatalf("expected nested locale path to be ignored")
	}
	if shouldIgnoreSourcePath("docs/index.mdx", targets) {
		t.Fatalf("expected root docs source path not to be ignored")
	}
}

func TestResolveSourcePathsWithDoublestar(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "index.mdx"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write root mdx: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "nested", "guide.mdx"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write nested mdx: %v", err)
	}

	pattern := filepath.Join(dir, "docs", "**", "*.mdx")
	paths, err := resolveSourcePaths(pattern)
	if err != nil {
		t.Fatalf("resolve source paths: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d (%v)", len(paths), paths)
	}
}

func TestResolveTargetPathWithDoublestar(t *testing.T) {
	sourcePattern := "docs/**/*.mdx"
	targetPattern := "docs/fr/**/*.mdx"
	sourcePath := "docs/guides/quickstart.mdx"

	got, err := resolveTargetPath(sourcePattern, targetPattern, sourcePath)
	if err != nil {
		t.Fatalf("resolve target path: %v", err)
	}
	if want := "docs/fr/guides/quickstart.mdx"; got != want {
		t.Fatalf("target path = %q, want %q", got, want)
	}
}

func TestResolveTargetPathRequiresDoublestarInTargetWhenSourceHasIt(t *testing.T) {
	_, err := resolveTargetPath("docs/**/*.mdx", "docs/fr/index.mdx", "docs/index.mdx")
	if err == nil || !strings.Contains(err.Error(), "must include glob tokens") {
		t.Fatalf("expected doublestar mapping error, got %v", err)
	}
}

func TestRunCompletedEventParityForEarlyExitAndFatalError(t *testing.T) {
	t.Run("dry run early exit", func(t *testing.T) {
		svc := newTestService()
		sourcePath := "/tmp/source.json"
		targetPath := "/tmp/out.json"
		svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
			cfg := testConfig(sourcePath, targetPath)
			return &cfg, nil
		}
		svc.readFile = func(path string) ([]byte, error) {
			switch path {
			case sourcePath:
				return []byte(`{"a":"A"}`), nil
			default:
				return nil, filepath.ErrBadPattern
			}
		}

		var completed Event
		report, err := svc.Run(context.Background(), Input{DryRun: true, OnEvent: func(e Event) {
			if e.Kind == EventCompleted {
				completed = e
			}
		}})
		if err != nil {
			t.Fatalf("run dry run: %v", err)
		}
		if completed.PlannedTotal != report.PlannedTotal || completed.SkippedByLock != report.SkippedByLock || completed.ExecutableTotal != report.ExecutableTotal {
			t.Fatalf("completed event mismatch: event=%+v report=%+v", completed, report)
		}
	})

	t.Run("fatal error exit", func(t *testing.T) {
		svc := newTestService()
		sourcePath := "/tmp/source.json"
		targetPath := "/tmp/out.json"
		svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
			cfg := testConfig(sourcePath, targetPath)
			return &cfg, nil
		}
		svc.readFile = func(path string) ([]byte, error) {
			switch path {
			case sourcePath:
				return []byte(`{"a":"A"}`), nil
			case targetPath:
				return []byte(`{}`), nil
			default:
				return nil, filepath.ErrBadPattern
			}
		}
		svc.saveLock = func(_ string, _ lockfile.File) error {
			return errors.New("disk full")
		}

		var completed Event
		report, err := svc.Run(context.Background(), Input{OnEvent: func(e Event) {
			if e.Kind == EventCompleted {
				completed = e
			}
		}})
		if err == nil {
			t.Fatalf("expected fatal error")
		}
		if completed.PlannedTotal != report.PlannedTotal || completed.Succeeded != report.Succeeded || completed.Failed != report.Failed || completed.PersistedToLock != report.PersistedToLock {
			t.Fatalf("completed event mismatch: event=%+v report=%+v", completed, report)
		}
	})
}

func TestMarshalTargetFileDispatchParity(t *testing.T) {
	svc := newTestService()
	sourceTemplate := map[string][]byte{
		"/tmp/source.xlf":         []byte(`<?xml version="1.0" encoding="UTF-8"?><xliff version="1.2"><file><body><trans-unit id="hello"><source>Hello</source><target>Hello</target></trans-unit></body></file></xliff>`),
		"/tmp/source.po":          []byte("msgid \"hello\"\nmsgstr \"Hello\"\n"),
		"/tmp/source.md":          []byte("# Hello\n"),
		"/tmp/source.mdx":         []byte("# Hello\n"),
		"/tmp/source.strings":     []byte("\"hello\" = \"Hello\";\n"),
		"/tmp/source.stringsdict": []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?><plist version=\"1.0\"><dict><key>hello</key><string>Hello</string></dict></plist>"),
		"/tmp/source.csv":         []byte("key,source,target\nhello,Hello,Hello\n"),
		"/tmp/source.json":        []byte(`{"hello":"Hello"}`),
		"/tmp/source.arb":         []byte(`{"@@locale":"en","hello":"Hello","@hello":{"description":"Greeting"}}`),
	}
	svc.readFile = func(path string) ([]byte, error) {
		if b, ok := sourceTemplate[path]; ok {
			return b, nil
		}
		return nil, os.ErrNotExist
	}

	cases := []struct {
		target string
		source string
	}{
		{target: "/tmp/out.xlf", source: "/tmp/source.xlf"},
		{target: "/tmp/out.xlif", source: "/tmp/source.xlf"},
		{target: "/tmp/out.xliff", source: "/tmp/source.xlf"},
		{target: "/tmp/out.po", source: "/tmp/source.po"},
		{target: "/tmp/out.md", source: "/tmp/source.md"},
		{target: "/tmp/out.mdx", source: "/tmp/source.mdx"},
		{target: "/tmp/out.strings", source: "/tmp/source.strings"},
		{target: "/tmp/out.stringsdict", source: "/tmp/source.stringsdict"},
		{target: "/tmp/out.csv", source: "/tmp/source.csv"},
		{target: "/tmp/out.json", source: "/tmp/source.json"},
		{target: "/tmp/out.arb", source: "/tmp/source.arb"},
	}

	for _, tc := range cases {
		content, err := svc.marshalTargetFile(tc.target, tc.source, "fr", map[string]string{"hello": "Bonjour"}, map[string]string{"hello": "Bonjour"})
		if err != nil {
			t.Fatalf("marshal %s: %v", tc.target, err)
		}
		if len(content) == 0 {
			t.Fatalf("marshal %s returned empty content", tc.target)
		}
	}
}

func TestMarshalSourceTemplateTargetPrefersTargetTemplateForXLIFFWhenAllKeysPresent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.xlf"
	targetPath := "/tmp/out.xlf"
	source := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <!-- source-note -->
      <trans-unit id="hello"><source>Hello</source><target>Hello</target></trans-unit>
    </body>
  </file>
</xliff>`)
	target := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <!-- target-note -->
      <trans-unit id="hello"><source>Hello</source><target>Bonjour</target></trans-unit>
    </body>
  </file>
</xliff>`)
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".xlf", targetPath, sourcePath, "fr", map[string]string{"hello": "Salut"})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}
	out := string(content)
	if !strings.Contains(out, "target-note") || strings.Contains(out, "source-note") {
		t.Fatalf("expected target template metadata preserved, got %q", out)
	}
}

func TestMarshalSourceTemplateTargetPrefersTargetTemplateForPOWhenAllKeysPresent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.po"
	targetPath := "/tmp/out.po"
	source := []byte(`# source-comment
msgid "hello"
msgstr "Hello"
`)
	target := []byte(`# target-comment
msgid "hello"
msgstr "Bonjour"
`)
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".po", targetPath, sourcePath, "fr", map[string]string{"hello": "Salut"})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}
	out := string(content)
	if !strings.Contains(out, "# target-comment") || strings.Contains(out, "# source-comment") {
		t.Fatalf("expected target template comment preserved, got %q", out)
	}
}

func TestMarshalSourceTemplateTargetPrefersTargetTemplateForStringsWhenAllKeysPresent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.strings"
	targetPath := "/tmp/out.strings"
	source := []byte("/* source-comment */\n\"hello\" = \"Hello\";\n")
	target := []byte("/* target-comment */\n\"hello\" = \"Bonjour\";\n")
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".strings", targetPath, sourcePath, "fr", map[string]string{"hello": "Salut"})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}
	out := string(content)
	if !strings.Contains(out, "target-comment") || strings.Contains(out, "source-comment") {
		t.Fatalf("expected target template comment preserved, got %q", out)
	}
}

func TestMarshalSourceTemplateTargetPrefersTargetTemplateForStringsdictWhenAllKeysPresent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.stringsdict"
	targetPath := "/tmp/out.stringsdict"
	source := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <!-- source-comment -->
  <key>hello</key>
  <string>Hello</string>
</dict>
</plist>`)
	target := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <!-- target-comment -->
  <key>hello</key>
  <string>Bonjour</string>
</dict>
</plist>`)
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".stringsdict", targetPath, sourcePath, "fr", map[string]string{"hello": "Salut"})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}
	out := string(content)
	if !strings.Contains(out, "target-comment") || strings.Contains(out, "source-comment") {
		t.Fatalf("expected target template comment preserved, got %q", out)
	}
}

func TestMarshalSourceTemplateTargetPrefersTargetTemplateForARBWhenAllKeysPresent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.arb"
	targetPath := "/tmp/out.arb"
	source := []byte(`{
  "@@locale": "en",
  "hello": "Hello",
  "@hello": {
    "description": "source-description"
  }
}`)
	target := []byte(`{
  "@@locale": "fr",
  "hello": "Bonjour",
  "@hello": {
    "description": "target-description"
  }
}`)
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".arb", targetPath, sourcePath, "fr", map[string]string{"hello": "Salut"})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("decode output arb: %v", err)
	}

	if payload["@@locale"] != "fr" {
		t.Fatalf("expected target template locale metadata preserved, got %#v", payload["@@locale"])
	}
	meta, ok := payload["@hello"].(map[string]any)
	if !ok {
		t.Fatalf("expected @hello metadata map, got %#v", payload["@hello"])
	}
	if meta["description"] != "target-description" {
		t.Fatalf("expected target template metadata preserved, got %#v", meta["description"])
	}
}

func TestMarshalSourceTemplateTargetDeletesRemovedKey(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.po"
	targetPath := "/tmp/out.po"
	source := []byte(`msgid "keep"
msgstr "Keep"
`)
	target := []byte(`# target-comment
msgid "keep"
msgstr "Garder"

msgid "delete_me"
msgstr "Supprimer"
`)
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".po", targetPath, sourcePath, "fr", map[string]string{"keep": "Garder"})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}
	out := string(content)
	if strings.Contains(out, `msgid "delete_me"`) {
		t.Fatalf("expected deleted key to be removed, got %q", out)
	}
}

func TestMarshalSourceTemplateTargetDeletesAndInsertsKey(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.po"
	targetPath := "/tmp/out.po"
	source := []byte(`msgid "keep"
msgstr "Keep"

msgid "new_key"
msgstr "New value"
`)
	target := []byte(`# target-comment
msgid "keep"
msgstr "Garder"

msgid "delete_me"
msgstr "Supprimer"
`)
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".po", targetPath, sourcePath, "fr", map[string]string{
		"keep":    "Garder",
		"new_key": "Nouvelle valeur",
	})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}
	out := string(content)
	if strings.Contains(out, `msgid "delete_me"`) {
		t.Fatalf("expected deleted key to be removed, got %q", out)
	}
	if !strings.Contains(out, "msgid \"new_key\"\nmsgstr \"Nouvelle valeur\"") {
		t.Fatalf("expected new key inserted and translated, got %q", out)
	}
}

func TestMarshalSourceTemplateTargetFallsBackToSourceWhenTargetParseFails(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.po"
	targetPath := "/tmp/out.po"
	source := []byte(`# source-comment
msgid "keep"
msgstr "Keep"

msgid "new_key"
msgstr "New value"
`)
	target := []byte("this is not valid po content")
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".po", targetPath, sourcePath, "fr", map[string]string{
		"keep":    "Garder",
		"new_key": "Nouvelle valeur",
	})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}
	out := string(content)
	if !strings.Contains(out, "# source-comment") {
		t.Fatalf("expected source template fallback on target parse failure, got %q", out)
	}
	if !strings.Contains(out, "msgid \"new_key\"\nmsgstr \"Nouvelle valeur\"") {
		t.Fatalf("expected new key present after fallback, got %q", out)
	}
}

func TestMarshalSourceTemplateTargetDeletesAndInsertsKeyForXLIFF(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.xlf"
	targetPath := "/tmp/out.xlf"
	source := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="keep"><source>Keep</source><target>Keep</target></trans-unit>
      <trans-unit id="new_key"><source>New value</source><target>New value</target></trans-unit>
    </body>
  </file>
</xliff>`)
	target := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="keep"><source>Keep</source><target>Garder</target></trans-unit>
      <trans-unit id="delete_me"><source>Delete</source><target>Supprimer</target></trans-unit>
    </body>
  </file>
</xliff>`)
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".xlf", targetPath, sourcePath, "fr", map[string]string{
		"keep":    "Garder",
		"new_key": "Nouvelle valeur",
	})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}
	out := string(content)
	if strings.Contains(out, `id="delete_me"`) {
		t.Fatalf("expected deleted unit removed, got %q", out)
	}
	if !strings.Contains(out, `id="new_key"`) || !strings.Contains(out, "<target>Nouvelle valeur</target>") {
		t.Fatalf("expected new unit inserted and translated, got %q", out)
	}
}

func TestMarshalSourceTemplateTargetDeletesAndInsertsKeyForStrings(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.strings"
	targetPath := "/tmp/out.strings"
	source := []byte(`"keep" = "Keep";
"new_key" = "New value";
`)
	target := []byte(`"keep" = "Garder";
"delete_me" = "Supprimer";
`)
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".strings", targetPath, sourcePath, "fr", map[string]string{
		"keep":    "Garder",
		"new_key": "Nouvelle valeur",
	})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}
	out := string(content)
	if strings.Contains(out, `"delete_me"`) {
		t.Fatalf("expected deleted key removed, got %q", out)
	}
	if !strings.Contains(out, `"new_key" = "Nouvelle valeur";`) {
		t.Fatalf("expected new key inserted and translated, got %q", out)
	}
}

func TestMarshalSourceTemplateTargetDeletesAndInsertsKeyForStringsdict(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.stringsdict"
	targetPath := "/tmp/out.stringsdict"
	source := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>keep</key>
  <string>Keep</string>
  <key>new_key</key>
  <string>New value</string>
</dict>
</plist>`)
	target := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>keep</key>
  <string>Garder</string>
  <key>delete_me</key>
  <string>Supprimer</string>
</dict>
</plist>`)
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return source, nil
		case targetPath:
			return target, nil
		default:
			return nil, os.ErrNotExist
		}
	}

	content, err := svc.marshalSourceTemplateTarget(".stringsdict", targetPath, sourcePath, "fr", map[string]string{
		"keep":    "Garder",
		"new_key": "Nouvelle valeur",
	})
	if err != nil {
		t.Fatalf("marshal source-template target: %v", err)
	}
	out := string(content)
	if strings.Contains(out, "<key>delete_me</key>") {
		t.Fatalf("expected deleted key removed, got %q", out)
	}
	if !strings.Contains(out, "<key>new_key</key>") || !strings.Contains(out, "<string>Nouvelle valeur</string>") {
		t.Fatalf("expected new key inserted and translated, got %q", out)
	}
}

func TestRunPruneRemovesDeletedPOKeys(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.po"
	targetPath := "/tmp/out.po"
	source := `msgid "keep"
msgstr "Keep"
`
	target := `msgid "keep"
msgstr "Garder"

msgid "remove_me"
msgstr "Supprimer"
`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		case targetPath:
			return []byte(target), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source != "Keep" {
			t.Fatalf("unexpected source %q", req.Source)
		}
		return "Garder", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	report, err := svc.Run(context.Background(), Input{Prune: true})
	if err != nil {
		t.Fatalf("run prune: %v", err)
	}
	if report.PruneApplied == 0 {
		t.Fatalf("expected prune to apply deletions, got %+v", report)
	}
	out := string(written)
	if strings.Contains(out, `msgid "remove_me"`) {
		t.Fatalf("expected removed key pruned, got %q", out)
	}
}

func TestRunPruneRemovesDeletedXLIFFUnits(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.xlf"
	targetPath := "/tmp/out.xlf"
	source := `<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="keep"><source>Keep</source><target>Keep</target></trans-unit>
    </body>
  </file>
</xliff>`
	target := `<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="keep"><source>Keep</source><target>Garder</target></trans-unit>
      <trans-unit id="remove_me"><source>Remove</source><target>Supprimer</target></trans-unit>
    </body>
  </file>
</xliff>`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		case targetPath:
			return []byte(target), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source != "Keep" {
			t.Fatalf("unexpected source %q", req.Source)
		}
		return "Garder", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	report, err := svc.Run(context.Background(), Input{Prune: true})
	if err != nil {
		t.Fatalf("run prune: %v", err)
	}
	if report.PruneApplied == 0 {
		t.Fatalf("expected prune to apply deletions, got %+v", report)
	}
	out := string(written)
	if strings.Contains(out, `id="remove_me"`) {
		t.Fatalf("expected removed unit pruned, got %q", out)
	}
}

func TestRunContinuesWhenOneTargetFileFailsToFlush(t *testing.T) {
	svc := newTestService()
	badSource := "/tmp/bad_source.json"
	goodSource := "/tmp/good_source.json"
	badTarget := "/tmp/bad_out.unsupported"
	goodTarget := "/tmp/good_out.json"

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(goodSource, goodTarget)
		cfg.Buckets["bad"] = config.BucketConfig{
			Files: []config.BucketFileMapping{{From: badSource, To: badTarget}},
		}
		cfg.Groups["default"] = config.GroupConfig{Targets: []string{"fr"}, Buckets: []string{"bad", "ui"}}
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case badSource:
			return []byte(`{"bad":"Bad"}`), nil
		case goodSource:
			return []byte(`{"good":"Good"}`), nil
		case goodTarget:
			return []byte(`{}`), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	writes := map[string][]byte{}
	var writeMu sync.Mutex
	svc.writeFile = func(path string, content []byte) error {
		writeMu.Lock()
		writes[path] = append([]byte(nil), content...)
		writeMu.Unlock()
		return nil
	}

	report, err := svc.Run(context.Background(), Input{Workers: 1})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.Failed == 0 {
		t.Fatalf("expected at least one failure for unsupported extension, got %+v", report)
	}

	writeMu.Lock()
	_, wroteBad := writes[badTarget]
	goodContent, wroteGood := writes[goodTarget]
	writeMu.Unlock()
	if wroteBad {
		t.Fatalf("expected bad target not to be flushed")
	}
	if !wroteGood {
		t.Fatalf("expected good target to still flush")
	}
	if !strings.Contains(string(goodContent), "\"good\": \"GOOD\"") {
		t.Fatalf("expected good target content to be written, got %q", string(goodContent))
	}
}

func TestRunWritesFormatJSJSONDefaultMessageOnly(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	source := `{
  "checkout.submit": {
    "defaultMessage": "Submit order",
    "description": "Checkout submit button"
  },
  "home.title": {
    "defaultMessage": "Welcome",
    "description": "Homepage title"
  }
}`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "FR(" + req.Source + ")", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	var payload map[string]map[string]any
	if err := json.Unmarshal(written, &payload); err != nil {
		t.Fatalf("decode written payload: %v", err)
	}
	if payload["checkout.submit"]["defaultMessage"] != "FR(Submit order)" {
		t.Fatalf("expected translated checkout.submit defaultMessage, got %+v", payload["checkout.submit"])
	}
	if payload["checkout.submit"]["description"] != "Checkout submit button" {
		t.Fatalf("expected checkout.submit description preserved, got %+v", payload["checkout.submit"])
	}
	if payload["home.title"]["defaultMessage"] != "FR(Welcome)" {
		t.Fatalf("expected translated home.title defaultMessage, got %+v", payload["home.title"])
	}
	if _, ok := payload["checkout"]; ok {
		t.Fatalf("unexpected nested key for dotted id in payload: %+v", payload)
	}
}

func TestRunMixedDefaultMessageJSONUsesStandardNestedMode(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	source := `{
  "meta": {
    "defaultMessage": "Do not treat as FormatJS root",
    "note": "This is nested metadata text"
  },
  "home": {
    "title": "Welcome"
  }
}`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "FR(" + req.Source + ")", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(written, &payload); err != nil {
		t.Fatalf("decode written payload: %v", err)
	}

	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested meta object, got %+v", payload["meta"])
	}
	home, ok := payload["home"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested home object, got %+v", payload["home"])
	}

	if meta["defaultMessage"] != "FR(Do not treat as FormatJS root)" {
		t.Fatalf("expected translated meta.defaultMessage, got %+v", meta)
	}
	if meta["note"] != "FR(This is nested metadata text)" {
		t.Fatalf("expected translated meta.note, got %+v", meta)
	}
	if home["title"] != "FR(Welcome)" {
		t.Fatalf("expected translated home.title, got %+v", home)
	}
}

func TestRunFlushesEachTargetOnceWithMixedTaskOutcomes(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"ok":"hello","bad":"boom"}`), nil
		case targetPath:
			return []byte(`{"existing":"v"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source == "boom" {
			return "", errors.New("translation failed")
		}
		return strings.ToUpper(req.Source), nil
	}

	writes := 0
	svc.writeFile = func(path string, _ []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		writes++
		return nil
	}

	report, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.Succeeded != 1 || report.Failed != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if writes != 1 {
		t.Fatalf("expected exactly one flush per target, got %d", writes)
	}
}

func TestRunExperimentalContextMemoryGeneratesAndInjectsSharedMemory(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"welcome":"Welcome","bye":"Goodbye"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	seenTranslationContexts := []string{}
	svc.translate = func(ctx context.Context, req translator.Request) (string, error) {
		if strings.HasPrefix(req.Source, "Representative source entries:") {
			translator.SetUsage(ctx, translator.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5})
			return "Terminology: keep onboarding terms consistent.", nil
		}
		seenTranslationContexts = append(seenTranslationContexts, req.Context)
		translator.SetUsage(ctx, translator.Usage{PromptTokens: 10, CompletionTokens: 1, TotalTokens: 11})
		return "FR(" + req.Source + ")", nil
	}

	report, err := svc.Run(context.Background(), Input{
		ExperimentalContextMemory: true,
		ContextMemoryScope:        ContextMemoryScopeFile,
		ContextMemoryMaxChars:     1200,
	})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	if !report.ContextMemoryEnabled {
		t.Fatalf("expected context memory report to be enabled")
	}
	if report.ContextMemoryGenerated != 1 || report.ContextMemoryFallbackGroups != 0 {
		t.Fatalf("unexpected context memory counts: generated=%d fallback=%d", report.ContextMemoryGenerated, report.ContextMemoryFallbackGroups)
	}
	if report.PromptTokens != 23 || report.CompletionTokens != 4 || report.TotalTokens != 27 {
		t.Fatalf("unexpected token totals with context memory: %+v", report.TokenUsage)
	}
	if len(seenTranslationContexts) != 2 {
		t.Fatalf("expected 2 translation requests, got %d", len(seenTranslationContexts))
	}
	for _, got := range seenTranslationContexts {
		if !strings.Contains(got, "Shared memory:") {
			t.Fatalf("expected shared memory in translation context, got %q", got)
		}
	}
}

func TestRunExperimentalContextMemorySummaryFailureFallsBack(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"welcome":"Welcome"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if strings.HasPrefix(req.Source, "Representative source entries:") {
			return "", errors.New("summary unavailable")
		}
		if strings.Contains(req.Context, "Shared memory:") {
			t.Fatalf("did not expect shared memory when summary generation fails, got %q", req.Context)
		}
		return "FR(" + req.Source + ")", nil
	}

	report, err := svc.Run(context.Background(), Input{
		ExperimentalContextMemory: true,
		ContextMemoryScope:        ContextMemoryScopeFile,
		ContextMemoryMaxChars:     1200,
	})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	if report.ContextMemoryGenerated != 0 {
		t.Fatalf("expected no generated context memory, got %d", report.ContextMemoryGenerated)
	}
	if report.ContextMemoryFallbackGroups != 1 {
		t.Fatalf("expected one context memory fallback group, got %d", report.ContextMemoryFallbackGroups)
	}
	if len(report.Warnings) == 0 {
		t.Fatalf("expected warning for context memory generation failure")
	}
	if report.Succeeded != 1 || report.Failed != 0 {
		t.Fatalf("expected translation to continue despite context summary failure, got %+v", report)
	}
}

func TestRunExperimentalContextMemoryRejectsInvalidScope(t *testing.T) {
	svc := newTestService()
	_, err := svc.Run(context.Background(), Input{
		ExperimentalContextMemory: true,
		ContextMemoryScope:        "invalid",
		DryRun:                    true,
	})
	if err == nil || !strings.Contains(err.Error(), "invalid context memory scope") {
		t.Fatalf("expected invalid context memory scope error, got %v", err)
	}
}

func TestRunExperimentalContextMemoryEmitsPhaseEvent(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if strings.HasPrefix(req.Source, "Representative source entries:") {
			return "Terminology: keep greeting terms consistent.", nil
		}
		return "FR(" + req.Source + ")", nil
	}

	events := make([]Event, 0, 8)
	_, err := svc.Run(context.Background(), Input{
		ExperimentalContextMemory: true,
		ContextMemoryScope:        ContextMemoryScopeFile,
		OnEvent: func(event Event) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	foundContextPhase := false
	for _, event := range events {
		if event.Kind == EventPhase && event.Phase == PhaseContextMemory {
			foundContextPhase = true
			break
		}
	}
	if !foundContextPhase {
		t.Fatalf("expected %q phase event, got %+v", PhaseContextMemory, events)
	}
}

func TestRunExperimentalContextMemoryEmitsProgressEvents(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello","bye":"Bye"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if strings.HasPrefix(req.Source, "Representative source entries:") {
			return "Terminology: keep greeting terms consistent.", nil
		}
		return "FR(" + req.Source + ")", nil
	}

	events := make([]Event, 0, 16)
	_, err := svc.Run(context.Background(), Input{
		ExperimentalContextMemory: true,
		ContextMemoryScope:        ContextMemoryScopeFile,
		OnEvent: func(event Event) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	progressEvents := 0
	seenTotal := false
	seenCompletion := false
	seenStart := false
	seenDone := false
	seenProgress := false
	seenFileTarget := false
	for _, event := range events {
		if event.Kind != EventContextMemory {
			continue
		}
		progressEvents++
		if event.ContextMemoryTotal > 0 {
			seenTotal = true
		}
		if event.ContextMemoryProcessed == event.ContextMemoryTotal && event.ContextMemoryTotal > 0 {
			seenCompletion = true
		}
		if event.ContextMemoryState == ContextMemoryStateStart {
			seenStart = true
			if strings.HasSuffix(event.TargetPath, "source.json") {
				seenFileTarget = true
			}
		}
		if event.ContextMemoryState == ContextMemoryStateDone {
			seenDone = true
		}
		if event.ContextMemoryState == ContextMemoryStateProgress {
			seenProgress = true
		}
	}
	if progressEvents == 0 {
		t.Fatalf("expected context memory progress events, got %+v", events)
	}
	if !seenTotal {
		t.Fatalf("expected context memory progress to include total groups, got %+v", events)
	}
	if !seenCompletion {
		t.Fatalf("expected context memory progress completion event, got %+v", events)
	}
	if !seenStart || !seenDone {
		t.Fatalf("expected context memory start/done events for list UI, got %+v", events)
	}
	if !seenProgress {
		t.Fatalf("expected context memory progress events, got %+v", events)
	}
	if !seenFileTarget {
		t.Fatalf("expected context memory file target in events, got %+v", events)
	}
}

func TestRunExperimentalContextMemoryBuildsOneContextPerFile(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello","bye":"Bye"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	contextBuildCalls := 0
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if strings.HasPrefix(req.Source, "Representative source entries:") {
			contextBuildCalls++
			return "Terminology: keep greeting terms consistent.", nil
		}
		return "FR(" + req.Source + ")", nil
	}

	_, err := svc.Run(context.Background(), Input{
		ExperimentalContextMemory: true,
		ContextMemoryScope:        ContextMemoryScopeFile,
	})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if contextBuildCalls != 1 {
		t.Fatalf("expected one context-memory build per file, got %d", contextBuildCalls)
	}
}

func TestInterleaveTasksByContextKeyAlternatesScopes(t *testing.T) {
	in := []Task{
		{EntryKey: "a1", ContextKey: "file-a"},
		{EntryKey: "a2", ContextKey: "file-a"},
		{EntryKey: "a3", ContextKey: "file-a"},
		{EntryKey: "b1", ContextKey: "file-b"},
		{EntryKey: "b2", ContextKey: "file-b"},
	}
	got := interleaveTasksByContextKey(in)
	keys := make([]string, 0, len(got))
	for _, task := range got {
		keys = append(keys, task.EntryKey)
	}
	want := []string{"a1", "b1", "a2", "b2", "a3"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Fatalf("interleaved order = %v, want %v", keys, want)
	}
}
