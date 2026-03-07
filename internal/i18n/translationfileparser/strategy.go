package translationfileparser

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Parser parses translation file content into key/value pairs.
type Parser interface {
	Parse(content []byte) (map[string]string, error)
}

// ContextParser optionally returns per-entry context that can be used to enrich prompts.
// The returned map key must match message keys from Parse.
type ContextParser interface {
	ParseWithContext(content []byte) (map[string]string, map[string]string, error)
}

// Strategy selects a parser based on file extension.
type Strategy struct {
	parsersByExt map[string]Parser
}

// NewDefaultStrategy returns a strategy preconfigured for JSON and XLIFF files.
func NewDefaultStrategy() *Strategy {
	s := &Strategy{parsersByExt: map[string]Parser{}}
	s.Register(".json", JSONParser{})
	s.Register(".arb", ARBParser{})
	s.Register(".xlf", XLIFFParser{})
	s.Register(".xlif", XLIFFParser{})
	s.Register(".xliff", XLIFFParser{})
	s.Register(".po", POFileParser{})
	s.Register(".md", MarkdownParser{})
	s.Register(".mdx", MarkdownParser{})
	s.Register(".strings", AppleStringsParser{})
	s.Register(".stringsdict", AppleStringsdictParser{})
	s.Register(".csv", CSVParser{})
	return s
}

// Register binds a parser to a file extension.
func (s *Strategy) Register(ext string, parser Parser) {
	if s.parsersByExt == nil {
		s.parsersByExt = map[string]Parser{}
	}

	normalizedExt := strings.ToLower(strings.TrimSpace(ext))
	if normalizedExt == "" {
		return
	}
	if !strings.HasPrefix(normalizedExt, ".") {
		normalizedExt = "." + normalizedExt
	}

	s.parsersByExt[normalizedExt] = parser
}

// Parse resolves a parser from the file path extension and parses content.
func (s *Strategy) Parse(path string, content []byte) (map[string]string, error) {
	values, _, err := s.ParseWithContext(path, content)
	if err != nil {
		return nil, err
	}

	return values, nil
}

// ParseWithContext resolves a parser from the file path extension and parses content.
// Some parser implementations may return additional per-entry context (for example,
// FormatJS/ARB descriptions).
func (s *Strategy) ParseWithContext(path string, content []byte) (map[string]string, map[string]string, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(path)))
	if ext == "" {
		return nil, nil, fmt.Errorf("translation file parser: file %q has no extension", path)
	}

	parser, ok := s.parsersByExt[ext]
	if !ok {
		return nil, nil, fmt.Errorf("translation file parser: unsupported file extension %q", ext)
	}

	if contextParser, ok := parser.(ContextParser); ok {
		values, entryContext, err := contextParser.ParseWithContext(content)
		if err != nil {
			return nil, nil, fmt.Errorf("translation file parser: parse %q: %w", path, err)
		}
		return values, entryContext, nil
	}

	values, err := parser.Parse(content)
	if err != nil {
		return nil, nil, fmt.Errorf("translation file parser: parse %q: %w", path, err)
	}

	return values, nil, nil
}
