package evalset

import (
	"fmt"
	"strings"
)

// Dataset defines a translation evaluation dataset.
type Dataset struct {
	Version  string            `json:"version,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Cases    []Case            `json:"cases"`
}

// Case defines a single evaluation sample.
type Case struct {
	ID           string   `json:"id"`
	Source       string   `json:"source"`
	TargetLocale string   `json:"targetLocale"`
	Context      string   `json:"context,omitempty"`
	Reference    string   `json:"reference,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Bucket       string   `json:"bucket,omitempty"`
	Group        string   `json:"group,omitempty"`
}

// Validate checks the dataset semantics.
func (d Dataset) Validate() error {
	if len(d.Cases) == 0 {
		return fmt.Errorf("cases: must not be empty")
	}

	ids := make(map[string]struct{}, len(d.Cases))

	for i, tc := range d.Cases {
		normalizedID := strings.TrimSpace(tc.ID)

		if normalizedID == "" {
			return fmt.Errorf("cases[%d].id: must not be empty", i)
		}

		if _, exists := ids[normalizedID]; exists {
			return fmt.Errorf("cases[%d].id: duplicate id %q", i, tc.ID)
		}

		if strings.TrimSpace(tc.Source) == "" {
			return fmt.Errorf("cases[%d].source: must not be empty", i)
		}

		if strings.TrimSpace(tc.TargetLocale) == "" {
			return fmt.Errorf("cases[%d].targetLocale: must not be empty", i)
		}

		ids[normalizedID] = struct{}{}
	}

	return nil
}
