package cache

import (
	"path/filepath"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/config"
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
