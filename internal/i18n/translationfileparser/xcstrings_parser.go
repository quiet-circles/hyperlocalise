package translationfileparser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// XCStringsParser parses Apple Strings Catalog (.xcstrings) files.
type XCStringsParser struct{}

func (p XCStringsParser) Parse(content []byte) (map[string]string, error) {
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("xcstrings decode: %w", err)
	}
	if payload == nil {
		return map[string]string{}, nil
	}

	rawStrings, ok := payload["strings"]
	if !ok {
		return map[string]string{}, nil
	}
	stringsMap, ok := rawStrings.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("xcstrings field %q must be object, got %T", "strings", rawStrings)
	}

	sourceLanguage, _ := payload["sourceLanguage"].(string)
	out := map[string]string{}
	for _, key := range sortedMapKeys(stringsMap) {
		rawEntry := stringsMap[key]
		entry, ok := rawEntry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("xcstrings key %q must be object, got %T", key, rawEntry)
		}
		localization, err := selectXCStringsLocalization(entry, sourceLanguage)
		if err != nil {
			return nil, err
		}
		if localization == nil {
			continue
		}
		if err := collectXCStringsValues(out, key, localization); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// MarshalXCStrings preserves catalog metadata and updates only localized string values.
func MarshalXCStrings(template []byte, values map[string]string, targetLocale string) ([]byte, error) {
	if strings.TrimSpace(targetLocale) == "" {
		return nil, fmt.Errorf("xcstrings target locale is required")
	}

	var payload map[string]any
	if err := json.Unmarshal(template, &payload); err != nil {
		return nil, fmt.Errorf("xcstrings decode: %w", err)
	}
	if payload == nil {
		payload = map[string]any{}
	}

	rawStrings, ok := payload["strings"]
	if !ok {
		rawStrings = map[string]any{}
		payload["strings"] = rawStrings
	}
	stringsMap, ok := rawStrings.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("xcstrings field %q must be object, got %T", "strings", rawStrings)
	}

	sourceLanguage, _ := payload["sourceLanguage"].(string)
	for _, key := range sortedMapKeys(stringsMap) {
		rawEntry := stringsMap[key]
		entry, ok := rawEntry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("xcstrings key %q must be object, got %T", key, rawEntry)
		}

		localization, err := selectXCStringsLocalizationForMarshal(entry, sourceLanguage, targetLocale)
		if err != nil {
			return nil, err
		}
		if localization == nil {
			continue
		}
		if err := applyXCStringsValues(localization, key, values); err != nil {
			return nil, err
		}
	}

	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("xcstrings encode: %w", err)
	}
	return append(content, '\n'), nil
}

func selectXCStringsLocalization(entry map[string]any, sourceLanguage string) (map[string]any, error) {
	rawLocalizations, ok := entry["localizations"]
	if !ok {
		return nil, nil
	}
	localizations, ok := rawLocalizations.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("xcstrings field %q must be object, got %T", "localizations", rawLocalizations)
	}
	if len(localizations) == 0 {
		return nil, nil
	}

	if sourceLanguage != "" {
		if rawLoc, ok := localizations[sourceLanguage]; ok {
			loc, ok := rawLoc.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("xcstrings localization %q must be object, got %T", sourceLanguage, rawLoc)
			}
			return loc, nil
		}
	}

	locales := sortedMapKeys(localizations)
	if len(locales) == 0 {
		return nil, nil
	}

	locale := locales[0]
	rawLoc := localizations[locale]
	loc, ok := rawLoc.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("xcstrings localization %q must be object, got %T", locale, rawLoc)
	}
	return loc, nil
}

func selectXCStringsLocalizationForMarshal(entry map[string]any, sourceLanguage, targetLocale string) (map[string]any, error) {
	if strings.TrimSpace(targetLocale) == "" {
		return nil, fmt.Errorf("xcstrings target locale is required")
	}

	rawLocalizations, ok := entry["localizations"]
	if !ok {
		return nil, nil
	}
	localizations, ok := rawLocalizations.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("xcstrings field %q must be object, got %T", "localizations", rawLocalizations)
	}
	if len(localizations) == 0 {
		return nil, nil
	}

	if rawLoc, ok := localizations[targetLocale]; ok {
		loc, ok := rawLoc.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("xcstrings localization %q must be object, got %T", targetLocale, rawLoc)
		}
		return loc, nil
	}

	base, err := selectXCStringsLocalization(entry, sourceLanguage)
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, nil
	}

	cloned := cloneXCStringsObject(base)
	resetXCStringsState(cloned, "needs_review")
	localizations[targetLocale] = cloned
	return cloned, nil
}

func collectXCStringsValues(out map[string]string, key string, node map[string]any) error {
	rawUnit, hasUnit := node["stringUnit"]
	if hasUnit {
		unit, ok := rawUnit.(map[string]any)
		if !ok {
			return fmt.Errorf("xcstrings field %q must be object, got %T", "stringUnit", rawUnit)
		}
		rawValue, ok := unit["value"]
		if ok {
			value, ok := rawValue.(string)
			if !ok {
				return fmt.Errorf("xcstrings key %q field %q must be string, got %T", key, "value", rawValue)
			}
			out[key] = value
		}
	}

	rawVariations, hasVariations := node["variations"]
	if !hasVariations {
		return nil
	}
	variations, ok := rawVariations.(map[string]any)
	if !ok {
		return fmt.Errorf("xcstrings field %q must be object, got %T", "variations", rawVariations)
	}

	for _, dimension := range sortedMapKeys(variations) {
		rawSelectors := variations[dimension]
		selectors, ok := rawSelectors.(map[string]any)
		if !ok {
			return fmt.Errorf("xcstrings field %q must be object, got %T", key+"."+dimension, rawSelectors)
		}
		for _, selector := range sortedMapKeys(selectors) {
			rawChild := selectors[selector]
			child, ok := rawChild.(map[string]any)
			if !ok {
				return fmt.Errorf("xcstrings field %q must be object, got %T", key+"."+dimension+"."+selector, rawChild)
			}
			if err := collectXCStringsValues(out, key+"."+dimension+"."+selector, child); err != nil {
				return err
			}
		}
	}

	return nil
}

func applyXCStringsValues(node map[string]any, key string, values map[string]string) error {
	rawUnit, hasUnit := node["stringUnit"]
	if hasUnit {
		unit, ok := rawUnit.(map[string]any)
		if !ok {
			return fmt.Errorf("xcstrings field %q must be object, got %T", "stringUnit", rawUnit)
		}
		if translated, ok := values[key]; ok {
			currentValue, hasCurrent := unit["value"].(string)
			unit["value"] = translated
			if !hasCurrent || currentValue != translated {
				unit["state"] = "needs_review"
			}
		}
	}

	rawVariations, hasVariations := node["variations"]
	if !hasVariations {
		return nil
	}
	variations, ok := rawVariations.(map[string]any)
	if !ok {
		return fmt.Errorf("xcstrings field %q must be object, got %T", "variations", rawVariations)
	}

	for _, dimension := range sortedMapKeys(variations) {
		rawSelectors := variations[dimension]
		selectors, ok := rawSelectors.(map[string]any)
		if !ok {
			return fmt.Errorf("xcstrings field %q must be object, got %T", key+"."+dimension, rawSelectors)
		}
		for _, selector := range sortedMapKeys(selectors) {
			rawChild := selectors[selector]
			child, ok := rawChild.(map[string]any)
			if !ok {
				return fmt.Errorf("xcstrings field %q must be object, got %T", key+"."+dimension+"."+selector, rawChild)
			}
			if err := applyXCStringsValues(child, key+"."+dimension+"."+selector, values); err != nil {
				return err
			}
		}
	}

	return nil
}

func cloneXCStringsObject(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = cloneXCStringsValue(value)
	}
	return out
}

func cloneXCStringsArray(input []any) []any {
	out := make([]any, len(input))
	for i, value := range input {
		out[i] = cloneXCStringsValue(value)
	}
	return out
}

func cloneXCStringsValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneXCStringsObject(typed)
	case []any:
		return cloneXCStringsArray(typed)
	default:
		return typed
	}
}

func resetXCStringsState(node map[string]any, state string) {
	rawUnit, hasUnit := node["stringUnit"]
	if hasUnit {
		if unit, ok := rawUnit.(map[string]any); ok {
			unit["state"] = state
		}
	}

	rawVariations, hasVariations := node["variations"]
	if !hasVariations {
		return
	}
	variations, ok := rawVariations.(map[string]any)
	if !ok {
		return
	}

	for _, dimension := range sortedMapKeys(variations) {
		rawSelectors := variations[dimension]
		selectors, ok := rawSelectors.(map[string]any)
		if !ok {
			continue
		}
		for _, selector := range sortedMapKeys(selectors) {
			rawChild := selectors[selector]
			child, ok := rawChild.(map[string]any)
			if !ok {
				continue
			}
			resetXCStringsState(child, state)
		}
	}
}
