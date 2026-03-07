package cache

import (
	"context"
	"math"
	"path/filepath"
	"sync"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/config"
)

func TestTMUpsertPersistsProvenanceAndSource(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	if err := svc.L2.Upsert(context.Background(), TMWrite{
		SourceLocale:   "en",
		TargetLocale:   "fr",
		SourceText:     "Hello",
		TranslatedText: "Bonjour",
		Score:          0.8,
		Metadata: TMMetadata{
			Provenance: TMProvenanceCurated,
			Source:     TMSourceSyncPull,
		},
	}); err != nil {
		t.Fatalf("upsert tm entry: %v", err)
	}

	var row TranslationMemoryEntry
	if err := svc.db.Where("source_locale = ? AND target_locale = ? AND source_text = ?", "en", "fr", "Hello").Take(&row).Error; err != nil {
		t.Fatalf("load tm row: %v", err)
	}
	if row.Provenance != TMProvenanceCurated {
		t.Fatalf("provenance=%q, want %q", row.Provenance, TMProvenanceCurated)
	}
	if row.Source != TMSourceSyncPull {
		t.Fatalf("source=%q, want %q", row.Source, TMSourceSyncPull)
	}
}

func TestTMLookupReturnsSimilarityAndMetadata(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	if err := svc.L2.Upsert(context.Background(), TMWrite{
		SourceLocale:   "en",
		TargetLocale:   "fr",
		SourceText:     "Welcome to our app",
		TranslatedText: "Bienvenue dans notre application",
		Score:          0.7,
		Metadata: TMMetadata{
			Provenance: TMProvenanceLLM,
			Source:     TMSourceRun,
		},
	}); err != nil {
		t.Fatalf("upsert tm entry: %v", err)
	}

	results, err := svc.L2.Lookup(context.Background(), "en", "fr", "Welcome to our application", 1)
	if err != nil {
		t.Fatalf("lookup tm entry: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d, want 1", len(results))
	}
	got := results[0]
	if got.Similarity <= 0 || got.Similarity >= 1 {
		t.Fatalf("similarity=%f, want between 0 and 1", got.Similarity)
	}
	if got.Metadata.Provenance != TMProvenanceLLM {
		t.Fatalf("provenance=%q, want %q", got.Metadata.Provenance, TMProvenanceLLM)
	}
	if got.Metadata.Source != TMSourceRun {
		t.Fatalf("source=%q, want %q", got.Metadata.Source, TMSourceRun)
	}
}

func TestTMLookupPrefersCuratedWhenSimilarityComparable(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	entries := []TMWrite{
		{
			SourceLocale:   "en",
			TargetLocale:   "fr",
			SourceText:     "Accept terms.",
			TranslatedText: "Accepter les conditions (brouillon)",
			Score:          0.95,
			Metadata: TMMetadata{
				Provenance: TMProvenanceDraft,
				Source:     TMSourceRun,
			},
		},
		{
			SourceLocale:   "en",
			TargetLocale:   "fr",
			SourceText:     "Accept term",
			TranslatedText: "Accepter les conditions (curated)",
			Score:          0.70,
			Metadata: TMMetadata{
				Provenance: TMProvenanceCurated,
				Source:     TMSourceSyncPull,
			},
		},
	}
	for _, entry := range entries {
		if err := svc.L2.Upsert(context.Background(), entry); err != nil {
			t.Fatalf("upsert tm entry: %v", err)
		}
	}

	results, err := svc.L2.Lookup(context.Background(), "en", "fr", "Accept terms", 1)
	if err != nil {
		t.Fatalf("lookup tm entry: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d, want 1", len(results))
	}
	if results[0].Metadata.Provenance != TMProvenanceCurated {
		t.Fatalf("provenance=%q, want %q", results[0].Metadata.Provenance, TMProvenanceCurated)
	}
}

func TestTMLookupAutoAcceptThreshold(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	if err := svc.L2.Upsert(context.Background(), TMWrite{
		SourceLocale:   "en",
		TargetLocale:   "fr",
		SourceText:     "Hello",
		TranslatedText: "Bonjour",
		Score:          0.8,
		Metadata: TMMetadata{
			Provenance: TMProvenanceCurated,
			Source:     TMSourceSyncPull,
		},
	}); err != nil {
		t.Fatalf("upsert tm entry: %v", err)
	}
	if err := svc.L2.Upsert(context.Background(), TMWrite{
		SourceLocale:   "en",
		TargetLocale:   "fr",
		SourceText:     "Hallo",
		TranslatedText: "Salut",
		Score:          0.8,
		Metadata: TMMetadata{
			Provenance: TMProvenanceDraft,
			Source:     TMSourceRun,
		},
	}); err != nil {
		t.Fatalf("upsert tm entry: %v", err)
	}

	results, err := svc.L2.Lookup(context.Background(), "en", "fr", "Hello", 2)
	if err != nil {
		t.Fatalf("lookup tm entries: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results=%d, want 2", len(results))
	}
	if !results[0].AutoAccepted {
		t.Fatal("expected best match to be auto-accepted")
	}
	if results[1].AutoAccepted {
		t.Fatal("expected weaker match to be suggest-only")
	}
}

func TestTMLookupUsesConfiguredAutoAcceptThreshold(t *testing.T) {
	t.Parallel()

	svc := newTMTestServiceWithThreshold(t, 0.95)
	if err := svc.L2.Upsert(context.Background(), TMWrite{
		SourceLocale:   "en",
		TargetLocale:   "fr",
		SourceText:     "abcdefghij",
		TranslatedText: "alpha",
		Score:          0.9,
		Metadata: TMMetadata{
			Provenance: TMProvenanceCurated,
			Source:     TMSourceSyncPull,
		},
	}); err != nil {
		t.Fatalf("upsert tm entry: %v", err)
	}

	results, err := svc.L2.Lookup(context.Background(), "en", "fr", "abcdefghiX", 1)
	if err != nil {
		t.Fatalf("lookup tm entry: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d, want 1", len(results))
	}
	if math.Abs(results[0].Similarity-0.9) > 0.000001 {
		t.Fatalf("similarity=%f, want approximately 0.9", results[0].Similarity)
	}
	if results[0].AutoAccepted {
		t.Fatal("expected suggestion-only when similarity below configured threshold")
	}
}

func TestTMLookupAutoAcceptsAtBoundary(t *testing.T) {
	t.Parallel()

	svc := newTMTestServiceWithThreshold(t, 0.9)
	if err := svc.L2.Upsert(context.Background(), TMWrite{
		SourceLocale:   "en",
		TargetLocale:   "fr",
		SourceText:     "abcdefghij",
		TranslatedText: "alpha",
		Score:          0.9,
		Metadata: TMMetadata{
			Provenance: TMProvenanceCurated,
			Source:     TMSourceSyncPull,
		},
	}); err != nil {
		t.Fatalf("upsert tm entry: %v", err)
	}

	results, err := svc.L2.Lookup(context.Background(), "en", "fr", "abcdefghiX", 1)
	if err != nil {
		t.Fatalf("lookup tm entry: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d, want 1", len(results))
	}
	if !results[0].AutoAccepted {
		t.Fatal("expected auto-accept at threshold boundary")
	}
}

func TestTMUpsertNormalizesMissingMetadataToUnknown(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	if err := svc.L2.Upsert(context.Background(), TMWrite{
		SourceLocale:   "en",
		TargetLocale:   "fr",
		SourceText:     "Hello",
		TranslatedText: "Bonjour",
		Score:          0.7,
	}); err != nil {
		t.Fatalf("upsert tm entry: %v", err)
	}

	results, err := svc.L2.Lookup(context.Background(), "en", "fr", "Hello", 1)
	if err != nil {
		t.Fatalf("lookup tm entry: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d, want 1", len(results))
	}
	if results[0].Metadata.Provenance != TMProvenanceUnknown {
		t.Fatalf("provenance=%q, want %q", results[0].Metadata.Provenance, TMProvenanceUnknown)
	}
	if results[0].Metadata.Source != TMSourceUnknown {
		t.Fatalf("source=%q, want %q", results[0].Metadata.Source, TMSourceUnknown)
	}
}

func TestTMLookupPrefersDraftOverUnknownWhenSimilarityComparable(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	entries := []TMWrite{
		{
			SourceLocale:   "en",
			TargetLocale:   "fr",
			SourceText:     "Accept terms.",
			TranslatedText: "Inconnu",
			Score:          0.95,
			Metadata: TMMetadata{
				Provenance: TMProvenanceUnknown,
				Source:     TMSourceLegacy,
			},
		},
		{
			SourceLocale:   "en",
			TargetLocale:   "fr",
			SourceText:     "Accept term",
			TranslatedText: "Brouillon",
			Score:          0.70,
			Metadata: TMMetadata{
				Provenance: TMProvenanceDraft,
				Source:     TMSourceRun,
			},
		},
	}
	for _, entry := range entries {
		if err := svc.L2.Upsert(context.Background(), entry); err != nil {
			t.Fatalf("upsert tm entry: %v", err)
		}
	}

	results, err := svc.L2.Lookup(context.Background(), "en", "fr", "Accept terms", 1)
	if err != nil {
		t.Fatalf("lookup tm entry: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d, want 1", len(results))
	}
	if results[0].Metadata.Provenance != TMProvenanceDraft {
		t.Fatalf("provenance=%q, want %q", results[0].Metadata.Provenance, TMProvenanceDraft)
	}
}

func TestTMLookupPrefersCuratedOverLLMWhenSimilarityComparable(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	entries := []TMWrite{
		{
			SourceLocale:   "en",
			TargetLocale:   "fr",
			SourceText:     "Privacy preference A",
			TranslatedText: "Préférence confidentialité (llm)",
			Score:          0.95,
			Metadata: TMMetadata{
				Provenance: TMProvenanceLLM,
				Source:     TMSourceRun,
			},
		},
		{
			SourceLocale:   "en",
			TargetLocale:   "fr",
			SourceText:     "Privacy preference B",
			TranslatedText: "Préférences confidentialité (curated)",
			Score:          0.7,
			Metadata: TMMetadata{
				Provenance: TMProvenanceCurated,
				Source:     TMSourceSyncPull,
			},
		},
	}
	for _, entry := range entries {
		if err := svc.L2.Upsert(context.Background(), entry); err != nil {
			t.Fatalf("upsert tm entry: %v", err)
		}
	}

	results, err := svc.L2.Lookup(context.Background(), "en", "fr", "Privacy preference C", 1)
	if err != nil {
		t.Fatalf("lookup tm entry: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d, want 1", len(results))
	}
	if results[0].Metadata.Provenance != TMProvenanceCurated {
		t.Fatalf("provenance=%q, want %q", results[0].Metadata.Provenance, TMProvenanceCurated)
	}
}

func TestTMUpsertRejectsInvalidMetadataValues(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	err := svc.L2.Upsert(context.Background(), TMWrite{
		SourceLocale:   "en",
		TargetLocale:   "fr",
		SourceText:     "Hello",
		TranslatedText: "Bonjour",
		Score:          0.7,
		Metadata: TMMetadata{
			Provenance: "bad_value",
			Source:     TMSourceRun,
		},
	})
	if err == nil {
		t.Fatal("expected invalid provenance to fail upsert")
	}
}

func TestTMUpsertRejectsInvalidSourceValues(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	err := svc.L2.Upsert(context.Background(), TMWrite{
		SourceLocale:   "en",
		TargetLocale:   "fr",
		SourceText:     "Hello",
		TranslatedText: "Bonjour",
		Score:          0.7,
		Metadata: TMMetadata{
			Provenance: TMProvenanceCurated,
			Source:     "bad_source",
		},
	})
	if err == nil {
		t.Fatal("expected invalid source to fail upsert")
	}
}

func TestTMUpsertSupportsLegacyFlatMetadataFields(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	if err := svc.L2.Upsert(context.Background(), TMWrite{
		SourceLocale:   "en",
		TargetLocale:   "fr",
		SourceText:     "Legacy",
		TranslatedText: "Hérité",
		Score:          0.7,
		Provenance:     TMProvenanceCurated,
		Source:         TMSourceLegacy,
	}); err != nil {
		t.Fatalf("upsert tm entry with legacy fields: %v", err)
	}
	results, err := svc.L2.Lookup(context.Background(), "en", "fr", "Legacy", 1)
	if err != nil {
		t.Fatalf("lookup tm entry: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results=%d, want 1", len(results))
	}
	if results[0].Metadata.Provenance != TMProvenanceCurated || results[0].Metadata.Source != TMSourceLegacy {
		t.Fatalf("unexpected metadata: %+v", results[0].Metadata)
	}
	if results[0].Provenance != TMProvenanceCurated || results[0].Source != TMSourceLegacy {
		t.Fatalf("unexpected legacy metadata projection: provenance=%q source=%q", results[0].Provenance, results[0].Source)
	}
}

func TestTMUpsertConcurrentWritersKeepSingleCanonicalRow(t *testing.T) {
	t.Parallel()

	svc := newTMTestService(t)
	const writers = 24
	const writesPerWriter = 25

	var wg sync.WaitGroup
	errCh := make(chan error, writers*writesPerWriter)
	for i := 0; i < writers; i++ {
		writerID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < writesPerWriter; j++ {
				write := TMWrite{
					SourceLocale:   "en",
					TargetLocale:   "fr",
					SourceText:     "Concurrent key",
					TranslatedText: "Bonjour",
					Score:          0.8,
					Metadata: TMMetadata{
						Provenance: TMProvenanceCurated,
						Source:     TMSourceSyncPull,
					},
				}
				if (writerID+j)%2 == 0 {
					write.TranslatedText = "Salut"
					write.Metadata.Provenance = TMProvenanceDraft
					write.Metadata.Source = TMSourceRun
				}
				if err := svc.L2.Upsert(context.Background(), write); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent upsert failed: %v", err)
	}

	var rows []TranslationMemoryEntry
	if err := svc.db.Where("source_locale = ? AND target_locale = ? AND source_text = ?", "en", "fr", "Concurrent key").Find(&rows).Error; err != nil {
		t.Fatalf("load concurrent tm rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}

	got := rows[0]
	if got.TranslatedText != "Bonjour" && got.TranslatedText != "Salut" {
		t.Fatalf("unexpected translated_text=%q", got.TranslatedText)
	}
	if got.Provenance != TMProvenanceCurated && got.Provenance != TMProvenanceDraft {
		t.Fatalf("unexpected provenance=%q", got.Provenance)
	}
	if got.Source != TMSourceSyncPull && got.Source != TMSourceRun {
		t.Fatalf("unexpected source=%q", got.Source)
	}
}

func newTMTestService(t *testing.T) *Service {
	return newTMTestServiceWithThreshold(t, config.DefaultCacheL2AutoAcceptThreshold)
}

func newTMTestServiceWithThreshold(t *testing.T, threshold float64) *Service {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "cache.sqlite")
	svc, err := NewFromConfig(config.CacheConfig{
		Enabled: true,
		DBPath:  dbPath,
		SQLite:  config.CacheSQLiteConfig{MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: 5},
		L2: config.CacheTierConfig{
			Enabled:             true,
			AutoAcceptThreshold: threshold,
		},
	})
	if err != nil {
		t.Fatalf("new cache service: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}
