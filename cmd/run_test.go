package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
