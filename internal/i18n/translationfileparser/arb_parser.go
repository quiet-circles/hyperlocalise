package translationfileparser

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// ARBParser parses Flutter .arb translation files.
type ARBParser struct{}

func (p ARBParser) Parse(content []byte) (map[string]string, error) {
	values, _, err := p.ParseWithContext(content)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (p ARBParser) ParseWithContext(content []byte) (map[string]string, map[string]string, error) {
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, nil, fmt.Errorf("arb decode: %w", err)
	}
	if payload == nil {
		return map[string]string{}, nil, nil
	}

	out := map[string]string{}
	descriptions := map[string]string{}
	for _, key := range sortedMapKeys(payload) {
		if isARBMetadataKey(key) {
			continue
		}

		value, ok := payload[key].(string)
		if !ok {
			return nil, nil, fmt.Errorf("arb key %q must be string, got %T", key, payload[key])
		}
		out[key] = value
		if description := parseARBDescription(payload, key); description != "" {
			descriptions[key] = description
		}
	}
	if len(descriptions) == 0 {
		return out, nil, nil
	}
	return out, descriptions, nil
}

func parseARBDescription(payload map[string]any, key string) string {
	raw, ok := payload["@"+key]
	if !ok {
		return ""
	}
	meta, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	descRaw, ok := meta["description"]
	if !ok {
		return ""
	}
	description, ok := descRaw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(description)
}

// MarshalARB preserves metadata keys and rewrites only translatable message keys.
func MarshalARB(template []byte, values map[string]string) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(template, &payload); err != nil {
		return nil, fmt.Errorf("arb decode: %w", err)
	}
	if payload == nil {
		payload = map[string]any{}
	}

	out := map[string]any{}
	for key, value := range payload {
		if isARBMetadataKey(key) {
			out[key] = value
		}
	}

	for _, key := range sortedMapKeys(values) {
		out[key] = values[key]
	}

	content, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("arb encode: %w", err)
	}
	return append(content, '\n'), nil
}

func isARBMetadataKey(key string) bool {
	return strings.HasPrefix(key, "@")
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
