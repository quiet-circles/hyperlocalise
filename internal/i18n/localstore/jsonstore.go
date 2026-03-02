package localstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/pathresolver"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/syncsvc"
)

type JSONStore struct {
	cfg           *config.I18NConfig
	localePattern string
	namespace     string
}

func NewJSONStore(cfg *config.I18NConfig) (*JSONStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("new json store: config is nil")
	}

	localePattern, namespace, err := resolveLocalePattern(cfg.Buckets)
	if err != nil {
		return nil, err
	}

	return &JSONStore{cfg: cfg, localePattern: localePattern, namespace: namespace}, nil
}

func (s *JSONStore) ReadSnapshot(ctx context.Context, req syncsvc.LocalReadRequest) (storage.CatalogSnapshot, error) {
	return s.readSnapshot(ctx, req)
}

func (s *JSONStore) BuildPushSnapshot(ctx context.Context, req syncsvc.LocalReadRequest) (storage.CatalogSnapshot, error) {
	return s.readSnapshot(ctx, req)
}

func (s *JSONStore) readSnapshot(_ context.Context, req syncsvc.LocalReadRequest) (storage.CatalogSnapshot, error) {
	locales := req.Locales
	if len(locales) == 0 {
		locales = append([]string(nil), s.cfg.Locales.Targets...)
	}

	var entries []storage.Entry
	for _, locale := range locales {
		path := s.localePath(locale)
		valueMap, err := readLocaleValues(path)
		if err != nil {
			return storage.CatalogSnapshot{}, fmt.Errorf("read locale file %q: %w", path, err)
		}

		metaMap, err := readLocaleMeta(metaPathFor(path))
		if err != nil {
			return storage.CatalogSnapshot{}, fmt.Errorf("read locale metadata %q: %w", metaPathFor(path), err)
		}

		for key, value := range valueMap {
			entry := storage.Entry{
				Key:       key,
				Locale:    locale,
				Value:     value,
				Namespace: s.namespace,
			}
			if meta, ok := metaMap[entryMetaID(key, "")]; ok {
				entry.Provenance = meta.Provenance
				entry.Remote = meta.Remote
			}
			if strings.TrimSpace(entry.Provenance.Origin) == "" {
				entry.Provenance.Origin = storage.OriginUnknown
			}
			entries = append(entries, entry)
		}
	}

	return storage.CatalogSnapshot{Entries: entries}, nil
}

func (s *JSONStore) ApplyPull(_ context.Context, plan syncsvc.ApplyPullPlan) (syncsvc.ApplyResult, error) {
	byLocale := make(map[string][]storage.Entry)
	for _, entry := range plan.Creates {
		byLocale[entry.Locale] = append(byLocale[entry.Locale], entry)
	}
	for _, entry := range plan.Updates {
		byLocale[entry.Locale] = append(byLocale[entry.Locale], entry)
	}

	applied := make([]storage.EntryID, 0)

	for locale, entries := range byLocale {
		path := s.localePath(locale)
		values, err := readLocaleValues(path)
		if err != nil {
			return syncsvc.ApplyResult{}, fmt.Errorf("read locale file %q before apply: %w", path, err)
		}
		metaPath := metaPathFor(path)
		metaMap, err := readLocaleMeta(metaPath)
		if err != nil {
			return syncsvc.ApplyResult{}, fmt.Errorf("read locale metadata %q before apply: %w", metaPath, err)
		}

		for _, entry := range entries {
			values[entry.Key] = entry.Value
			metaMap[entryMetaID(entry.Key, entry.Context)] = entryMeta{
				Provenance: entry.Provenance,
				Remote:     entry.Remote,
			}
			applied = append(applied, entry.ID())
		}

		if err := writeJSONAtomic(path, values); err != nil {
			return syncsvc.ApplyResult{}, fmt.Errorf("write locale file %q: %w", path, err)
		}
		if err := writeJSONAtomic(metaPath, metaMap); err != nil {
			return syncsvc.ApplyResult{}, fmt.Errorf("write locale metadata %q: %w", metaPath, err)
		}
	}

	return syncsvc.ApplyResult{Applied: applied}, nil
}

func (s *JSONStore) localePath(locale string) string {
	return pathresolver.ResolveTargetPath(s.localePattern, s.cfg.Locales.Source, locale)
}

func resolveLocalePattern(buckets map[string]config.BucketConfig) (string, string, error) {
	if len(buckets) == 0 {
		return "", "", fmt.Errorf("new json store: buckets is required")
	}

	names := make([]string, 0, len(buckets))
	for name := range buckets {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		bucket := buckets[name]
		for _, file := range bucket.Files {
			if strings.TrimSpace(file.To) != "" {
				return file.To, file.From, nil
			}
		}
	}

	return "", "", fmt.Errorf("new json store: buckets.*.files[].to is required")
}

type entryMeta struct {
	Provenance storage.EntryProvenance `json:"provenance,omitempty"`
	Remote     storage.RemoteMeta      `json:"remote,omitempty"`
}

func readLocaleValues(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	var values map[string]string
	if err := json.Unmarshal(content, &values); err != nil {
		return nil, err
	}
	if values == nil {
		values = map[string]string{}
	}
	return values, nil
}

func readLocaleMeta(path string) (map[string]entryMeta, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]entryMeta{}, nil
		}
		return nil, err
	}

	var meta map[string]entryMeta
	if err := json.Unmarshal(content, &meta); err != nil {
		return nil, err
	}
	if meta == nil {
		meta = map[string]entryMeta{}
	}
	return meta, nil
}

func metaPathFor(localePath string) string {
	ext := filepath.Ext(localePath)
	if ext == "" {
		return localePath + ".meta.json"
	}
	base := strings.TrimSuffix(localePath, ext)
	return base + ".meta" + ext
}

func entryMetaID(key, context string) string {
	return key + "\x1f" + context
}

func writeJSONAtomic(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')

	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return nil
}
