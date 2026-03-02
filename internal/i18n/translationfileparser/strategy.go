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

// Strategy selects a parser based on file extension.
type Strategy struct {
	parsersByExt map[string]Parser
}

// NewDefaultStrategy returns a strategy preconfigured for JSON and XLIFF files.
func NewDefaultStrategy() *Strategy {
	s := &Strategy{parsersByExt: map[string]Parser{}}
	s.Register(".json", JSONParser{})
	s.Register(".xlf", XLIFFParser{})
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
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(path)))
	if ext == "" {
		return nil, fmt.Errorf("translation file parser: file %q has no extension", path)
	}

	parser, ok := s.parsersByExt[ext]
	if !ok {
		return nil, fmt.Errorf("translation file parser: unsupported file extension %q", ext)
	}

	values, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("translation file parser: parse %q: %w", path, err)
	}

	return values, nil
}
