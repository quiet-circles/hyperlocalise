package storage

import (
	"context"
	"encoding/json"
	"time"
)

const (
	OriginLLM      = "llm"
	OriginHuman    = "human"
	OriginImported = "imported"
	OriginUnknown  = "unknown"

	StateDraft   = "draft"
	StateCurated = "curated"
)

// Entry is the normalized translation item exchanged across local storage and remote providers.
type Entry struct {
	Key        string            `json:"key"`
	Context    string            `json:"context,omitempty"`
	Locale     string            `json:"locale"`
	Value      string            `json:"value"`
	Namespace  string            `json:"namespace,omitempty"`
	Provenance EntryProvenance   `json:"provenance,omitempty"`
	Remote     RemoteMeta        `json:"remote,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// EntryProvenance stores local state for LLM vs human-curated flows.
type EntryProvenance struct {
	Origin    string    `json:"origin,omitempty"`
	State     string    `json:"state,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	UpdatedBy string    `json:"updated_by,omitempty"`
}

// RemoteMeta stores remote adapter metadata for an entry.
type RemoteMeta struct {
	Adapter   string    `json:"adapter,omitempty"`
	Revision  string    `json:"revision,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// EntryID is the stable identity for a translation entry.
type EntryID struct {
	Key     string `json:"key"`
	Context string `json:"context,omitempty"`
	Locale  string `json:"locale"`
}

func (e Entry) ID() EntryID {
	return EntryID{
		Key:     e.Key,
		Context: e.Context,
		Locale:  e.Locale,
	}
}

// CatalogSnapshot is a provider/local snapshot at a point in time.
type CatalogSnapshot struct {
	Entries     []Entry    `json:"entries"`
	Revision    string     `json:"revision,omitempty"`
	RetrievedAt *time.Time `json:"retrieved_at,omitempty"`
}

// PullRequest downloads entries from a remote storage provider.
type PullRequest struct {
	Locales       []string          `json:"locales,omitempty"`
	Namespaces    []string          `json:"namespaces,omitempty"`
	KeyPrefixes   []string          `json:"key_prefixes,omitempty"`
	AdapterConfig json.RawMessage   `json:"-"`
	Options       map[string]string `json:"options,omitempty"`
}

// PullResult returns downloaded remote entries and adapter warnings.
type PullResult struct {
	Snapshot CatalogSnapshot `json:"snapshot"`
	Warnings []Warning       `json:"warnings,omitempty"`
}

// PushRequest uploads entries to a remote storage provider.
type PushRequest struct {
	Entries       []Entry            `json:"entries"`
	Locales       []string           `json:"locales,omitempty"`
	AdapterConfig json.RawMessage    `json:"-"`
	Options       map[string]string  `json:"options,omitempty"`
	Baseline      map[EntryID]string `json:"baseline,omitempty"`
}

// PushResult returns provider apply summary and warnings.
type PushResult struct {
	Applied   []EntryID  `json:"applied,omitempty"`
	Skipped   []EntryID  `json:"skipped,omitempty"`
	Conflicts []Conflict `json:"conflicts,omitempty"`
	Warnings  []Warning  `json:"warnings,omitempty"`
	Revision  string     `json:"revision,omitempty"`
}

type Warning struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type Conflict struct {
	ID          EntryID `json:"id"`
	Reason      string  `json:"reason"`
	LocalValue  string  `json:"local_value,omitempty"`
	RemoteValue string  `json:"remote_value,omitempty"`
	LocalState  string  `json:"local_state,omitempty"`
	RemoteState string  `json:"remote_state,omitempty"`
}

// Capabilities describe remote adapter features.
type Capabilities struct {
	SupportsContext    bool `json:"supports_context"`
	SupportsVersions   bool `json:"supports_versions"`
	SupportsDeletes    bool `json:"supports_deletes"`
	SupportsNamespaces bool `json:"supports_namespaces"`
}

// StorageAdapter is the remote translation storage integration contract.
type StorageAdapter interface {
	Name() string
	Capabilities() Capabilities
	Pull(ctx context.Context, req PullRequest) (PullResult, error)
	Push(ctx context.Context, req PushRequest) (PushResult, error)
}
