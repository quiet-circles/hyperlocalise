package translationfileparser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

// MarshalARB preserves target-template metadata and ordering while carrying
// source-template message metadata for newly appended keys.
func MarshalARB(template []byte, sourceTemplate []byte, values map[string]string) ([]byte, error) {
	fields, err := parseARBObjectFields(template)
	if err != nil {
		return nil, fmt.Errorf("arb decode: %w", err)
	}

	templateMessageKeys := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if isARBMetadataKey(field.Key) {
			continue
		}
		templateMessageKeys[field.Key] = struct{}{}
	}

	sourceMessageMetadata, err := arbMessageMetadataFields(sourceTemplate)
	if err != nil {
		return nil, fmt.Errorf("arb decode: %w", err)
	}

	written := make(map[string]struct{}, len(values))
	var out bytes.Buffer
	out.WriteString("{\n")

	first := true
	writeField := func(key string, value []byte) error {
		if !first {
			out.WriteString(",\n")
		}
		first = false

		encodedKey, err := json.Marshal(key)
		if err != nil {
			return err
		}
		out.WriteString("  ")
		out.Write(encodedKey)
		out.WriteString(": ")
		out.Write(value)
		return nil
	}

	for _, field := range fields {
		if isARBMetadataKey(field.Key) {
			if messageKey, isMessageMeta := arbMessageMetadataKey(field.Key, templateMessageKeys); isMessageMeta {
				if _, ok := values[messageKey]; !ok {
					continue
				}
			}
			if err := writeField(field.Key, field.RawValue); err != nil {
				return nil, fmt.Errorf("arb encode: %w", err)
			}
			continue
		}

		value, ok := values[field.Key]
		if !ok {
			continue
		}
		encodedValue, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("arb encode: %w", err)
		}
		if err := writeField(field.Key, encodedValue); err != nil {
			return nil, fmt.Errorf("arb encode: %w", err)
		}
		written[field.Key] = struct{}{}
	}

	for _, key := range sortedMapKeys(values) {
		if _, ok := written[key]; ok {
			continue
		}
		encodedValue, err := json.Marshal(values[key])
		if err != nil {
			return nil, fmt.Errorf("arb encode: %w", err)
		}
		if err := writeField(key, encodedValue); err != nil {
			return nil, fmt.Errorf("arb encode: %w", err)
		}
		if rawMeta, ok := sourceMessageMetadata[key]; ok {
			if err := writeField("@"+key, rawMeta); err != nil {
				return nil, fmt.Errorf("arb encode: %w", err)
			}
		}
	}

	out.WriteString("\n}\n")
	return out.Bytes(), nil
}

type arbObjectField struct {
	Key      string
	RawValue json.RawMessage
}

func parseARBObjectFields(content []byte) ([]arbObjectField, error) {
	dec := json.NewDecoder(bytes.NewReader(content))

	open, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := open.(json.Delim)
	if !ok || delim != '{' {
		return nil, fmt.Errorf("expected object")
	}

	fields := make([]arbObjectField, 0)
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key")
		}

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		fields = append(fields, arbObjectField{Key: key, RawValue: raw})
	}

	closeToken, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok = closeToken.(json.Delim)
	if !ok || delim != '}' {
		return nil, fmt.Errorf("expected object end")
	}

	// Confirm no tokens remain after the closing '}'.
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("unexpected trailing json tokens")
		}
		return nil, err
	}

	return fields, nil
}

func arbMessageMetadataKey(metaKey string, templateMessageKeys map[string]struct{}) (string, bool) {
	if !strings.HasPrefix(metaKey, "@") || strings.HasPrefix(metaKey, "@@") {
		return "", false
	}

	messageKey := strings.TrimPrefix(metaKey, "@")
	if _, ok := templateMessageKeys[messageKey]; ok {
		return messageKey, true
	}
	return "", false
}

func arbMessageMetadataFields(content []byte) (map[string]json.RawMessage, error) {
	fields, err := parseARBObjectFields(content)
	if err != nil {
		return nil, err
	}

	messageKeys := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if isARBMetadataKey(field.Key) {
			continue
		}
		messageKeys[field.Key] = struct{}{}
	}

	metadataByKey := make(map[string]json.RawMessage)
	for _, field := range fields {
		messageKey, isMessageMeta := arbMessageMetadataKey(field.Key, messageKeys)
		if !isMessageMeta {
			continue
		}
		metadataByKey[messageKey] = field.RawValue
	}
	return metadataByKey, nil
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
