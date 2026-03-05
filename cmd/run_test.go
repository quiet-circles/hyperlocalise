package cmd

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestRunDryRunPruneFiltersByTargetLocale(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	frTargetPath := filepath.Join(dir, "dist", "fr", "strings.json")
	deTargetPath := filepath.Join(dir, "dist", "de", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(frTargetPath), 0o755); err != nil {
		t.Fatalf("create fr target dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(deTargetPath), 0o755); err != nil {
		t.Fatalf("create de target dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.WriteFile(frTargetPath, []byte(`{"hello":"Bonjour","old":"Ancien"}`), 0o600); err != nil {
		t.Fatalf("write fr target file: %v", err)
	}
	if err := os.WriteFile(deTargetPath, []byte(`{"hello":"Hallo","old":"Alt"}`), 0o600); err != nil {
		t.Fatalf("write de target file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr","de"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(filepath.Join(dir, "dist", "{{target}}", "strings.json")) + `"}]}},
	  "groups": {"default":{"targets":["fr","de"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--prune", "--target-locale", "de"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command dry-run prune filtered target locale: %v", err)
	}
	if !strings.Contains(out.String(), "planned_total=1") {
		t.Fatalf("expected only one planned task, got %q", out.String())
	}
	if !strings.Contains(out.String(), "prune_candidates=1") {
		t.Fatalf("expected one prune candidate, got %q", out.String())
	}
	if !strings.Contains(out.String(), "prune target="+filepath.ToSlash(deTargetPath)+" key=old") {
		t.Fatalf("expected de prune candidate, got %q", out.String())
	}
	if strings.Contains(out.String(), "prune target="+filepath.ToSlash(frTargetPath)+" key=old") {
		t.Fatalf("expected fr prune candidate to be filtered out, got %q", out.String())
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

func TestRunDryRunFiltersByTargetLocale(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	frTargetPath := filepath.Join(dir, "dist", "fr", "strings.json")
	deTargetPath := filepath.Join(dir, "dist", "de", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr","de"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(filepath.Join(dir, "dist", "{{target}}", "strings.json")) + `"}]}},
	  "groups": {"default":{"targets":["fr","de"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--target-locale", "de"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command dry-run filtered target locale: %v", err)
	}
	if !strings.Contains(out.String(), "planned_total=1") {
		t.Fatalf("expected only one planned task, got %q", out.String())
	}
	if strings.Contains(out.String(), filepath.ToSlash(frTargetPath)) {
		t.Fatalf("expected fr locale to be filtered out, got %q", out.String())
	}
	if !strings.Contains(out.String(), filepath.ToSlash(deTargetPath)) {
		t.Fatalf("expected de locale task, got %q", out.String())
	}
}

func TestRunTargetLocaleRespectsGroupTargets(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr","de"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(filepath.Join(dir, "dist", "{{target}}", "strings.json")) + `"}]}},
	  "groups": {
	    "default":{"targets":["fr"],"buckets":["ui"]},
	    "secondary":{"targets":["de"],"buckets":["ui"]}
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
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--group", "default", "--target-locale", "de"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command dry-run group filtered target locale: %v", err)
	}
	if !strings.Contains(out.String(), "planned_total=0") {
		t.Fatalf("expected no planned tasks when locale is outside group targets, got %q", out.String())
	}
}

func TestRunRejectsUnknownTargetLocale(t *testing.T) {
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
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--target-locale", "de"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown target locale error")
	}
	if !strings.Contains(err.Error(), `planning tasks: unknown target locale "de"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRejectsEmptyTargetLocale(t *testing.T) {
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
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--target-locale", ""})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected empty target locale error")
	}
	if !strings.Contains(err.Error(), "invalid --target-locale value: must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRejectsWhitespaceTargetLocale(t *testing.T) {
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
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--target-locale", "   "})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected whitespace target locale error")
	}
	if !strings.Contains(err.Error(), "invalid --target-locale value: must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRejectsMixedEmptyTargetLocaleValue(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	sourcePath := filepath.Join(dir, "content", "en", "strings.json")
	frTargetPath := filepath.Join(dir, "dist", "fr", "strings.json")
	deTargetPath := filepath.Join(dir, "dist", "de", "strings.json")

	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	content := `{
	  "locales": {"source":"en","targets":["fr","de"]},
	  "buckets": {"ui":{"files":[{"from":"` + filepath.ToSlash(sourcePath) + `","to":"` + filepath.ToSlash(filepath.Join(dir, "dist", "{{target}}", "strings.json")) + `"}]}},
	  "groups": {"default":{"targets":["fr","de"],"buckets":["ui"]}},
	  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate {{input}}"}}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--config", configPath, "--dry-run", "--target-locale", ",de"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected mixed empty target locale error")
	}
	if !strings.Contains(err.Error(), "invalid --target-locale value: must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.String(), filepath.ToSlash(frTargetPath)) || strings.Contains(out.String(), filepath.ToSlash(deTargetPath)) {
		t.Fatalf("expected run to fail before planning tasks, got %q", out.String())
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

func TestRunExperimentalContextMemoryFlagsPlumbedToServiceInput(t *testing.T) {
	originalRunFunc := runFunc
	t.Cleanup(func() { runFunc = originalRunFunc })

	var gotInput runsvc.Input
	runFunc = func(_ context.Context, input runsvc.Input) (runsvc.Report, error) {
		gotInput = input
		return runsvc.Report{}, nil
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{
		"run",
		"--experimental-context-memory",
		"--context-memory-scope", "bucket",
		"--context-memory-max-chars", "900",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run with context memory flags: %v", err)
	}
	if !gotInput.ExperimentalContextMemory {
		t.Fatalf("expected --experimental-context-memory to set runsvc.Input.ExperimentalContextMemory")
	}
	if gotInput.ContextMemoryScope != runsvc.ContextMemoryScopeBucket {
		t.Fatalf("unexpected context memory scope: %q", gotInput.ContextMemoryScope)
	}
	if gotInput.ContextMemoryMaxChars != 900 {
		t.Fatalf("unexpected context memory max chars: %d", gotInput.ContextMemoryMaxChars)
	}
}

func TestRunRejectsInvalidContextMemoryScope(t *testing.T) {
	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--context-memory-scope", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected invalid context memory scope error")
	}
	if !strings.Contains(err.Error(), "invalid --context-memory-scope value") {
		t.Fatalf("unexpected error: %v", err)
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

func TestRunWritesMachineReadableArtifact(t *testing.T) {
	originalRunFunc := runFunc
	t.Cleanup(func() { runFunc = originalRunFunc })

	dir := t.TempDir()
	reportPath := filepath.Join(dir, "reports", "run-report.json")
	runFunc = func(_ context.Context, _ runsvc.Input) (runsvc.Report, error) {
		return runsvc.Report{
			PlannedTotal:    2,
			ExecutableTotal: 2,
			Succeeded:       2,
			TokenUsage:      runsvc.TokenUsage{PromptTokens: 123, CompletionTokens: 45, TotalTokens: 168},
			LocaleUsage:     map[string]runsvc.TokenUsage{"fr": {PromptTokens: 100, CompletionTokens: 30, TotalTokens: 130}},
			Batches:         []runsvc.BatchUsage{{TargetLocale: "fr", TargetPath: "dist/fr/strings.json", EntryKey: "hello", TokenUsage: runsvc.TokenUsage{PromptTokens: 100, CompletionTokens: 30, TotalTokens: 130}}},
		}, nil
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"run", "--output", reportPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run with output artifact: %v", err)
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report artifact: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("decode report artifact: %v", err)
	}
	if payload["totalTokens"] != float64(168) {
		t.Fatalf("expected totalTokens in report artifact, got %+v", payload)
	}
	localeUsage, ok := payload["localeUsage"].(map[string]any)
	if !ok {
		t.Fatalf("expected localeUsage object, got %+v", payload["localeUsage"])
	}
	if _, ok := localeUsage["fr"]; !ok {
		t.Fatalf("expected fr locale usage in report artifact, got %+v", localeUsage)
	}
}
