package translationfileparser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// XCStringsParser parses Apple Xcode Strings Catalog (.xcstrings) files.
type XCStringsParser struct{}

func (p XCStringsParser) Parse(content []byte) (map[string]string, error) {
	catalog, err := parseXCStringsCatalog(content)
	if err != nil {
		return nil, err
	}

	out := map[string]string{}
	for _, key := range sortedMapKeys(catalog.Strings) {
		entry, ok := catalog.Strings[key].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("xcstrings entry %q must be an object", key)
		}

		localization, err := selectCatalogLocalization(key, entry, catalog.SourceLanguage)
		if err != nil {
			return nil, err
		}
		if localization == nil {
			continue
		}

		if err := collectCatalogValues(out, key, localization); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func MarshalXCStrings(template []byte, values map[string]string) ([]byte, error) {
	catalog, err := parseXCStringsCatalog(template)
	if err != nil {
		return nil, err
	}

	for _, key := range sortedMapKeys(catalog.Strings) {
		entry, ok := catalog.Strings[key].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("xcstrings entry %q must be an object", key)
		}

		localization, err := selectCatalogLocalization(key, entry, catalog.SourceLanguage)
		if err != nil {
			return nil, err
		}
		if localization == nil {
			continue
		}

		applyCatalogValues(values, key, localization)
	}

	content, err := json.MarshalIndent(catalog.Root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("xcstrings encode: %w", err)
	}
	return append(content, '\n'), nil
}

type xcstringsCatalog struct {
	Root           map[string]any
	SourceLanguage string
	Strings        map[string]any
}

func parseXCStringsCatalog(content []byte) (xcstringsCatalog, error) {
	root := map[string]any{}
	if err := json.Unmarshal(content, &root); err != nil {
		return xcstringsCatalog{}, fmt.Errorf("xcstrings decode: %w", err)
	}

	sourceLanguage, _ := root["sourceLanguage"].(string)
	stringsNode, ok := root["strings"]
	if !ok {
		return xcstringsCatalog{Root: root, SourceLanguage: sourceLanguage, Strings: map[string]any{}}, nil
	}

	stringsMap, ok := stringsNode.(map[string]any)
	if !ok {
		return xcstringsCatalog{}, fmt.Errorf("xcstrings field \"strings\" must be an object")
	}

	return xcstringsCatalog{Root: root, SourceLanguage: sourceLanguage, Strings: stringsMap}, nil
}

func selectCatalogLocalization(key string, entry map[string]any, preferredLocale string) (map[string]any, error) {
	locNode, ok := entry["localizations"]
	if !ok {
		return nil, nil
	}
	localizations, ok := locNode.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("xcstrings entry %q localizations must be an object", key)
	}
	if len(localizations) == 0 {
		return nil, nil
	}

	if preferredLocale != "" {
		if candidate, ok := localizations[preferredLocale]; ok {
			locMap, ok := candidate.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("xcstrings entry %q localization %q must be an object", key, preferredLocale)
			}
			return locMap, nil
		}
	}

	for _, locale := range sortedMapKeys(localizations) {
		candidate, ok := localizations[locale].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("xcstrings entry %q localization %q must be an object", key, locale)
		}
		return candidate, nil
	}
	return nil, nil
}

func collectCatalogValues(out map[string]string, keyPrefix string, node map[string]any) error {
	if stringUnitNode, ok := node["stringUnit"]; ok {
		stringUnit, ok := stringUnitNode.(map[string]any)
		if !ok {
			return fmt.Errorf("xcstrings key %q field \"stringUnit\" must be an object", keyPrefix)
		}
		if value, ok := stringUnit["value"]; ok {
			text, ok := value.(string)
			if !ok {
				return fmt.Errorf("xcstrings key %q field \"value\" must be a string", keyPrefix)
			}
			out[keyPrefix] = text
		}
	}

	variationsNode, ok := node["variations"]
	if !ok {
		return nil
	}

	variations, ok := variationsNode.(map[string]any)
	if !ok {
		return fmt.Errorf("xcstrings key %q field \"variations\" must be an object", keyPrefix)
	}
	for _, variationType := range sortedMapKeys(variations) {
		selectorNode, ok := variations[variationType].(map[string]any)
		if !ok {
			return fmt.Errorf("xcstrings key %q variation %q must be an object", keyPrefix, variationType)
		}
		for _, selector := range sortedMapKeys(selectorNode) {
			nextNode, ok := selectorNode[selector].(map[string]any)
			if !ok {
				return fmt.Errorf("xcstrings key %q variation %q selector %q must be an object", keyPrefix, variationType, selector)
			}
			nextKey := strings.Join([]string{keyPrefix, variationType, selector}, ".")
			if err := collectCatalogValues(out, nextKey, nextNode); err != nil {
				return err
			}
		}
	}

	return nil
}

func applyCatalogValues(values map[string]string, keyPrefix string, node map[string]any) {
	if stringUnitNode, ok := node["stringUnit"].(map[string]any); ok {
		if translated, ok := values[keyPrefix]; ok {
			if _, exists := stringUnitNode["value"]; exists {
				stringUnitNode["value"] = translated
			}
		}
	}

	variationsNode, ok := node["variations"].(map[string]any)
	if !ok {
		return
	}
	for _, variationType := range sortedMapKeys(variationsNode) {
		selectorNode, ok := variationsNode[variationType].(map[string]any)
		if !ok {
			continue
		}
		for _, selector := range sortedMapKeys(selectorNode) {
			nextNode, ok := selectorNode[selector].(map[string]any)
			if !ok {
				continue
			}
			nextKey := strings.Join([]string{keyPrefix, variationType, selector}, ".")
			applyCatalogValues(values, nextKey, nextNode)
		}
	}
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
