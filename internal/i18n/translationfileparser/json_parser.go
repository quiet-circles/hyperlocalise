package translationfileparser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// JSONParser parses translation JSON files.
type JSONParser struct{}

func (p JSONParser) Parse(content []byte) (map[string]string, error) {
	values, _, err := p.ParseWithContext(content)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (p JSONParser) ParseWithContext(content []byte) (map[string]string, map[string]string, error) {
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, nil, fmt.Errorf("json decode: %w", err)
	}

	if payload == nil {
		return map[string]string{}, nil, nil
	}

	out := make(map[string]string)
	formatJS, err := parseStrictFormatJSMessages(out, payload)
	if err != nil {
		return nil, nil, err
	}
	if formatJS {
		return out, parseFormatJSDescriptions(payload), nil
	}

	if err := flattenJSON(out, "", payload); err != nil {
		return nil, nil, err
	}

	return out, nil, nil
}

func parseStrictFormatJSMessages(out map[string]string, payload map[string]any) (bool, error) {
	formatJS, err := isStrictFormatJSRoot(payload)
	if err != nil {
		return false, err
	}
	if !formatJS {
		return false, nil
	}

	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		message := payload[key].(map[string]any)
		out[key] = message["defaultMessage"].(string)
	}

	return true, nil
}

func parseFormatJSDescriptions(payload map[string]any) map[string]string {
	out := make(map[string]string)
	for key, value := range payload {
		message, ok := value.(map[string]any)
		if !ok {
			continue
		}
		raw, ok := message["description"]
		if !ok {
			continue
		}
		description, ok := raw.(string)
		if !ok {
			continue
		}
		description = strings.TrimSpace(description)
		if description == "" {
			continue
		}
		out[key] = description
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isStrictFormatJSRoot(payload map[string]any) (bool, error) {
	if len(payload) == 0 {
		return false, nil
	}

	for key, value := range payload {
		message, ok := value.(map[string]any)
		if !ok {
			return false, nil
		}
		raw, ok := message["defaultMessage"]
		if !ok {
			return false, nil
		}
		if _, ok := raw.(string); !ok {
			return false, fmt.Errorf("json key %q field %q must be string, got %T", key, "defaultMessage", raw)
		}
	}

	return true, nil
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
