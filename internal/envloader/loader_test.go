package envloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFilesInDirLoadsFromDotEnv(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".env"), "OPENAI_API_KEY=from-env\n")

	withUnsetEnv(t, "OPENAI_API_KEY")

	if err := LoadFilesInDir(dir); err != nil {
		t.Fatalf("load env files: %v", err)
	}

	got := os.Getenv("OPENAI_API_KEY")
	if got != "from-env" {
		t.Fatalf("OPENAI_API_KEY: got %q want %q", got, "from-env")
	}
}

func TestLoadFilesInDirPrefersDotEnvLocalOverDotEnv(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".env"), "OPENAI_API_KEY=from-env\n")
	mustWriteFile(t, filepath.Join(dir, ".env.local"), "OPENAI_API_KEY=from-env-local\n")

	withUnsetEnv(t, "OPENAI_API_KEY")

	if err := LoadFilesInDir(dir); err != nil {
		t.Fatalf("load env files: %v", err)
	}

	got := os.Getenv("OPENAI_API_KEY")
	if got != "from-env-local" {
		t.Fatalf("OPENAI_API_KEY: got %q want %q", got, "from-env-local")
	}
}

func TestLoadFilesInDirDoesNotOverrideExistingEnv(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".env"), "OPENAI_API_KEY=from-env\n")
	mustWriteFile(t, filepath.Join(dir, ".env.local"), "OPENAI_API_KEY=from-env-local\n")

	t.Setenv("OPENAI_API_KEY", "already-exported")

	if err := LoadFilesInDir(dir); err != nil {
		t.Fatalf("load env files: %v", err)
	}

	got := os.Getenv("OPENAI_API_KEY")
	if got != "already-exported" {
		t.Fatalf("OPENAI_API_KEY: got %q want %q", got, "already-exported")
	}
}

func TestLoadFilesInDirIgnoresMissingFiles(t *testing.T) {
	dir := t.TempDir()

	if err := LoadFilesInDir(dir); err != nil {
		t.Fatalf("load env files: %v", err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func withUnsetEnv(t *testing.T, key string) {
	t.Helper()

	orig, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset env %s: %v", key, err)
	}

	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, orig)
			return
		}

		_ = os.Unsetenv(key)
	})
}
