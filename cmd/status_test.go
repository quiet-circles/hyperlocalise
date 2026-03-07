package cmd

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

func TestComputeStatus(t *testing.T) {
	tests := []struct {
		name     string
		entry    storage.Entry
		expected string
	}{
		{
			name: "empty value is untranslated",
			entry: storage.Entry{
				Value: "",
			},
			expected: "untranslated",
		},
		{
			name: "whitespace only is untranslated",
			entry: storage.Entry{
				Value: "   ",
			},
			expected: "untranslated",
		},
		{
			name: "human curated is translated",
			entry: storage.Entry{
				Value: "Hello",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginHuman,
					State:  storage.StateCurated,
				},
			},
			expected: "translated",
		},
		{
			name: "llm draft is needs_review",
			entry: storage.Entry{
				Value: "Hola",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginLLM,
					State:  storage.StateDraft,
				},
			},
			expected: "needs_review",
		},
		{
			name: "llm curated is translated",
			entry: storage.Entry{
				Value: "Hola",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginLLM,
					State:  storage.StateCurated,
				},
			},
			expected: "translated",
		},
		{
			name: "unknown origin with value is translated",
			entry: storage.Entry{
				Value: "Hello",
				Provenance: storage.EntryProvenance{
					Origin: storage.OriginUnknown,
				},
			},
			expected: "translated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeStatus(tt.entry)
			if result != tt.expected {
				t.Errorf("computeStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterByLocaleAndBucket(t *testing.T) {
	entries := []storage.Entry{
		{Key: "a", Locale: "es-ES", Namespace: "ui"},
		{Key: "b", Locale: "es-ES", Namespace: "docs"},
		{Key: "c", Locale: "fr-FR", Namespace: "ui"},
		{Key: "d", Locale: "de-DE", Namespace: "ui"},
		{Key: "e", Locale: "es-ES"},
	}

	t.Run("filter by locale", func(t *testing.T) {
		result := filterByLocaleAndBucket(entries, []string{"es-ES", "fr-FR"}, "", nil)
		if len(result) != 4 {
			t.Errorf("expected 4 entries, got %d", len(result))
		}
	})

	t.Run("filter by locale single", func(t *testing.T) {
		result := filterByLocaleAndBucket(entries, []string{"fr-FR"}, "", nil)
		if len(result) != 1 || result[0].Key != "c" {
			t.Errorf("expected key=c, got %v", result)
		}
	})

	t.Run("no filter returns none when nil", func(t *testing.T) {
		result := filterByLocaleAndBucket(entries, nil, "", nil)
		if len(result) != 0 {
			t.Errorf("expected 0 entries with nil locales, got %d", len(result))
		}
	})

	t.Run("bucket filter excludes namespaces outside bucket", func(t *testing.T) {
		bucketFiles := map[string]struct{}{
			"ui": {},
		}
		result := filterByLocaleAndBucket(entries, []string{"es-ES", "fr-FR"}, "uiOnly", bucketFiles)
		if len(result) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(result))
		}
		for _, entry := range result {
			if entry.Namespace != "ui" {
				t.Fatalf("unexpected namespace in result: %+v", entry)
			}
		}
	})
}

func TestResolveStatusLocales(t *testing.T) {
	cfg := testStatusConfig()

	t.Run("group intersects explicit locales", func(t *testing.T) {
		got, err := resolveStatusLocales(cfg, []string{"fr", "es", "de"}, "europe")
		if err != nil {
			t.Fatalf("resolveStatusLocales() error = %v", err)
		}
		want := []string{"fr", "de"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resolveStatusLocales() = %v, want %v", got, want)
		}
	})

	t.Run("group with no explicit locales uses group targets", func(t *testing.T) {
		got, err := resolveStatusLocales(cfg, nil, "europe")
		if err != nil {
			t.Fatalf("resolveStatusLocales() error = %v", err)
		}
		want := []string{"fr", "de"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resolveStatusLocales() = %v, want %v", got, want)
		}
	})

	t.Run("bucket-only group keeps requested locales", func(t *testing.T) {
		got, err := resolveStatusLocales(cfg, []string{"es"}, "bucket-only")
		if err != nil {
			t.Fatalf("resolveStatusLocales() error = %v", err)
		}
		want := []string{"es"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resolveStatusLocales() = %v, want %v", got, want)
		}
	})

	t.Run("group with no overlap errors", func(t *testing.T) {
		_, err := resolveStatusLocales(cfg, []string{"es"}, "europe")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "no locales matched group") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestStatusBucketFiles(t *testing.T) {
	cfg := testStatusConfig()

	t.Run("unknown bucket returns error", func(t *testing.T) {
		_, err := statusBucketFiles(cfg, "missing")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), `unknown bucket "missing"`) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWriteStatusCSVSortsLocalesPerKey(t *testing.T) {
	entries := []storage.Entry{
		{Key: "welcome", Locale: "fr", Value: "bonjour"},
		{Key: "welcome", Locale: "de", Value: "hallo"},
		{Key: "welcome", Locale: "es", Value: "hola"},
		{Key: "bye", Locale: "es", Value: "adios"},
		{Key: "bye", Locale: "de", Value: "tschuss"},
	}

	var out bytes.Buffer
	if err := writeStatusCSV(&out, entries, "en"); err != nil {
		t.Fatalf("writeStatusCSV() error = %v", err)
	}

	rows, err := csv.NewReader(bytes.NewReader(out.Bytes())).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 6 {
		t.Fatalf("expected 6 rows including header, got %d", len(rows))
	}

	want := [][]string{
		{"key", "namespace", "locale", "status", "origin", "state"},
		{"bye", "", "de", "translated", "", ""},
		{"bye", "", "es", "translated", "", ""},
		{"welcome", "", "de", "translated", "", ""},
		{"welcome", "", "es", "translated", "", ""},
		{"welcome", "", "fr", "translated", "", ""},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("csv rows mismatch\n got: %v\nwant: %v", rows, want)
	}
}

func TestStatusCommandUnknownBucket(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	content := `{
  "locales": {"source":"en","targets":["fr"]},
  "buckets": {"ui":{"files":[{"from":"ui.json","to":"lang/[locale].json"}]}},
  "groups": {"default":{"targets":["fr"],"buckets":["ui"]}},
  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate"}}}
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	cmd.SetArgs([]string{"status", "--config", configPath, "--bucket", "missing"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `unknown bucket "missing"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusCommandBucketFilterUsesLocalstoreNamespace(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "i18n.jsonc")
	langDir := filepath.Join(dir, "lang")
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		t.Fatalf("mkdir lang dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "fr.json"), []byte("{\"hello\":\"bonjour\"}\n"), 0o600); err != nil {
		t.Fatalf("write locale file: %v", err)
	}

	content := `{
  "locales": {"source":"en","targets":["fr"]},
  "buckets": {"ui":{"files":[{"from":"ui/messages.json","to":"` + filepath.ToSlash(filepath.Join(dir, "lang", "[locale].json")) + `"}]}},
  "groups": {"default":{"targets":["fr"],"buckets":["ui"]}},
  "llm": {"profiles":{"default":{"provider":"openai","model":"gpt-4.1-mini","prompt":"Translate"}}}
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"status", "--config", configPath, "--bucket", "ui"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute status command: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "ui/messages.json") {
		t.Fatalf("expected namespace in status output, got: %s", got)
	}
}

func testStatusConfig() *config.I18NConfig {
	return &config.I18NConfig{
		Locales: config.LocaleConfig{
			Source:  "en",
			Targets: []string{"fr", "de", "es"},
		},
		Buckets: map[string]config.BucketConfig{
			"ui": {
				Files: []config.BucketFileMapping{{From: "ui", To: "lang/[locale].json"}},
			},
		},
		Groups: map[string]config.GroupConfig{
			"europe": {
				Targets: []string{"fr", "de"},
			},
			"bucket-only": {
				Buckets: []string{"ui"},
			},
		},
	}
}
