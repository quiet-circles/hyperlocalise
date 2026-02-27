package poeditor

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
	AdapterName         = "poeditor"
	defaultTokenEnvName = "POEDITOR_API_TOKEN"
)

type Config struct {
	ProjectID       string   `json:"projectID"`
	APIToken        string   `json:"-"`
	APITokenEnv     string   `json:"apiTokenEnv,omitempty"`
	SourceLanguage  string   `json:"sourceLanguage,omitempty"`
	TargetLanguages []string `json:"targetLanguages,omitempty"`
	TimeoutSeconds  int      `json:"timeoutSeconds,omitempty"`
}

type Client interface {
	ListTerms(ctx context.Context, in ListTermsInput) ([]TermTranslation, string, error)
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
		return nil, fmt.Errorf("poeditor adapter: client must not be nil")
	}
	return &Adapter{cfg: cfg, client: client}, nil
}

func ParseConfig(raw json.RawMessage) (Config, error) {
	var cfg Config
	if len(raw) == 0 {
		return cfg, fmt.Errorf("poeditor config: must not be empty")
	}
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return cfg, fmt.Errorf("poeditor config: decode: %w", err)
	}
	if _, exists := rawMap["apiToken"]; exists {
		return cfg, fmt.Errorf("poeditor config: apiToken is not supported; use %s", defaultTokenEnvName)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("poeditor config: decode: %w", err)
	}

	if strings.TrimSpace(cfg.APITokenEnv) == "" {
		cfg.APITokenEnv = defaultTokenEnvName
	}
	if strings.TrimSpace(cfg.APIToken) == "" {
		cfg.APIToken = os.Getenv(cfg.APITokenEnv)
		if strings.TrimSpace(cfg.APIToken) == "" && cfg.APITokenEnv != defaultTokenEnvName {
			cfg.APIToken = os.Getenv(defaultTokenEnvName)
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
		return fmt.Errorf("poeditor config: projectID is required")
	}
	if strings.TrimSpace(cfg.APIToken) == "" {
		return fmt.Errorf("poeditor config: API token is required (%s)", defaultTokenEnvName)
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

	terms, revision, err := a.client.ListTerms(ctx, ListTermsInput{
		ProjectID: a.cfg.ProjectID,
		APIToken:  a.cfg.APIToken,
		Locales:   locales,
	})
	if err != nil {
		return storage.PullResult{}, fmt.Errorf("poeditor pull: %w", err)
	}

	entries := make([]storage.Entry, 0, len(terms))
	now := time.Now().UTC()
	for _, t := range terms {
		if strings.TrimSpace(t.Locale) == "" {
			continue
		}
		if strings.TrimSpace(t.Value) == "" {
			continue
		}
		entries = append(entries, storage.Entry{
			Key:     t.Term,
			Context: t.Context,
			Locale:  t.Locale,
			Value:   t.Value,
			Provenance: storage.EntryProvenance{
				Origin:    storage.OriginHuman,
				State:     storage.StateCurated,
				UpdatedAt: now,
			},
			Remote: storage.RemoteMeta{
				Adapter:  AdapterName,
				Revision: revision,
			},
		})
	}

	retrievedAt := now
	return storage.PullResult{
		Snapshot: storage.CatalogSnapshot{
			Entries:     entries,
			Revision:    revision,
			RetrievedAt: &retrievedAt,
		},
	}, nil
}

func (a *Adapter) Push(ctx context.Context, req storage.PushRequest) (storage.PushResult, error) {
	payload := make([]TermTranslation, 0, len(req.Entries))
	for _, entry := range req.Entries {
		if strings.TrimSpace(entry.Value) == "" {
			continue
		}
		payload = append(payload, TermTranslation{
			Term:    entry.Key,
			Context: entry.Context,
			Locale:  entry.Locale,
			Value:   entry.Value,
		})
	}

	revision, err := a.client.UpsertTranslations(ctx, UpsertTranslationsInput{
		ProjectID: a.cfg.ProjectID,
		APIToken:  a.cfg.APIToken,
		Entries:   payload,
	})
	if err != nil {
		return storage.PushResult{}, fmt.Errorf("poeditor push: %w", err)
	}

	applied := make([]storage.EntryID, 0, len(req.Entries))
	for _, entry := range req.Entries {
		applied = append(applied, entry.ID())
	}

	return storage.PushResult{
		Applied:  applied,
		Revision: revision,
	}, nil
}

type TermTranslation struct {
	Term    string
	Context string
	Locale  string
	Value   string
}

type ListTermsInput struct {
	ProjectID string
	APIToken  string
	Locales   []string
}

type UpsertTranslationsInput struct {
	ProjectID string
	APIToken  string
	Entries   []TermTranslation
}
