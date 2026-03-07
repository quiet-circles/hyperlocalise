package runsvc

// JSON helpers support nested-key translation updates, pruning, and lenient recovery.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	jsoncparser "github.com/tidwall/jsonc"
)

func unmarshalJSONForPath(path string, content []byte, out any) error {
	firstErr := json.Unmarshal(content, out)
	if firstErr == nil {
		return nil
	}
	if strings.EqualFold(filepath.Ext(path), ".jsonc") {
		return json.Unmarshal(jsoncparser.ToJSON(content), out)
	}
	return firstErr
}
func marshalJSONTarget(path string, template []byte, values map[string]string, pruneKeys map[string]struct{}) ([]byte, error) {
	var payload map[string]any
	if err := unmarshalJSONForPath(path, template, &payload); err != nil {
		return nil, fmt.Errorf("flush outputs: decode template %q: %w", path, err)
	}
	if payload == nil {
		payload = map[string]any{}
	}

	allowedValues := values
	if pruneKeys != nil {
		allowedValues = make(map[string]string, len(values))
		for key, value := range values {
			if _, ok := pruneKeys[key]; ok {
				allowedValues[key] = value
			}
		}
	}

	if isStrictFormatJSTemplate(payload) {
		if pruneKeys != nil {
			pruneFormatJSEntries(payload, pruneKeys)
		}
		applyFormatJSUpdates(payload, allowedValues)
	} else {
		if pruneKeys != nil {
			pruneNestedJSONStringFields(payload, "", pruneKeys)
		}
		applyNestedJSONTranslations(payload, allowedValues)
	}

	// Note: JSONC comments/trailing commas are not preserved on write-back.
	// We always emit canonical JSON syntax (while allowing .jsonc extension).
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
	}
	return append(content, '\n'), nil
}

func (s *Service) marshalJSONTargetWithFallback(path, sourcePath string, values map[string]string, pruneKeys map[string]struct{}) ([]byte, error) {
	targetTemplate, err := s.readFile(path)
	if err == nil {
		content, marshalErr := marshalJSONTarget(path, targetTemplate, values, pruneKeys)
		if marshalErr == nil {
			return content, nil
		}

		sourceTemplate, srcErr := s.readFile(sourcePath)
		if srcErr != nil {
			return nil, fmt.Errorf("flush outputs: read template source %q: %w", sourcePath, srcErr)
		}
		fallbackContent, fallbackErr := marshalJSONTarget(path, sourceTemplate, values, pruneKeys)
		if fallbackErr == nil {
			return fallbackContent, nil
		}
		return nil, errors.Join(
			marshalErr,
			fmt.Errorf("flush outputs: fallback template %q: %w", sourcePath, fallbackErr),
		)
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("flush outputs: read target file %q: %w", path, err)
	}

	sourceTemplate, srcErr := s.readFile(sourcePath)
	if srcErr != nil {
		return nil, fmt.Errorf("flush outputs: read template source %q: %w", sourcePath, srcErr)
	}
	return marshalJSONTarget(path, sourceTemplate, values, pruneKeys)
}

func isStrictFormatJSTemplate(payload map[string]any) bool {
	if len(payload) == 0 {
		return false
	}

	for _, raw := range payload {
		message, ok := raw.(map[string]any)
		if !ok {
			return false
		}
		defaultMessage, ok := message["defaultMessage"]
		if !ok {
			return false
		}
		if _, ok := defaultMessage.(string); !ok {
			return false
		}
	}
	return true
}

func pruneFormatJSEntries(payload map[string]any, keep map[string]struct{}) {
	for key, raw := range payload {
		if _, ok := keep[key]; ok {
			continue
		}
		message, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if _, ok := message["defaultMessage"]; ok {
			delete(payload, key)
		}
	}
}

func applyFormatJSUpdates(payload map[string]any, values map[string]string) {
	for _, key := range sortedEntryKeys(values) {
		raw, ok := payload[key]
		if !ok {
			payload[key] = map[string]any{"defaultMessage": values[key]}
			continue
		}
		message, ok := raw.(map[string]any)
		if !ok {
			payload[key] = map[string]any{"defaultMessage": values[key]}
			continue
		}
		message["defaultMessage"] = values[key]
	}
}

func applyNestedJSONTranslations(payload map[string]any, values map[string]string) {
	for _, key := range sortedEntryKeys(values) {
		setNestedValue(payload, key, values[key])
	}
}

func pruneNestedJSONStringFields(payload map[string]any, prefix string, allowed map[string]struct{}) {
	for _, key := range sortedEntryKeysMapAny(payload) {
		value := payload[key]
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch typed := value.(type) {
		case string:
			if _, ok := allowed[fullKey]; !ok {
				delete(payload, key)
			}
		case map[string]any:
			pruneNestedJSONStringFields(typed, fullKey, allowed)
			if len(typed) == 0 {
				delete(payload, key)
			}
		}
	}
}

func parseJSONEntriesLenient(path string, content []byte) (map[string]string, error) {
	var payload map[string]any
	if err := unmarshalJSONForPath(path, content, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return map[string]string{}, nil
	}

	out := map[string]string{}
	if isStrictFormatJSTemplate(payload) {
		for _, key := range sortedEntryKeysMapAny(payload) {
			message := payload[key].(map[string]any)
			raw, ok := message["defaultMessage"].(string)
			if ok {
				out[key] = raw
			}
		}
		return out, nil
	}
	collectNestedJSONStrings(out, "", payload)
	return out, nil
}

func collectNestedJSONStrings(out map[string]string, prefix string, payload map[string]any) {
	for _, key := range sortedEntryKeysMapAny(payload) {
		value := payload[key]
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch typed := value.(type) {
		case string:
			out[fullKey] = typed
		case map[string]any:
			collectNestedJSONStrings(out, fullKey, typed)
		}
	}
}

func sortedEntryKeysMapAny(entries map[string]any) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func setNestedValue(payload map[string]any, dottedKey, value string) {
	parts := strings.Split(dottedKey, ".")
	current := payload
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		next, ok := current[part]
		if !ok {
			nested := map[string]any{}
			current[part] = nested
			current = nested
			continue
		}
		nested, ok := next.(map[string]any)
		if !ok {
			nested = map[string]any{}
			current[part] = nested
		}
		current = nested
	}
}
