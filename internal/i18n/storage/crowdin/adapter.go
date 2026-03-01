package crowdin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

const (
	AdapterName         = "crowdin"
	defaultTokenEnvName = "CROWDIN_API_TOKEN"
)

type Config struct {
	ProjectID       string   `json:"projectID"`
	APIToken        string   `json:"-"`
	APITokenEnv     string   `json:"apiTokenEnv,omitempty"`
	SourceLanguage  string   `json:"sourceLanguage,omitempty"`
	TargetLanguages []string `json:"targetLanguages,omitempty"`
	TimeoutSeconds  int      `json:"timeoutSeconds,omitempty"`
}

type StringTranslation struct {
	Key     string
	Context string
	Locale  string
	Value   string
}

type ListStringsInput struct {
	ProjectID string
	APIToken  string
	Locales   []string
}

type UpsertTranslationsInput struct {
	ProjectID string
	APIToken  string
	Entries   []StringTranslation
}

type Client interface {
	ListStrings(ctx context.Context, in ListStringsInput) ([]StringTranslation, string, error)
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
		return nil, fmt.Errorf("crowdin adapter: client must not be nil")
	}
	return &Adapter{cfg: cfg, client: client}, nil
}

func ParseConfig(raw json.RawMessage) (Config, error) {
	var cfg Config
	if len(raw) == 0 {
		return cfg, fmt.Errorf("crowdin config: must not be empty")
	}

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return cfg, fmt.Errorf("crowdin config: decode: %w", err)
	}
	if _, exists := rawMap["apiToken"]; exists {
		return cfg, fmt.Errorf("crowdin config: apiToken is not supported; use %s", defaultTokenEnvName)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("crowdin config: decode: %w", err)
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
		return fmt.Errorf("crowdin config: projectID is required")
	}
	projectID, err := strconv.Atoi(strings.TrimSpace(cfg.ProjectID))
	if err != nil || projectID <= 0 {
		return fmt.Errorf("crowdin config: projectID must be a positive integer")
	}
	if strings.TrimSpace(cfg.APIToken) == "" {
		return fmt.Errorf("crowdin config: API token is required (%s)", defaultTokenEnvName)
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

	stringsResp, revision, err := a.client.ListStrings(ctx, ListStringsInput{
		ProjectID: a.cfg.ProjectID,
		APIToken:  a.cfg.APIToken,
		Locales:   locales,
	})
	if err != nil {
		return storage.PullResult{}, fmt.Errorf("crowdin pull: %w", err)
	}

	entries := make([]storage.Entry, 0, len(stringsResp))
	now := time.Now().UTC()
	for _, tr := range stringsResp {
		if strings.TrimSpace(tr.Locale) == "" {
			continue
		}
		if strings.TrimSpace(tr.Value) == "" {
			continue
		}
		entries = append(entries, storage.Entry{
			Key:     tr.Key,
			Context: tr.Context,
			Locale:  tr.Locale,
			Value:   tr.Value,
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
	return storage.PullResult{Snapshot: storage.CatalogSnapshot{Entries: entries, Revision: revision, RetrievedAt: &retrievedAt}}, nil
}

func (a *Adapter) Push(ctx context.Context, req storage.PushRequest) (storage.PushResult, error) {
	payload := make([]StringTranslation, 0, len(req.Entries))
	applied := make([]storage.EntryID, 0, len(req.Entries))
	indexByID := make(map[storage.EntryID]int, len(req.Entries))

	for _, entry := range req.Entries {
		key := strings.TrimSpace(entry.Key)
		locale := strings.TrimSpace(entry.Locale)
		if strings.TrimSpace(entry.Value) == "" {
			continue
		}
		if key == "" || locale == "" {
			continue
		}

		id := entry.ID()
		translation := StringTranslation{
			Key:     key,
			Context: entry.Context,
			Locale:  locale,
			Value:   entry.Value,
		}
		if idx, exists := indexByID[id]; exists {
			// Keep one remote write per EntryID, but let the latest local value win.
			payload[idx] = translation
			continue
		}

		indexByID[id] = len(payload)
		payload = append(payload, translation)
		applied = append(applied, id)
	}

	if len(payload) == 0 {
		return storage.PushResult{
			Applied:  nil,
			Revision: time.Now().UTC().Format(time.RFC3339Nano),
		}, nil
	}

	revision, err := a.client.UpsertTranslations(ctx, UpsertTranslationsInput{ProjectID: a.cfg.ProjectID, APIToken: a.cfg.APIToken, Entries: payload})
	if err != nil {
		sentIndexes := sentIndexesFromError(err)
		partialApplied := make([]storage.EntryID, 0, len(sentIndexes))
		for _, idx := range sentIndexes {
			if idx < 0 || idx >= len(applied) {
				continue
			}
			partialApplied = append(partialApplied, applied[idx])
		}
		return storage.PushResult{
			Applied: partialApplied,
		}, fmt.Errorf("crowdin push: %w", err)
	}

	return storage.PushResult{Applied: applied, Revision: revision}, nil
}
