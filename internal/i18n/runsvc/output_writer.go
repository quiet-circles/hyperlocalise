package runsvc

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translationfileparser"
)

func (s *Service) flushOutputs(staged map[string]stagedOutput, pruneTargets map[string]map[string]struct{}) error {
	targetPaths := make([]string, 0, len(staged)+len(pruneTargets))
	for path := range staged {
		targetPaths = append(targetPaths, path)
	}
	for path := range pruneTargets {
		targetPaths = append(targetPaths, path)
	}
	slices.Sort(targetPaths)
	targetPaths = slices.Compact(targetPaths)

	for _, targetPath := range targetPaths {
		if err := s.flushOutputForTarget(targetPath, staged[targetPath], pruneTargets[targetPath]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) flushOutputForTarget(targetPath string, output stagedOutput, keep map[string]struct{}) error {
	values, err := s.loadExistingTarget(targetPath)
	if err != nil {
		return err
	}
	if keep != nil {
		for key := range values {
			if _, ok := keep[key]; !ok {
				delete(values, key)
			}
		}
	}
	maps.Copy(values, output.entries)

	content, err := s.marshalTargetFile(targetPath, output.sourcePath, values)
	if err != nil {
		return err
	}
	if err := s.writeFile(targetPath, content); err != nil {
		return fmt.Errorf("flush outputs: write %q: %w", targetPath, err)
	}
	return nil
}

func buildPlannedTargetKeySet(planned []Task) map[string]map[string]struct{} {
	keep := map[string]map[string]struct{}{}
	for _, task := range planned {
		bucket := keep[task.TargetPath]
		if bucket == nil {
			bucket = map[string]struct{}{}
			keep[task.TargetPath] = bucket
		}
		bucket[task.EntryKey] = struct{}{}
	}
	return keep
}

func (s *Service) planPruneCandidates(pruneTargets map[string]map[string]struct{}) ([]PruneCandidate, error) {
	candidates := make([]PruneCandidate, 0)
	targetPaths := make([]string, 0, len(pruneTargets))
	for path := range pruneTargets {
		targetPaths = append(targetPaths, path)
	}
	slices.Sort(targetPaths)

	for _, targetPath := range targetPaths {
		existing, err := s.loadExistingTarget(targetPath)
		if err != nil {
			return nil, err
		}
		for _, key := range sortedEntryKeys(existing) {
			if _, ok := pruneTargets[targetPath][key]; !ok {
				candidates = append(candidates, PruneCandidate{TargetPath: targetPath, EntryKey: key})
			}
		}
	}
	return candidates, nil
}

func validatePruneLimit(in Input, candidates int) error {
	if !in.Prune || in.DryRun || in.PruneForce {
		return nil
	}
	limit := in.PruneLimit
	if limit <= 0 {
		limit = defaultPruneLimit
	}
	if candidates <= limit {
		return nil
	}
	return fmt.Errorf("prune safety limit exceeded: %d keys scheduled for deletion (limit %d). rerun with --prune-max-deletions %d or --prune-force", candidates, limit, candidates)
}

func (s *Service) loadExistingTarget(path string) (map[string]string, error) {
	content, err := s.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("flush outputs: read target file %q: %w", path, err)
	}
	entries, err := s.newParser().Parse(path, content)
	if err != nil {
		return nil, fmt.Errorf("flush outputs: parse target file %q: %w", path, err)
	}
	return entries, nil
}

func (s *Service) marshalTargetFile(path, sourcePath string, values map[string]string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".mdx", ".strings", ".stringsdict", ".csv":
		return s.marshalTemplateBasedTarget(ext, path, sourcePath, values)
	case ".json":
		return marshalJSONTarget(path, values)
	default:
		return nil, fmt.Errorf("flush outputs: unsupported target file extension %q for %q", ext, path)
	}
}

func (s *Service) marshalTemplateBasedTarget(ext, path, sourcePath string, values map[string]string) ([]byte, error) {
	template, err := s.loadTemplateFallback(path, sourcePath)
	if err != nil {
		return nil, err
	}

	switch ext {
	case ".md", ".mdx":
		return translationfileparser.MarshalMarkdown(template, values), nil
	case ".strings":
		content, err := translationfileparser.MarshalAppleStrings(template, values)
		if err != nil {
			return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
		}
		return content, nil
	case ".stringsdict":
		content, err := translationfileparser.MarshalAppleStringsdict(template, values)
		if err != nil {
			return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
		}
		return content, nil
	case ".csv":
		content, err := translationfileparser.MarshalCSV(template, values, translationfileparser.CSVParser{})
		if err != nil {
			return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
		}
		return content, nil
	default:
		return nil, fmt.Errorf("flush outputs: unsupported target file extension %q for %q", ext, path)
	}
}

func marshalJSONTarget(path string, values map[string]string) ([]byte, error) {
	payload := map[string]any{}
	for _, key := range sortedEntryKeys(values) {
		setNestedValue(payload, key, values[key])
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
	}
	return append(content, '\n'), nil
}

func (s *Service) loadTemplateFallback(targetPath, sourcePath string) ([]byte, error) {
	content, err := s.readFile(targetPath)
	if err == nil {
		return content, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("flush outputs: read target file %q: %w", targetPath, err)
	}
	template, srcErr := s.readFile(sourcePath)
	if srcErr != nil {
		return nil, fmt.Errorf("flush outputs: read template source %q: %w", sourcePath, srcErr)
	}
	return template, nil
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

func writeBytesAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
