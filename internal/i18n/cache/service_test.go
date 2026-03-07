package cache

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"gorm.io/gorm"
)

func TestNewFromConfigDisabled(t *testing.T) {
	t.Parallel()

	svc, err := NewFromConfig(config.CacheConfig{Enabled: false})
	if err != nil {
		t.Fatalf("new cache service: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.db != nil {
		t.Fatal("expected db to be nil when cache is disabled")
	}
}

func TestNewFromConfigEnabledMigratesSchema(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "cache", "cache.sqlite")
	svc, err := NewFromConfig(config.CacheConfig{
		Enabled: true,
		DBPath:  dbPath,
		SQLite: config.CacheSQLiteConfig{
			MaxOpenConns:    1,
			MaxIdleConns:    1,
			ConnMaxLifetime: 5,
		},
		L1: config.CacheTierConfig{Enabled: true, MaxItems: 10},
		L2: config.CacheTierConfig{Enabled: true},
	})
	if err != nil {
		t.Fatalf("new cache service: %v", err)
	}
	t.Cleanup(func() {
		_ = svc.Close()
	})

	if !svc.db.Migrator().HasTable(&ExactCacheEntry{}) {
		t.Fatal("expected exact cache table")
	}
	if !svc.db.Migrator().HasTable(&TranslationMemoryEntry{}) {
		t.Fatal("expected translation memory table")
	}
}

func TestNewFromConfigMigrationIsIdempotent(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "cache.sqlite")
	cfg := config.CacheConfig{
		Enabled: true,
		DBPath:  dbPath,
		SQLite:  config.CacheSQLiteConfig{MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: 5},
		L1:      config.CacheTierConfig{Enabled: true, MaxItems: 10},
		L2:      config.CacheTierConfig{Enabled: true},
	}

	svc1, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("first new cache service: %v", err)
	}
	if err := svc1.Close(); err != nil {
		t.Fatalf("close first cache service: %v", err)
	}

	svc2, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("second new cache service: %v", err)
	}
	if err := svc2.Close(); err != nil {
		t.Fatalf("close second cache service: %v", err)
	}
}

func TestL1GetUpdatesHitMetadataTimestamp(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "cache.sqlite")
	svc, err := NewFromConfig(config.CacheConfig{
		Enabled: true,
		DBPath:  dbPath,
		SQLite:  config.CacheSQLiteConfig{MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: 5},
		L1:      config.CacheTierConfig{Enabled: true, MaxItems: 10},
	})
	if err != nil {
		t.Fatalf("new cache service: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	if err := svc.L1.Put(context.Background(), ExactCacheWrite{
		Key:          "k1",
		Value:        "v1",
		SourceLocale: "en",
		TargetLocale: "fr",
		Provider:     "openai",
		Model:        "gpt-4.1-mini",
		SourceHash:   "hash",
	}); err != nil {
		t.Fatalf("seed cache entry: %v", err)
	}
	stale := time.Now().UTC().Add(-2 * time.Hour)
	if err := svc.db.Model(&ExactCacheEntry{}).Where("cache_key = ?", "k1").Update("updated_at", stale).Error; err != nil {
		t.Fatalf("set stale updated_at: %v", err)
	}

	if _, hit, err := svc.L1.Get(context.Background(), "k1"); err != nil {
		t.Fatalf("lookup cache entry: %v", err)
	} else if !hit {
		t.Fatal("expected cache hit")
	}

	var row ExactCacheEntry
	if err := svc.db.Where("cache_key = ?", "k1").Take(&row).Error; err != nil {
		t.Fatalf("reload cache entry: %v", err)
	}
	if !row.UpdatedAt.After(stale) {
		t.Fatalf("expected updated_at to move forward on hit, stale=%s got=%s", stale, row.UpdatedAt)
	}
}

func TestL1PutPersistsMetadataColumns(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "cache.sqlite")
	svc, err := NewFromConfig(config.CacheConfig{
		Enabled: true,
		DBPath:  dbPath,
		SQLite:  config.CacheSQLiteConfig{MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: 5},
		L1:      config.CacheTierConfig{Enabled: true, MaxItems: 10},
	})
	if err != nil {
		t.Fatalf("new cache service: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	if err := svc.L1.Put(context.Background(), ExactCacheWrite{
		Key:          "k-meta",
		Value:        "v-meta",
		SourceLocale: "en-US",
		TargetLocale: "fr-FR",
		Provider:     "openai",
		Model:        "gpt-5.2",
		SourceHash:   "source-hash",
	}); err != nil {
		t.Fatalf("put cache entry: %v", err)
	}

	var row ExactCacheEntry
	if err := svc.db.Where("cache_key = ?", "k-meta").Take(&row).Error; err != nil {
		t.Fatalf("load cache row: %v", err)
	}
	if row.SourceLocale != "en-US" || row.TargetLocale != "fr-FR" || row.Provider != "openai" || row.Model != "gpt-5.2" || row.SourceHash != "source-hash" {
		t.Fatalf("unexpected metadata row: %+v", row)
	}
}

func TestL1GetReturnsCachedValueEvenWhenTouchUpdateFails(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "cache.sqlite")
	svc, err := NewFromConfig(config.CacheConfig{
		Enabled: true,
		DBPath:  dbPath,
		SQLite:  config.CacheSQLiteConfig{MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: 5},
		L1:      config.CacheTierConfig{Enabled: true, MaxItems: 10},
	})
	if err != nil {
		t.Fatalf("new cache service: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	if err := svc.L1.Put(context.Background(), ExactCacheWrite{
		Key:          "k-touch-fail",
		Value:        "v-touch-fail",
		SourceLocale: "en",
		TargetLocale: "fr",
		Provider:     "openai",
		Model:        "gpt",
		SourceHash:   "hash",
	}); err != nil {
		t.Fatalf("seed cache entry: %v", err)
	}

	callbackName := "test:force_touch_update_failure"
	if err := svc.db.Callback().Update().Before("gorm:update").Register(callbackName, func(db *gorm.DB) {
		db.AddError(errors.New("forced update failure"))
	}); err != nil {
		t.Fatalf("register update callback: %v", err)
	}
	t.Cleanup(func() {
		_ = svc.db.Callback().Update().Remove(callbackName)
	})

	value, hit, err := svc.L1.Get(context.Background(), "k-touch-fail")
	if err != nil {
		t.Fatalf("get cache entry: %v", err)
	}
	if !hit {
		t.Fatal("expected cache hit")
	}
	if value != "v-touch-fail" {
		t.Fatalf("value=%q, want %q", value, "v-touch-fail")
	}
}
