package translationfileparser

import (
	"encoding/json"
	"fmt"
	"sort"
)

// JSONParser parses translation JSON files.
type JSONParser struct{}

func (p JSONParser) Parse(content []byte) (map[string]string, error) {
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}

	if payload == nil {
		return map[string]string{}, nil
	}

	out := make(map[string]string)
	if err := flattenJSON(out, "", payload); err != nil {
		return nil, err
	}

	return out, nil
}

func flattenJSON(out map[string]string, prefix string, input map[string]any) error {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := input[key]
		nextKey := key
		if prefix != "" {
			nextKey = prefix + "." + key
		}

		switch typed := value.(type) {
		case string:
			out[nextKey] = typed
		case map[string]any:
			if err := flattenJSON(out, nextKey, typed); err != nil {
				return err
			}
		default:
			return fmt.Errorf("json key %q must be string or object, got %T", nextKey, value)
		}
	}

	return nil
}
