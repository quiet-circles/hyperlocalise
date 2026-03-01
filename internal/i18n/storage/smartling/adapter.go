package smartling

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

const (
	AdapterName             = "smartling"
	defaultUserSecretEnvVar = "SMARTLING_USER_SECRET"
)

type Config struct {
	ProjectID       string   `json:"projectID"`
	UserIdentifier  string   `json:"userIdentifier"`
	UserSecret      string   `json:"-"`
	UserSecretEnv   string   `json:"userSecretEnv,omitempty"`
	TargetLanguages []string `json:"targetLanguages,omitempty"`
	TimeoutSeconds  int      `json:"timeoutSeconds,omitempty"`
}

type StringTranslation struct {
	Key     string `json:"stringText"`
	Context string `json:"instruction,omitempty"`
	Locale  string `json:"targetLocaleId"`
	Value   string `json:"translation"`
}

type ListTranslationsInput struct {
	ProjectID string
	Locales   []string
}

type UpsertTranslationsInput struct {
	ProjectID string
	Entries   []StringTranslation
}

type Client interface {
	ListTranslations(ctx context.Context, in ListTranslationsInput) ([]StringTranslation, string, error)
	UpsertTranslations(ctx context.Context, in UpsertTranslationsInput) (string, error)
}

type Adapter struct {
	cfg    Config
	client Client
}

func New(raw json.RawMessage) (storage.StorageAdapter, error) {
	cfg, err := ParseConfig(raw)
	if err != nil {
		return nil, err
	}

	client, err := NewHTTPClient(cfg)
	if err != nil {
		return nil, err
	}

	return NewWithClient(cfg, client)
}

func NewWithClient(cfg Config, client Client) (*Adapter, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("smartling adapter: client must not be nil")
	}
	return &Adapter{cfg: cfg, client: client}, nil
}

func ParseConfig(raw json.RawMessage) (Config, error) {
	var cfg Config
	if len(raw) == 0 {
		return cfg, fmt.Errorf("smartling config: must not be empty")
	}
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return cfg, fmt.Errorf("smartling config: decode: %w", err)
	}
	if _, exists := rawMap["userSecret"]; exists {
		return cfg, fmt.Errorf("smartling config: userSecret is not supported; use %s", defaultUserSecretEnvVar)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("smartling config: decode: %w", err)
	}

	if strings.TrimSpace(cfg.UserSecretEnv) == "" {
		cfg.UserSecretEnv = defaultUserSecretEnvVar
	}
	if strings.TrimSpace(cfg.UserSecret) == "" {
		cfg.UserSecret = os.Getenv(cfg.UserSecretEnv)
		if strings.TrimSpace(cfg.UserSecret) == "" && cfg.UserSecretEnv != defaultUserSecretEnvVar {
			cfg.UserSecret = os.Getenv(defaultUserSecretEnvVar)
		}
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 30
	}

	if err := validateConfig(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.ProjectID) == "" {
		return fmt.Errorf("smartling config: projectID is required")
	}
	if strings.TrimSpace(cfg.UserIdentifier) == "" {
		return fmt.Errorf("smartling config: userIdentifier is required")
	}
	if strings.TrimSpace(cfg.UserSecret) == "" {
		return fmt.Errorf("smartling config: user secret is required (%s)", defaultUserSecretEnvVar)
	}
	return nil
}

func (a *Adapter) Name() string { return AdapterName }

func (a *Adapter) Capabilities() storage.Capabilities {
	return storage.Capabilities{
		SupportsContext:    true,
		SupportsVersions:   false,
		SupportsDeletes:    false,
		SupportsNamespaces: false,
	}
}

func (a *Adapter) Pull(ctx context.Context, req storage.PullRequest) (storage.PullResult, error) {
	locales := req.Locales
	if len(locales) == 0 && len(a.cfg.TargetLanguages) > 0 {
		locales = append([]string(nil), a.cfg.TargetLanguages...)
	}

	stringsOut, revision, err := a.client.ListTranslations(ctx, ListTranslationsInput{
		ProjectID: a.cfg.ProjectID,
		Locales:   locales,
	})
	if err != nil {
		return storage.PullResult{}, fmt.Errorf("smartling pull: %w", err)
	}

	entries := make([]storage.Entry, 0, len(stringsOut))
	now := time.Now().UTC()
	for _, item := range stringsOut {
		if strings.TrimSpace(item.Locale) == "" || strings.TrimSpace(item.Value) == "" {
			continue
		}
		entries = append(entries, storage.Entry{
			Key:     item.Key,
			Context: item.Context,
			Locale:  item.Locale,
			Value:   item.Value,
			Provenance: storage.EntryProvenance{
				Origin:    storage.OriginHuman,
				State:     storage.StateCurated,
				UpdatedAt: now,
			},
			Remote: storage.RemoteMeta{Adapter: AdapterName, Revision: revision},
		})
	}

	retrievedAt := now
	return storage.PullResult{Snapshot: storage.CatalogSnapshot{Entries: entries, Revision: revision, RetrievedAt: &retrievedAt}}, nil
}

func (a *Adapter) Push(ctx context.Context, req storage.PushRequest) (storage.PushResult, error) {
	payload := make([]StringTranslation, 0, len(req.Entries))
	applied := make([]storage.EntryID, 0, len(req.Entries))
	indexByID := make(map[storage.EntryID]int, len(req.Entries))
	for _, entry := range req.Entries {
		key := strings.TrimSpace(entry.Key)
		locale := strings.TrimSpace(entry.Locale)
		if key == "" || locale == "" || strings.TrimSpace(entry.Value) == "" {
			continue
		}

		id := entry.ID()
		translation := StringTranslation{
			Key:     key,
			Context: strings.TrimSpace(entry.Context),
			Locale:  locale,
			Value:   entry.Value,
		}
		if idx, exists := indexByID[id]; exists {
			// Keep one write per EntryID and let the newest entry win.
			payload[idx] = translation
			continue
		}

		indexByID[id] = len(payload)
		payload = append(payload, translation)
		applied = append(applied, entry.ID())
	}

	revision, err := a.client.UpsertTranslations(ctx, UpsertTranslationsInput{ProjectID: a.cfg.ProjectID, Entries: payload})
	if err != nil {
		return storage.PushResult{}, fmt.Errorf("smartling push: %w", err)
	}
	return storage.PushResult{Applied: applied, Revision: revision}, nil
}
