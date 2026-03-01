package runsvc

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/lockfile"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translationfileparser"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

func TestRunFailsWhenSourceFileMissing(t *testing.T) {
	svc := newTestService()
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig("/tmp/missing.json", "/tmp/out.json")
		return &cfg, nil
	}
	svc.readFile = func(_ string) ([]byte, error) {
		return nil, filepath.ErrBadPattern
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

func TestRunContinueOnErrorReturnsPartialFailureReport(t *testing.T) {
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
	if len(report.Failures) != 1 || report.Failures[0].EntryKey != "bad" {
		t.Fatalf("unexpected failures: %+v", report.Failures)
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

func TestRunLockWriterPersistsEachSuccess(t *testing.T) {
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
	svc.saveLock = func(_ string, f lockfile.File) error {
		lockWrites++
		seenSizes = append(seenSizes, len(f.RunCompleted))
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
	if lockWrites != 3 {
		t.Fatalf("expected one lock write per success, got %d", lockWrites)
	}
	if seenSizes[len(seenSizes)-1] != 3 {
		t.Fatalf("expected final lock map size 3, got %v", seenSizes)
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
