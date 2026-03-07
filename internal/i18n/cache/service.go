package cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ExactCache is the L1 exact-match cache contract.
type ExactCache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Put(ctx context.Context, key, value string) error
}

// TranslationMemory is the L2 fuzzy memory contract.
type TranslationMemory interface {
	Upsert(ctx context.Context, sourceLocale, targetLocale, sourceText, translatedText string, score float64) error
	Lookup(ctx context.Context, sourceLocale, targetLocale, sourceText string, limit int) ([]TMResult, error)
}

// Retriever is an optional retrieval contract for context augmentation.
type Retriever interface {
	Retrieve(ctx context.Context, query string, limit int) ([]RAGDocument, error)
}

// TMResult is a translation-memory lookup result.
type TMResult struct {
	SourceText     string
	TranslatedText string
	Score          float64
}

// RAGDocument is an optional retrieval result.
type RAGDocument struct {
	ID      string
	Content string
	Score   float64
}

// Service groups L1/L2/RAG cache dependencies.
type Service struct {
	Enabled   bool
	L1        ExactCache
	L2        TranslationMemory
	Retriever Retriever

	db *gorm.DB
}

// NewFromConfig bootstraps cache service dependencies from config.
func NewFromConfig(cfg config.CacheConfig) (*Service, error) {
	svc := &Service{
		Enabled: cfg.Enabled,
		L1:      noopExactCache{},
		L2:      noopTranslationMemory{},
	}

	if !cfg.Enabled {
		return svc, nil
	}

	if err := ensureDBDir(cfg.DBPath); err != nil {
		return nil, fmt.Errorf("prepare cache db path: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite cache db: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve sqlite sql db: %w", err)
	}
	applyConnPool(sqlDB, cfg)

	if err := db.AutoMigrate(&ExactCacheEntry{}, &TranslationMemoryEntry{}); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("migrate cache schema: %w", err)
	}

	svc.db = db
	if cfg.L1.Enabled {
		svc.L1 = &exactSQLiteStore{db: db, maxItems: cfg.L1.MaxItems}
	}
	if cfg.L2.Enabled {
		svc.L2 = &tmSQLiteStore{db: db}
	}
	if cfg.RAG.Enabled {
		svc.Retriever = noopRetriever{}
	}

	return svc, nil
}

// Close closes underlying DB resources.
func (s *Service) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("resolve sqlite sql db: %w", err)
	}
	if closeErr := sqlDB.Close(); closeErr != nil && !errors.Is(closeErr, sql.ErrConnDone) {
		return fmt.Errorf("close sqlite cache db: %w", closeErr)
	}
	return nil
}

func applyConnPool(sqlDB *sql.DB, cfg config.CacheConfig) {
	sqlDB.SetMaxOpenConns(cfg.SQLite.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.SQLite.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.SQLite.ConnMaxLifetime) * time.Second)
}

func ensureDBDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return nil
}
