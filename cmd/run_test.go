package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/runsvc"
)

func TestRootHelpIncludesRunCommand(t *testing.T) {
	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run root help: %v", err)
	}

	if !strings.Contains(out.String(), "run") {
		t.Fatalf("expected help to include run command, got %q", out.String())
	}
}

func TestRunDryRunDoesNotWriteTargets(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	targetPath := filepath.Join(dir, "dist", "fr", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(targetPath) + `"}]}},
	  "groups": {"default":{"targets":["fr"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "dry_run=true") {
		t.Fatalf("expected dry-run output, got %q", out.String())
	}
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("expected no target file written in dry-run, stat err=%v", err)
	}
}

func TestRunDryRunPruneReportsCandidates(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	targetPath := filepath.Join(dir, "dist", "fr", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte(`{"hello":"Bonjour","old":"Ancien"}`), 0o600); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(targetPath) + `"}]}},
	  "groups": {"default":{"targets":["fr"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--prune"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command dry-run prune: %v", err)
	}
	if !strings.Contains(out.String(), "prune_candidates=1") {
		t.Fatalf("expected prune candidate summary, got %q", out.String())
	}
	if !strings.Contains(out.String(), "prune target=") {
		t.Fatalf("expected prune candidate details, got %q", out.String())
	}
}

func TestRunDryRunFiltersByBucket(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	uiSourcePath := filepath.Join(dir, "content", "en", "ui.json")
	uiTargetPath := filepath.Join(dir, "dist", "fr", "ui.json")
	marketingSourcePath := filepath.Join(dir, "content", "en", "marketing.json")
	marketingTargetPath := filepath.Join(dir, "dist", "fr", "marketing.json")

	if err := os.MkdirAll(filepath.Dir(uiSourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(uiSourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write ui source file: %v", err)
	}
	if err := os.WriteFile(marketingSourcePath, []byte(`{"sale":"Sale"}`), 0o600); err != nil {
		t.Fatalf("write marketing source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr"]},
	  "buckets": {
	    "ui":{"files":[{"from":"` + filepath.ToSlash(uiSourcePath) + `","to":"` + filepath.ToSlash(uiTargetPath) + `"}]},
	    "marketing":{"files":[{"from":"` + filepath.ToSlash(marketingSourcePath) + `","to":"` + filepath.ToSlash(marketingTargetPath) + `"}]}
	  },
	  "groups": {"default":{"targets":["fr"],"buckets":["ui","marketing"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--bucket", "marketing"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command dry-run filtered bucket: %v", err)
	}
	if !strings.Contains(out.String(), "planned_total=1") {
		t.Fatalf("expected only one planned task, got %q", out.String())
	}
	if strings.Contains(out.String(), filepath.ToSlash(uiTargetPath)) {
		t.Fatalf("expected ui bucket to be filtered out, got %q", out.String())
	}
	if !strings.Contains(out.String(), filepath.ToSlash(marketingTargetPath)) {
		t.Fatalf("expected marketing bucket task, got %q", out.String())
	}
}

func TestRunDryRunFiltersByGroup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	uiSourcePath := filepath.Join(dir, "content", "en", "ui.json")
	uiTargetPath := filepath.Join(dir, "dist", "fr", "ui.json")
	marketingSourcePath := filepath.Join(dir, "content", "en", "marketing.json")
	marketingTargetPath := filepath.Join(dir, "dist", "fr", "marketing.json")

	if err := os.MkdirAll(filepath.Dir(uiSourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(uiSourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write ui source file: %v", err)
	}
	if err := os.WriteFile(marketingSourcePath, []byte(`{"sale":"Sale"}`), 0o600); err != nil {
		t.Fatalf("write marketing source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr"]},
	  "buckets": {
	    "ui":{"files":[{"from":"` + filepath.ToSlash(uiSourcePath) + `","to":"` + filepath.ToSlash(uiTargetPath) + `"}]},
	    "marketing":{"files":[{"from":"` + filepath.ToSlash(marketingSourcePath) + `","to":"` + filepath.ToSlash(marketingTargetPath) + `"}]}
	  },
	  "groups": {
	    "default":{"targets":["fr"],"buckets":["ui"]},
	    "tests":{"targets":["fr"],"buckets":["marketing"]}
	  },
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--group", "tests"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command dry-run filtered group: %v", err)
	}
	if !strings.Contains(out.String(), "planned_total=1") {
		t.Fatalf("expected only one planned task, got %q", out.String())
	}
	if strings.Contains(out.String(), filepath.ToSlash(uiTargetPath)) {
		t.Fatalf("expected default group bucket to be filtered out, got %q", out.String())
	}
	if !strings.Contains(out.String(), filepath.ToSlash(marketingTargetPath)) {
		t.Fatalf("expected tests group task, got %q", out.String())
	}
}

func TestRunRejectsInvalidWorkersValue(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	targetPath := filepath.Join(dir, "dist", "fr", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(targetPath) + `"}]}},
	  "groups": {"default":{"targets":["fr"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--workers", "-1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected workers validation error")
	}
	if !strings.Contains(err.Error(), "invalid --workers value -1: must be >= 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRejectsInvalidProgressMode(t *testing.T) {
	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--progress", "blob"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected invalid progress mode error")
	}
	if !strings.Contains(err.Error(), "invalid --progress value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunReturnsErrorForUnknownBucket(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	targetPath := filepath.Join(dir, "dist", "fr", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(targetPath) + `"}]}},
	  "groups": {"default":{"targets":["fr"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--bucket", "nope"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown bucket error")
	}
	if !strings.Contains(err.Error(), `unknown bucket "nope"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunReturnsErrorForUnknownGroup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	targetPath := filepath.Join(dir, "dist", "fr", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(targetPath) + `"}]}},
	  "groups": {"default":{"targets":["fr"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--group", "nope"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown group error")
	}
	if !strings.Contains(err.Error(), `unknown group "nope"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunProgressOffSkipsProgressEvents(t *testing.T) {
	originalRunFunc := runFunc
	t.Cleanup(func() { runFunc = originalRunFunc })

	receivedOnEvent := false
	runFunc = func(_ context.Context, input runsvc.Input) (runsvc.Report, error) {
		receivedOnEvent = input.OnEvent != nil
		return runsvc.Report{}, nil
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--progress", "off"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run with progress off: %v", err)
	}
	if receivedOnEvent {
		t.Fatalf("expected no progress callback when --progress=off")
	}
	if strings.Contains(out.String(), "progress ") {
		t.Fatalf("expected no progress output, got %q", out.String())
	}
}

func TestRunForceFlagPlumbedToServiceInput(t *testing.T) {
	originalRunFunc := runFunc
	t.Cleanup(func() { runFunc = originalRunFunc })

	receivedForce := false
	runFunc = func(_ context.Context, input runsvc.Input) (runsvc.Report, error) {
		receivedForce = input.Force
		return runsvc.Report{}, nil
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run with force: %v", err)
	}
	if !receivedForce {
		t.Fatalf("expected --force to set runsvc.Input.Force")
	}
}

func TestRunProgressAutoNonTTYKeepsPlainOutput(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	targetPath := filepath.Join(dir, "dist", "fr", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(targetPath) + `"}]}},
	  "groups": {"default":{"targets":["fr"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--progress", "auto"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command dry-run with progress auto: %v", err)
	}
	if !strings.Contains(out.String(), "dry_run=true") {
		t.Fatalf("expected dry-run output, got %q", out.String())
	}
	if strings.Contains(out.String(), "progress ") {
		t.Fatalf("expected plain dry-run output without progress lines, got %q", out.String())
	}
}

func TestRunAllowsExplicitWorkersSetting(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	targetPath := filepath.Join(dir, "dist", "fr", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(targetPath) + `"}]}},
	  "groups": {"default":{"targets":["fr"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--workers", "1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command dry-run with workers: %v", err)
	}
	if !strings.Contains(out.String(), "dry_run=true") {
		t.Fatalf("expected dry-run output, got %q", out.String())
	}
}

func TestRunReturnsErrorOnPartialFailures(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	targetPath := filepath.Join(dir, "dist", "fr", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(targetPath) + `"}]}},
	  "groups": {"default":{"targets":["fr"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected run command to return error on failed task")
	}
	if !strings.Contains(err.Error(), "run completed with failures") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "failed=1") {
		t.Fatalf("expected failed count in output, got %q", out.String())
	}
}
