package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type noopExactCache struct{}

func (noopExactCache) Get(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}

func (noopExactCache) Put(_ context.Context, _ ExactCacheWrite) error {
	return nil
}

type noopTranslationMemory struct{}

func (noopTranslationMemory) Upsert(_ context.Context, _, _, _, _ string, _ float64) error {
	return nil
}

func (noopTranslationMemory) Lookup(_ context.Context, _, _, _ string, _ int) ([]TMResult, error) {
	return nil, nil
}

type noopRetriever struct{}

func (noopRetriever) Retrieve(_ context.Context, _ string, _ int) ([]RAGDocument, error) {
	return nil, nil
}

type exactSQLiteStore struct {
	db       *gorm.DB
	maxItems int
}

func (s *exactSQLiteStore) Get(ctx context.Context, key string) (string, bool, error) {
	var row ExactCacheEntry
	if err := s.db.WithContext(ctx).Where("cache_key = ?", key).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("lookup exact cache: %w", err)
	}
	if err := s.db.WithContext(ctx).Model(&ExactCacheEntry{}).Where("id = ?", row.ID).Update("updated_at", time.Now().UTC()).Error; err != nil {
		// Non-fatal: keep serving valid cache hits even if metadata touch fails.
		// This only affects LRU recency ordering for eviction.
	}
	return row.Value, true, nil
}

func (s *exactSQLiteStore) Put(ctx context.Context, write ExactCacheWrite) error {
	entry := ExactCacheEntry{
		CacheKey:     write.Key,
		SourceLocale: write.SourceLocale,
		TargetLocale: write.TargetLocale,
		Provider:     write.Provider,
		Model:        write.Model,
		SourceHash:   write.SourceHash,
		Value:        write.Value,
	}
	if err := s.db.WithContext(ctx).Where("cache_key = ?", write.Key).Assign(entry).FirstOrCreate(&entry).Error; err != nil {
		return fmt.Errorf("upsert exact cache: %w", err)
	}
	if s.maxItems > 0 {
		if err := s.evictIfNeeded(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *exactSQLiteStore) evictIfNeeded(ctx context.Context) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&ExactCacheEntry{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count exact cache entries: %w", err)
	}
	overflow := int(count) - s.maxItems
	if overflow <= 0 {
		return nil
	}
	var oldIDs []uint
	if err := s.db.WithContext(ctx).Model(&ExactCacheEntry{}).Order("updated_at asc").Limit(overflow).Pluck("id", &oldIDs).Error; err != nil {
		return fmt.Errorf("select exact cache eviction candidates: %w", err)
	}
	if len(oldIDs) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Delete(&ExactCacheEntry{}, oldIDs).Error; err != nil {
		return fmt.Errorf("evict exact cache entries: %w", err)
	}
	return nil
}

type tmSQLiteStore struct {
	db *gorm.DB
}

func (s *tmSQLiteStore) Upsert(ctx context.Context, sourceLocale, targetLocale, sourceText, translatedText string, score float64) error {
	entry := TranslationMemoryEntry{
		SourceLocale:   sourceLocale,
		TargetLocale:   targetLocale,
		SourceText:     sourceText,
		TranslatedText: translatedText,
		Score:          score,
	}
	query := s.db.WithContext(ctx).
		Where("source_locale = ? AND target_locale = ? AND source_text = ?", sourceLocale, targetLocale, sourceText).
		Assign(entry)
	if err := query.FirstOrCreate(&entry).Error; err != nil {
		return fmt.Errorf("upsert translation memory entry: %w", err)
	}
	return nil
}

func (s *tmSQLiteStore) Lookup(ctx context.Context, sourceLocale, targetLocale, sourceText string, limit int) ([]TMResult, error) {
	if limit <= 0 {
		limit = 5
	}
	var rows []TranslationMemoryEntry
	if err := s.db.WithContext(ctx).
		Where("source_locale = ? AND target_locale = ? AND source_text = ?", sourceLocale, targetLocale, sourceText).
		Order("score desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("lookup translation memory entries: %w", err)
	}
	results := make([]TMResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, TMResult{SourceText: row.SourceText, TranslatedText: row.TranslatedText, Score: row.Score})
	}
	return results, nil
}
