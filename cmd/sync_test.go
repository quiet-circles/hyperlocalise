package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncPullRequiresStorageConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	content := `{
	  "locale": {"source":"en","targets":["fr"]},
	  "buckets": {"json":{"include":["lang/[locale].json"]}},
	  "llm": {"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate"}}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"sync", "pull", "--config", configPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected sync pull error without storage config")
	}
	if !strings.Contains(err.Error(), "storage config is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
