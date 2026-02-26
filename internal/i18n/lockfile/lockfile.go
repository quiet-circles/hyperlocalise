package lockfile

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const DefaultPath = ".hyperlocalise.lock.json"

type File struct {
	Adapter      string                      `json:"adapter,omitempty"`
	ProjectID    string                      `json:"project_id,omitempty"`
	LastPullAt   *time.Time                  `json:"last_pull_at,omitempty"`
	LocaleStates map[string]LocaleCheckpoint `json:"locale_states,omitempty"`
}

type LocaleCheckpoint struct {
	Revision  string     `json:"revision,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

func Load(path string) (*File, error) {
	if path == "" {
		path = DefaultPath
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{LocaleStates: map[string]LocaleCheckpoint{}}, nil
		}
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	var f File
	if err := json.Unmarshal(content, &f); err != nil {
		return nil, fmt.Errorf("decode lockfile: %w", err)
	}
	if f.LocaleStates == nil {
		f.LocaleStates = map[string]LocaleCheckpoint{}
	}

	return &f, nil
}

func Save(path string, f File) error {
	if path == "" {
		path = DefaultPath
	}
	if f.LocaleStates == nil {
		f.LocaleStates = map[string]LocaleCheckpoint{}
	}

	content, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}
	content = append(content, '\n')

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}
	return nil
}
