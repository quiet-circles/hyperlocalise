package cmd

import (
	"testing"

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
	}

	t.Run("filter by locale", func(t *testing.T) {
		result := filterByLocaleAndBucket(entries, []string{"es-ES", "fr-FR"}, "", nil)
		if len(result) != 3 {
			t.Errorf("expected 3 entries, got %d", len(result))
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
}
