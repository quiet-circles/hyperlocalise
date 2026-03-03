package lockfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadMissingFileReturnsEmptyLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.lock.json")

	f, err := Load(path)
	if err != nil {
		t.Fatalf("load missing lockfile: %v", err)
	}
	if f == nil {
		t.Fatalf("expected lockfile object")
	}
	if f.LocaleStates == nil {
		t.Fatalf("expected initialized locale states map")
	}
	if f.RunCompleted == nil {
		t.Fatalf("expected initialized run completed map")
	}
	if f.RunCheckpoint == nil {
		t.Fatalf("expected initialized run checkpoint map")
	}
	if len(f.LocaleStates) != 0 {
		t.Fatalf("expected empty locale states, got %d", len(f.LocaleStates))
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.lock.json")
	now := time.Unix(1700000000, 0).UTC()

	err := Save(path, File{
		Adapter:    "poeditor",
		ProjectID:  "123",
		LastPullAt: &now,
		LocaleStates: map[string]LocaleCheckpoint{
			"fr": {
				Revision:  "rev1",
				UpdatedAt: &now,
			},
		},
		RunCompleted: map[string]RunCompletion{
			"locales/fr.json::hello": {
				CompletedAt: now,
				SourceHash:  "abc123",
			},
		},
		RunCheckpoint: map[string]RunCheckpoint{
			"locales/fr.json::hello": {
				TargetPath:   "locales/fr.json",
				SourcePath:   "locales/en.json",
				TargetLocale: "fr",
				EntryKey:     "hello",
				Value:        "Bonjour",
				SourceHash:   "abc123",
				UpdatedAt:    now,
			},
		},
	})
	if err != nil {
		t.Fatalf("save lockfile: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("load lockfile: %v", err)
	}
	if got.Adapter != "poeditor" || got.ProjectID != "123" {
		t.Fatalf("unexpected header fields: %+v", got)
	}
	checkpoint, ok := got.LocaleStates["fr"]
	if !ok {
		t.Fatalf("expected fr locale checkpoint")
	}
	if checkpoint.Revision != "rev1" {
		t.Fatalf("unexpected revision: %q", checkpoint.Revision)
	}
	if checkpoint.UpdatedAt == nil || !checkpoint.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected updated_at: %+v", checkpoint.UpdatedAt)
	}
	completion, ok := got.RunCompleted["locales/fr.json::hello"]
	if !ok {
		t.Fatalf("expected run completion")
	}
	if completion.SourceHash != "abc123" {
		t.Fatalf("unexpected source hash: %q", completion.SourceHash)
	}
	checkpointed, ok := got.RunCheckpoint["locales/fr.json::hello"]
	if !ok {
		t.Fatalf("expected run checkpoint")
	}
	if checkpointed.Value != "Bonjour" || checkpointed.SourceHash != "abc123" {
		t.Fatalf("unexpected checkpoint payload: %+v", checkpointed)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.lock.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write invalid lockfile: %v", err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "decode lockfile") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestSaveDefaultPath(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if chErr := os.Chdir(wd); chErr != nil {
			t.Fatalf("restore cwd: %v", chErr)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := Save("", File{}); err != nil {
		t.Fatalf("save default lockfile: %v", err)
	}
	if _, err := os.Stat(DefaultPath); err != nil {
		t.Fatalf("stat default lockfile: %v", err)
	}
}
