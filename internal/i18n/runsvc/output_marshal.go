package runsvc

// Output marshalling selects per-format writers and fallback templates.

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translationfileparser"
)

func (s *Service) marshalTargetFile(path, sourcePath, sourceLocale, targetLocale string, values map[string]string, stagedEntries map[string]string, pruneKeys map[string]struct{}) ([]byte, []string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".xlf", ".xlif", ".xliff", ".po", ".md", ".mdx", ".strings", ".stringsdict", ".csv", ".arb":
		return s.marshalTemplateBasedTarget(ext, path, sourcePath, sourceLocale, targetLocale, values, stagedEntries)
	case ".json", ".jsonc":
		content, err := s.marshalJSONTargetWithFallback(path, sourcePath, values, pruneKeys)
		return content, nil, err
	default:
		return nil, nil, fmt.Errorf("flush outputs: unsupported target file extension %q for %q", ext, path)
	}
}

func (s *Service) marshalTemplateBasedTarget(ext, path, sourcePath, sourceLocale, targetLocale string, values map[string]string, stagedEntries map[string]string) ([]byte, []string, error) {
	if ext == ".md" || ext == ".mdx" {
		return s.marshalMarkdownTarget(path, sourcePath, stagedEntries)
	}
	if ext == ".xlf" || ext == ".xlif" || ext == ".xliff" || ext == ".po" || ext == ".strings" || ext == ".stringsdict" || ext == ".arb" {
		content, err := s.marshalSourceTemplateTarget(ext, path, sourcePath, sourceLocale, targetLocale, values)
		return content, nil, err
	}

	template, err := s.loadTemplateFallback(path, sourcePath)
	if err != nil {
		return nil, nil, err
	}

	switch ext {
	case ".csv":
		content, err := marshalCSVTarget(template, values, targetLocale)
		if err != nil {
			return nil, nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
		}
		return content, nil, nil
	default:
		return nil, nil, fmt.Errorf("flush outputs: unsupported target file extension %q for %q", ext, path)
	}
}

func (s *Service) marshalSourceTemplateTarget(ext, path, sourcePath, sourceLocale, targetLocale string, values map[string]string) ([]byte, error) {
	sourceTemplate, err := s.readFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("flush outputs: read template source %q: %w", sourcePath, err)
	}

	template := sourceTemplate
	targetTemplate, err := s.readFile(path)
	if err == nil {
		targetEntries, parseErr := s.newParser().Parse(path, targetTemplate)
		if parseErr == nil {
			// For ARB files we always prefer the target template when it parses cleanly,
			// so @@locale, @key attribute blocks, and template-defined ordering are
			// preserved even when the key sets differ. MarshalARB handles new and
			// removed message keys when rewriting the file.
			if ext == ".arb" || hasExactKeySet(targetEntries, values) {
				template = targetTemplate
			}
		}
	}

	switch ext {
	case ".xlf", ".xlif", ".xliff":
		content, err := translationfileparser.MarshalXLIFF(template, values, sourceLocale, targetLocale)
		if err != nil {
			return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
		}
		return content, nil
	case ".po":
		content, err := translationfileparser.MarshalPOFile(template, values)
		if err != nil {
			return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
		}
		return content, nil
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
	case ".arb":
		content, err := translationfileparser.MarshalARB(template, sourceTemplate, values, targetLocale)
		if err != nil {
			return nil, fmt.Errorf("flush outputs: marshal %q: %w", path, err)
		}
		return content, nil
	default:
		return nil, fmt.Errorf("flush outputs: unsupported target file extension %q for %q", ext, path)
	}
}

func hasExactKeySet(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if _, ok := b[key]; !ok {
			return false
		}
	}
	return true
}

func (s *Service) marshalMarkdownTarget(path, sourcePath string, stagedEntries map[string]string) ([]byte, []string, error) {
	sourceTemplate, err := s.readFile(sourcePath)
	if err != nil {
		return nil, nil, fmt.Errorf("flush outputs: read template source %q: %w", sourcePath, err)
	}

	targetTemplate, err := s.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			content, diags := translationfileparser.MarshalMarkdownWithDiagnostics(sourceTemplate, stagedEntries)
			return content, markdownRenderWarnings(path, diags), nil
		}
		return nil, nil, fmt.Errorf("flush outputs: read target file %q: %w", path, err)
	}

	content, diags := translationfileparser.MarshalMarkdownWithTargetFallbackDiagnostics(sourceTemplate, targetTemplate, stagedEntries)
	return content, markdownRenderWarnings(path, diags), nil
}

func markdownRenderWarnings(path string, diags translationfileparser.MarkdownRenderDiagnostics) []string {
	if len(diags.SourceFallbackKeys) == 0 {
		return nil
	}
	keys := slices.Clone(diags.SourceFallbackKeys)
	slices.Sort(keys)
	keys = slices.Compact(keys)
	if len(keys) > 3 {
		return []string{fmt.Sprintf("markdown render fell back to source for %d segments in %q due to unrecoverable placeholder corruption (first keys: %s)", len(keys), path, strings.Join(keys[:3], ", "))}
	}
	return []string{fmt.Sprintf("markdown render fell back to source for %d segments in %q due to unrecoverable placeholder corruption (keys: %s)", len(keys), path, strings.Join(keys, ", "))}
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
