package cache

import "time"

// ExactCacheEntry stores exact-match translations for deterministic reuse.
type ExactCacheEntry struct {
	ID           uint   `gorm:"primaryKey"`
	CacheKey     string `gorm:"size:512;uniqueIndex;not null"`
	SourceLocale string `gorm:"size:32;index;not null"`
	TargetLocale string `gorm:"size:32;index;not null"`
	Provider     string `gorm:"size:64;index;not null"`
	Model        string `gorm:"size:128;index;not null"`
	SourceHash   string `gorm:"size:128;index;not null"`
	Value        string `gorm:"type:text;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TranslationMemoryEntry stores L2 memory candidates.
type TranslationMemoryEntry struct {
	ID             uint    `gorm:"primaryKey"`
	SourceLocale   string  `gorm:"size:32;index:idx_tm_locales_text,priority:1;not null"`
	TargetLocale   string  `gorm:"size:32;index:idx_tm_locales_text,priority:2;not null"`
	SourceText     string  `gorm:"type:text;index:idx_tm_locales_text,priority:3;not null"`
	TranslatedText string  `gorm:"type:text;not null"`
	Score          float64 `gorm:"not null;default:0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
