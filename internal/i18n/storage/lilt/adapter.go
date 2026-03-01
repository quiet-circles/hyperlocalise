package lilt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

const (
	AdapterName         = "lilt"
	defaultTokenEnvName = "LILT_API_TOKEN"
)

type Config struct {
	ProjectID       string   `json:"projectID"`
	APIToken        string   `json:"-"`
	APITokenEnv     string   `json:"apiTokenEnv,omitempty"`
	TargetLanguages []string `json:"targetLanguages,omitempty"`
	TimeoutSeconds  int      `json:"timeoutSeconds,omitempty"`
	PollIntervalMS  int      `json:"pollIntervalMs,omitempty"`
	MaxPolls        int      `json:"maxPolls,omitempty"`
}

type ExportedFile struct {
	Name string
	Data []byte
}

type PullInput struct {
	ProjectID string
	APIToken  string
	Locales   []string
}

type PullOutput struct {
	FilesByLocale map[string][]ExportedFile
	Revision      string
}

type UploadFile struct {
	Locale   string
	Filename string
	Data     []byte
}

type PushInput struct {
	ProjectID string
	APIToken  string
	Files     []UploadFile
}

type PushOutput struct {
	Revision string
	JobIDs   []string
}

type Client interface {
	PullTranslations(ctx context.Context, in PullInput) (PullOutput, error)
	PushTranslations(ctx context.Context, in PushInput) (PushOutput, error)
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
		return nil, fmt.Errorf("lilt adapter: client must not be nil")
	}
	return &Adapter{cfg: cfg, client: client}, nil
}

func ParseConfig(raw json.RawMessage) (Config, error) {
	var cfg Config
	if len(raw) == 0 {
		return cfg, fmt.Errorf("lilt config: must not be empty")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("lilt config: decode: %w", err)
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
	if cfg.PollIntervalMS <= 0 {
		cfg.PollIntervalMS = 1000
	}
	if cfg.MaxPolls <= 0 {
		cfg.MaxPolls = 60
	}
	if err := validateConfig(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.ProjectID) == "" {
		return fmt.Errorf("lilt config: projectID is required")
	}
	if strings.TrimSpace(cfg.APIToken) == "" {
		return fmt.Errorf("lilt config: API token is required (%s)", defaultTokenEnvName)
	}
	return nil
}

func (a *Adapter) Name() string { return AdapterName }

func (a *Adapter) Capabilities() storage.Capabilities {
	return storage.Capabilities{SupportsContext: true}
}

func (a *Adapter) Pull(ctx context.Context, req storage.PullRequest) (storage.PullResult, error) {
	locales := req.Locales
	if len(locales) == 0 && len(a.cfg.TargetLanguages) > 0 {
		locales = append([]string(nil), a.cfg.TargetLanguages...)
	}

	out, err := a.client.PullTranslations(ctx, PullInput{ProjectID: a.cfg.ProjectID, APIToken: a.cfg.APIToken, Locales: locales})
	if err != nil {
		return storage.PullResult{}, fmt.Errorf("lilt pull: %w", err)
	}

	now := time.Now().UTC()
	latestByID := map[storage.EntryID]storage.Entry{}
	for locale, files := range out.FilesByLocale {
		trimmedLocale := strings.TrimSpace(locale)
		for _, file := range files {
			entries, parseErr := ParseArtifact(file.Name, trimmedLocale, file.Data)
			if parseErr != nil {
				return storage.PullResult{}, fmt.Errorf("lilt pull parse %s/%s: %w", locale, file.Name, parseErr)
			}
			for _, entry := range entries {
				key := strings.TrimSpace(entry.Key)
				locale := strings.TrimSpace(entry.Locale)
				value := strings.TrimSpace(entry.Value)
				if key == "" || locale == "" || value == "" {
					continue
				}
				entry.Key = key
				entry.Locale = locale
				entry.Provenance = storage.EntryProvenance{Origin: storage.OriginHuman, State: storage.StateCurated, UpdatedAt: now}
				entry.Remote = storage.RemoteMeta{Adapter: AdapterName, Revision: out.Revision}
				latestByID[entry.ID()] = entry
			}
		}
	}

	entries := make([]storage.Entry, 0, len(latestByID))
	for _, entry := range latestByID {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Locale != entries[j].Locale {
			return entries[i].Locale < entries[j].Locale
		}
		if entries[i].Key != entries[j].Key {
			return entries[i].Key < entries[j].Key
		}
		return entries[i].Context < entries[j].Context
	})

	retrievedAt := now
	return storage.PullResult{Snapshot: storage.CatalogSnapshot{Entries: entries, Revision: out.Revision, RetrievedAt: &retrievedAt}}, nil
}

func (a *Adapter) Push(ctx context.Context, req storage.PushRequest) (storage.PushResult, error) {
	latestByID := map[storage.EntryID]storage.Entry{}
	skippedSet := map[storage.EntryID]struct{}{}
	for _, entry := range req.Entries {
		key := strings.TrimSpace(entry.Key)
		locale := strings.TrimSpace(entry.Locale)
		value := strings.TrimSpace(entry.Value)
		if key == "" || locale == "" || value == "" {
			skippedSet[entry.ID()] = struct{}{}
			continue
		}
		entry.Key = key
		entry.Locale = locale
		entry.Value = value
		latestByID[entry.ID()] = entry
	}
	// Remove IDs from skippedSet that ended up being applied
	for id := range latestByID {
		delete(skippedSet, id)
	}

	filesByLocale := map[string][]flatRecord{}
	applied := make([]storage.EntryID, 0, len(latestByID))
	for _, entry := range latestByID {
		filesByLocale[entry.Locale] = append(filesByLocale[entry.Locale], flatRecord{Key: entry.Key, Context: entry.Context, Value: entry.Value})
		applied = append(applied, entry.ID())
	}
	sort.Slice(applied, func(i, j int) bool {
		if applied[i].Locale != applied[j].Locale {
			return applied[i].Locale < applied[j].Locale
		}
		if applied[i].Key != applied[j].Key {
			return applied[i].Key < applied[j].Key
		}
		return applied[i].Context < applied[j].Context
	})

	files := make([]UploadFile, 0, len(filesByLocale))
	for locale, records := range filesByLocale {
		raw, err := json.Marshal(records)
		if err != nil {
			return storage.PushResult{}, fmt.Errorf("lilt push: marshal %s payload: %w", locale, err)
		}
		files = append(files, UploadFile{Locale: locale, Filename: fmt.Sprintf("hyperlocalise-%s.json", locale), Data: raw})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Locale < files[j].Locale })

	if len(files) == 0 {
		return storage.PushResult{Skipped: sortedIDs(skippedSet), Revision: time.Now().UTC().Format(time.RFC3339Nano)}, nil
	}

	out, err := a.client.PushTranslations(ctx, PushInput{ProjectID: a.cfg.ProjectID, APIToken: a.cfg.APIToken, Files: files})
	if err != nil {
		return storage.PushResult{Applied: applied, Skipped: sortedIDs(skippedSet)}, fmt.Errorf("lilt push: %w", err)
	}

	revision := strings.TrimSpace(out.Revision)
	if revision == "" && len(out.JobIDs) > 0 {
		revision = "jobs:" + strings.Join(out.JobIDs, ",")
	}
	warnings := make([]storage.Warning, 0, len(out.JobIDs))
	for _, jobID := range out.JobIDs {
		trimmed := strings.TrimSpace(jobID)
		if trimmed == "" {
			continue
		}
		warnings = append(warnings, storage.Warning{Code: "lilt_import_job", Message: fmt.Sprintf("lilt import job %s", trimmed)})
	}
	return storage.PushResult{Applied: applied, Skipped: sortedIDs(skippedSet), Revision: revision, Warnings: warnings}, nil
}

func sortedIDs(set map[storage.EntryID]struct{}) []storage.EntryID {
	if len(set) == 0 {
		return nil
	}
	out := make([]storage.EntryID, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Locale != out[j].Locale {
			return out[i].Locale < out[j].Locale
		}
		if out[i].Key != out[j].Key {
			return out[i].Key < out[j].Key
		}
		return out[i].Context < out[j].Context
	})
	return out
}
