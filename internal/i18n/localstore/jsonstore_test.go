package localstore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/syncsvc"
)

func TestJSONStoreReadSnapshotWithoutMetadata(t *testing.T) {
	dir := t.TempDir()
	langDir := filepath.Join(dir, "lang")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatalf("mkdir lang dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "fr.json"), []byte("{\"hello\":\"bonjour\"}\n"), 0o644); err != nil {
		t.Fatalf("write locale file: %v", err)
	}

	store := mustNewStore(t, filepath.Join(dir, "lang", "[locale].json"))
	snap, err := store.ReadSnapshot(context.Background(), syncsvc.LocalReadRequest{Locales: []string{"fr"}})
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if got := len(snap.Entries); got != 1 {
		t.Fatalf("expected 1 entry, got %d", got)
	}
	if got := snap.Entries[0].Provenance.Origin; got != storage.OriginUnknown {
		t.Fatalf("expected origin unknown, got %q", got)
	}
}

func TestJSONStoreApplyPullWritesMetadataSidecar(t *testing.T) {
	dir := t.TempDir()
	store := mustNewStore(t, filepath.Join(dir, "lang", "[locale].json"))

	_, err := store.ApplyPull(context.Background(), syncsvc.ApplyPullPlan{
		Creates: []storage.Entry{{
			Key:    "hello",
			Locale: "fr",
			Value:  "bonjour",
			Provenance: storage.EntryProvenance{
				Origin: storage.OriginHuman,
				State:  storage.StateCurated,
			},
		}},
	})
	if err != nil {
		t.Fatalf("apply pull: %v", err)
	}

	metaPath := filepath.Join(dir, "lang", "fr.meta.json")
	content, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read metadata sidecar: %v", err)
	}

	var meta map[string]map[string]any
	if err := json.Unmarshal(content, &meta); err != nil {
		t.Fatalf("decode metadata sidecar: %v", err)
	}
	if len(meta) != 1 {
		t.Fatalf("expected 1 metadata entry, got %d", len(meta))
	}
}

func TestJSONStoreBuildPushSnapshotUsesSameReadPath(t *testing.T) {
	dir := t.TempDir()
	langDir := filepath.Join(dir, "lang")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatalf("mkdir lang dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "fr.json"), []byte("{\"hello\":\"bonjour\"}\n"), 0o644); err != nil {
		t.Fatalf("write locale file: %v", err)
	}

	store := mustNewStore(t, filepath.Join(dir, "lang", "[locale].json"))
	snap, err := store.BuildPushSnapshot(context.Background(), syncsvc.LocalReadRequest{Locales: []string{"fr"}})
	if err != nil {
		t.Fatalf("build push snapshot: %v", err)
	}
	if got := len(snap.Entries); got != 1 {
		t.Fatalf("expected 1 entry, got %d", got)
	}
	if got := snap.Entries[0].Value; got != "bonjour" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestEntryMetaIDStable(t *testing.T) {
	got := entryMetaID("hello", "")
	if want := "hello\x1f"; got != want {
		t.Fatalf("unexpected entry meta id: got %q want %q", got, want)
	}
}

func mustNewStore(t *testing.T, pattern string) *JSONStore {
	t.Helper()

	store, err := NewJSONStore(&config.I18NConfig{
		Locale: config.LocaleConfig{
			Source:  "en",
			Targets: []string{"fr"},
		},
		Buckets: config.BucketsConfig{
			JSON: &config.BucketConfig{Include: []string{pattern}},
		},
		LLM: config.LLMConfig{
			Default: config.LLMDefaultConfig{
				Provider: "openai",
				Model:    "gpt-4.1-mini",
				Prompt:   "Translate",
			},
		},
	})
	if err != nil {
		t.Fatalf("new json store: %v", err)
	}
	return store
}
