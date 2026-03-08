package runsvc

// Output flushing coordinates reading existing targets, merging staged values, and writing final content.

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translationfileparser"
)

func (s *Service) flushOutputs(staged map[string]stagedOutput, pruneTargets map[string]map[string]struct{}, pruneMetadata map[string]stagedOutput) ([]string, error) {
	targetPaths := make([]string, 0, len(staged)+len(pruneTargets))
	for path := range staged {
		targetPaths = append(targetPaths, path)
	}
	for path := range pruneTargets {
		targetPaths = append(targetPaths, path)
	}
	slices.Sort(targetPaths)
	targetPaths = slices.Compact(targetPaths)

	var warnings []string
	for _, targetPath := range targetPaths {
		output, ok := staged[targetPath]
		if !ok {
			output = pruneMetadata[targetPath]
		}
		targetWarnings, err := s.flushOutputForTarget(targetPath, output, pruneTargets[targetPath])
		if err != nil {
			return nil, err
		}
		warnings = append(warnings, targetWarnings...)
	}
	return warnings, nil
}

func (s *Service) flushOutputForTarget(targetPath string, output stagedOutput, keep map[string]struct{}) ([]string, error) {
	values, loadWarnings, err := s.loadExistingTargetWithWarnings(targetPath, output.targetLocale)
	if err != nil {
		return nil, err
	}
	if keep != nil {
		for key := range values {
			if _, ok := keep[key]; !ok {
				delete(values, key)
			}
		}
	}
	if keep == nil {
		maps.Copy(values, output.entries)
	} else {
		for key, value := range output.entries {
			if _, ok := keep[key]; ok {
				values[key] = value
			}
		}
	}

	content, warnings, err := s.marshalTargetFile(targetPath, output.sourcePath, output.sourceLocale, output.targetLocale, values, output.entries, keep)
	if err != nil {
		return nil, err
	}
	if err := s.writeFile(targetPath, content); err != nil {
		return nil, fmt.Errorf("flush outputs: write %q: %w", targetPath, err)
	}
	return append(loadWarnings, warnings...), nil
}

func (s *Service) loadExistingTarget(path string) (map[string]string, error) {
	entries, _, err := s.loadExistingTargetWithWarnings(path, "")
	return entries, err
}

func (s *Service) loadExistingTargetWithWarnings(path, targetLocale string) (map[string]string, []string, error) {
	content, err := s.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil, nil
		}
		return nil, nil, fmt.Errorf("flush outputs: read target file %q: %w", path, err)
	}
	entries, err := parseExistingTargetEntries(path, content, targetLocale, s.newParser())
	if err != nil {
		if ext := strings.ToLower(filepath.Ext(path)); ext == ".json" || ext == ".jsonc" {
			// JSON targets may include non-translatable metadata fields (numbers, bools, arrays).
			// Recover string entries instead of failing the whole run.
			recovered, recoverErr := parseJSONEntriesLenient(path, content)
			if recoverErr == nil {
				return recovered, nil, nil
			}
			// If JSON is malformed, continue with source-template fallback during marshal.
			return map[string]string{}, []string{
				fmt.Sprintf("json target %q is malformed; existing translated values could not be loaded and source-template fallback will be used", path),
			}, nil
		}
		return nil, nil, fmt.Errorf("flush outputs: parse target file %q: %w", path, err)
	}
	return entries, nil, nil
}

func parseExistingTargetEntries(path string, content []byte, targetLocale string, parser *translationfileparser.Strategy) (map[string]string, error) {
	if strings.EqualFold(filepath.Ext(path), ".csv") {
		return parseCSVForTargetLocale(content, targetLocale)
	}
	return parser.Parse(path, content)
}
