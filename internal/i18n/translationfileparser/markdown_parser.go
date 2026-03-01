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

		if isImportExportLine(trimmed) {
			doc.parts = append(doc.parts, markdownPart{literal: line})
			continue
		}

		emitMarkdownLineParts(line, &doc, appendKey)
	}

	return doc, entries
}

func isImportExportLine(trimmed string) bool {
	if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "export ") {
		return true
	}
	return trimmed == "import" || trimmed == "export"
}

func emitMarkdownLineParts(line string, doc *markdownDocument, appendKey func(string)) {
	start := 0

	flushText := func(end int) {
		if end <= start {
			return
		}
		emitMarkdownTextParts(line[start:end], doc, appendKey)
		start = end
	}

	for idx := 0; idx < len(line); {
		if line[idx] == '`' {
			flushText(idx)
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
			flushText(idx)
			end := findMarkdownLinkDestinationEnd(line, idx+2)
			doc.parts = append(doc.parts, markdownPart{literal: line[idx:end]})
			idx = end
			start = idx
			continue
		}

		if line[idx] == '{' {
			flushText(idx)
			end := findBraceExpressionEnd(line, idx)
			doc.parts = append(doc.parts, markdownPart{literal: line[idx:end]})
			idx = end
			start = idx
			continue
		}

		if line[idx] == '<' && looksLikeJSXTagStart(line, idx) {
			flushText(idx)
			end := findJSXTagEnd(line, idx)
			doc.parts = append(doc.parts, markdownPart{literal: line[idx:end]})
			idx = end
			start = idx
			continue
		}

		idx++
	}

	flushText(len(line))
}

func findBraceExpressionEnd(line string, start int) int {
	depth := 0
	quote := byte(0)
	escaped := false

	for idx := start; idx < len(line); idx++ {
		ch := line[idx]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return idx + 1
			}
		}
	}

	return len(line)
}

func looksLikeJSXTagStart(line string, idx int) bool {
	if idx+1 >= len(line) {
		return false
	}
	next := line[idx+1]
	if next == '/' || next == '!' || next == '?' {
		return true
	}
	if (next >= 'A' && next <= 'Z') || (next >= 'a' && next <= 'z') {
		if strings.HasPrefix(line[idx+1:], "http") {
			return false
		}
		return true
	}
	return false
}

func findJSXTagEnd(line string, start int) int {
	quote := byte(0)
	escaped := false
	braceDepth := 0

	for idx := start + 1; idx < len(line); idx++ {
		ch := line[idx]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}

		switch ch {
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '>':
			if braceDepth == 0 {
				return idx + 1
			}
		}
	}

	return len(line)
}

func findMarkdownLinkDestinationEnd(line string, start int) int {
	depth := 1
	for idx := start; idx < len(line); idx++ {
		if line[idx] == '\\' {
			idx++
			continue
		}

		switch line[idx] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return idx + 1
			}
		}
	}

	return len(line)
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
