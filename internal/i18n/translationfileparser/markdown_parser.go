package translationfileparser

import (
	"fmt"
	"strings"
	"unicode"
)

type markdownPart struct {
	literal string
	key     string
	source  string
}

type markdownDocument struct {
	parts []markdownPart
}

func parseMarkdownDocument(content []byte) (markdownDocument, map[string]string) {
	text := string(content)
	lines := strings.SplitAfter(text, "\n")
	if len(lines) == 0 {
		lines = []string{text}
	}

	doc := markdownDocument{parts: make([]markdownPart, 0, len(lines))}
	entries := map[string]string{}
	keyIndex := 0

	inFrontmatter := len(lines) > 0 && strings.TrimSpace(lines[0]) == "---"

	inFence := false
	fenceMarker := ""

	appendKey := func(segment string) {
		keyIndex++
		key := fmt.Sprintf("md.%04d", keyIndex)
		doc.parts = append(doc.parts, markdownPart{key: key, source: segment})
		entries[key] = segment
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inFrontmatter {
			doc.parts = append(doc.parts, markdownPart{literal: line})
			if i > 0 && trimmed == "---" {
				inFrontmatter = false
			}
			continue
		}

		if inFence {
			doc.parts = append(doc.parts, markdownPart{literal: line})
			if strings.HasPrefix(trimmed, fenceMarker) {
				inFence = false
				fenceMarker = ""
			}
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			inFence = true
			fenceMarker = "```"
			doc.parts = append(doc.parts, markdownPart{literal: line})
			continue
		}
		if strings.HasPrefix(trimmed, "~~~") {
			inFence = true
			fenceMarker = "~~~"
			doc.parts = append(doc.parts, markdownPart{literal: line})
			continue
		}

		start := 0
		for idx := 0; idx < len(line); {
			if strings.HasPrefix(line[idx:], "`") {
				if idx > start {
					emitMarkdownTextParts(line[start:idx], &doc, appendKey)
				}
				end := idx + 1
				for end < len(line) && line[end] != '`' {
					end++
				}
				if end < len(line) {
					end++
				}
				doc.parts = append(doc.parts, markdownPart{literal: line[idx:end]})
				idx = end
				start = idx
				continue
			}

			if strings.HasPrefix(line[idx:], "](") {
				if idx > start {
					emitMarkdownTextParts(line[start:idx], &doc, appendKey)
				}
				end := idx + 2
				for end < len(line) && line[end] != ')' {
					end++
				}
				if end < len(line) {
					end++
				}
				doc.parts = append(doc.parts, markdownPart{literal: line[idx:end]})
				idx = end
				start = idx
				continue
			}

			idx++
		}

		if start < len(line) {
			emitMarkdownTextParts(line[start:], &doc, appendKey)
		}
	}

	return doc, entries
}

func emitMarkdownTextParts(segment string, doc *markdownDocument, appendKey func(string)) {
	start := 0
	flush := func(end int) {
		if end <= start {
			return
		}
		chunk := segment[start:end]
		if isTranslatableChunk(chunk) {
			appendKey(chunk)
		} else {
			doc.parts = append(doc.parts, markdownPart{literal: chunk})
		}
		start = end
	}

	for idx, r := range segment {
		switch r {
		case '#', '>', '*', '_', '|', '[', ']', '(', ')', '!', '-', '+':
			flush(idx)
			next := idx + len(string(r))
			flush(next)
		default:
		}
	}
	flush(len(segment))
}

func isTranslatableChunk(chunk string) bool {
	trimmed := strings.TrimSpace(chunk)
	if trimmed == "" {
		return false
	}
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func (d markdownDocument) render(values map[string]string) []byte {
	var b strings.Builder
	for _, part := range d.parts {
		if part.key == "" {
			b.WriteString(part.literal)
			continue
		}
		if v, ok := values[part.key]; ok {
			b.WriteString(v)
			continue
		}
		b.WriteString(part.source)
	}
	return []byte(b.String())
}

// MarkdownParser parses markdown files into stable key/value text segments.
type MarkdownParser struct{}

func (p MarkdownParser) Parse(content []byte) (map[string]string, error) {
	_, entries := parseMarkdownDocument(content)
	return entries, nil
}

func MarshalMarkdown(template []byte, values map[string]string) []byte {
	doc, _ := parseMarkdownDocument(template)
	return doc.render(values)
}
