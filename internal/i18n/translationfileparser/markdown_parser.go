package translationfileparser

import (
	"crypto/sha256"
	"encoding/hex"
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

type markdownKeyContext struct {
	text        string
	prevLiteral string
	nextLiteral string
}

func (d markdownDocument) keyContexts() []markdownKeyContext {
	out := make([]markdownKeyContext, 0)
	for i, part := range d.parts {
		if part.key == "" {
			continue
		}
		prev := ""
		if i > 0 && d.parts[i-1].key == "" {
			prev = d.parts[i-1].literal
		}
		next := ""
		if i+1 < len(d.parts) && d.parts[i+1].key == "" {
			next = d.parts[i+1].literal
		}
		out = append(out, markdownKeyContext{text: part.source, prevLiteral: prev, nextLiteral: next})
	}
	return out
}

func parseMarkdownDocument(content []byte) (markdownDocument, map[string]string) {
	text := string(content)
	lines := strings.SplitAfter(text, "\n")
	if len(lines) == 0 {
		lines = []string{text}
	}

	doc := markdownDocument{parts: make([]markdownPart, 0, len(lines))}
	entries := map[string]string{}
	hashOccurrences := map[string]int{}

	inFrontmatter := len(lines) > 0 && strings.TrimSpace(lines[0]) == "---"

	inFence := false
	fenceMarker := ""

	appendKey := func(segment string) {
		key := markdownSegmentKey(segment, hashOccurrences)
		doc.parts = append(doc.parts, markdownPart{key: key, source: segment})
		entries[key] = segment
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inFrontmatter {
			if i > 0 && trimmed == "---" {
				doc.parts = append(doc.parts, markdownPart{literal: line})
				inFrontmatter = false
				continue
			}
			emitFrontmatterLineParts(line, &doc, appendKey)
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

func markdownSegmentKey(segment string, occurrences map[string]int) string {
	sum := sha256.Sum256([]byte(segment))
	hash := hex.EncodeToString(sum[:])[:16]
	count := occurrences[hash]
	occurrences[hash] = count + 1
	if count == 0 {
		return fmt.Sprintf("md.%s", hash)
	}
	return fmt.Sprintf("md.%s.%d", hash, count+1)
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
		return !strings.HasPrefix(line[idx+1:], "http")
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

func emitFrontmatterLineParts(line string, doc *markdownDocument, appendKey func(string)) {
	if strings.TrimSpace(line) == "" {
		doc.parts = append(doc.parts, markdownPart{literal: line})
		return
	}

	newline := ""
	body := line
	if strings.HasSuffix(body, "\n") {
		newline = "\n"
		body = strings.TrimSuffix(body, "\n")
	}

	colon := strings.IndexByte(body, ':')
	if colon <= 0 {
		doc.parts = append(doc.parts, markdownPart{literal: line})
		return
	}

	key := strings.TrimSpace(body[:colon])
	if key == "" {
		doc.parts = append(doc.parts, markdownPart{literal: line})
		return
	}

	valuePart := body[colon+1:]
	lead := len(valuePart) - len(strings.TrimLeftFunc(valuePart, unicode.IsSpace))
	if lead >= len(valuePart) {
		doc.parts = append(doc.parts, markdownPart{literal: line})
		return
	}

	valueRest := valuePart[lead:]
	if len(valueRest) < 2 {
		doc.parts = append(doc.parts, markdownPart{literal: line})
		return
	}

	quote := valueRest[0]
	if quote != '"' && quote != '\'' {
		doc.parts = append(doc.parts, markdownPart{literal: line})
		return
	}

	end := findQuotedStringEnd(valueRest, quote)
	if end <= 1 {
		doc.parts = append(doc.parts, markdownPart{literal: line})
		return
	}

	quotedText := valueRest[1:end]
	if !isTranslatableChunk(quotedText) {
		doc.parts = append(doc.parts, markdownPart{literal: line})
		return
	}

	doc.parts = append(doc.parts, markdownPart{literal: body[:colon+1] + valuePart[:lead] + string(quote)})
	appendKey(quotedText)
	doc.parts = append(doc.parts, markdownPart{literal: valueRest[end:] + newline})
}

func findQuotedStringEnd(s string, quote byte) int {
	escaped := false
	for i := 1; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && quote == '"' {
			escaped = true
			continue
		}
		if ch == quote {
			return i
		}
	}
	return -1
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
			b.WriteString(preserveChunkBoundaryWhitespace(part.source, v))
			continue
		}
		b.WriteString(part.source)
	}
	return []byte(b.String())
}

func preserveChunkBoundaryWhitespace(source, translated string) string {
	leadEnd := len(source) - len(strings.TrimLeftFunc(source, unicode.IsSpace))
	trailStart := len(strings.TrimRightFunc(source, unicode.IsSpace))
	core := strings.TrimFunc(translated, unicode.IsSpace)
	return source[:leadEnd] + core + source[trailStart:]
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

// MarshalMarkdownWithTargetFallback renders markdown from the source template so new
// sections are included, while preserving existing target translations for entries
// not updated in the current run.
func MarshalMarkdownWithTargetFallback(sourceTemplate, targetTemplate []byte, values map[string]string) []byte {
	sourceDoc, sourceEntries := parseMarkdownDocument(sourceTemplate)
	targetDoc, _ := parseMarkdownDocument(targetTemplate)
	targetContexts := targetDoc.keyContexts()
	targetUsed := make([]bool, len(targetContexts))
	targetCursor := 0
	sourceContexts := sourceDoc.keyContexts()
	sourceCtxIdx := 0

	takeFallback := func(sourceCtx markdownKeyContext) (string, bool) {
		for i := targetCursor; i < len(targetContexts); i++ {
			if targetUsed[i] {
				continue
			}
			if targetContexts[i].prevLiteral == sourceCtx.prevLiteral && targetContexts[i].nextLiteral == sourceCtx.nextLiteral {
				targetUsed[i] = true
				targetCursor = i + 1
				return targetContexts[i].text, true
			}
		}
		for i := 0; i < len(targetContexts); i++ {
			if targetUsed[i] {
				continue
			}
			if targetContexts[i].prevLiteral == sourceCtx.prevLiteral && targetContexts[i].nextLiteral == sourceCtx.nextLiteral {
				targetUsed[i] = true
				if i >= targetCursor {
					targetCursor = i + 1
				}
				return targetContexts[i].text, true
			}
		}
		for i := targetCursor; i < len(targetContexts); i++ {
			if targetUsed[i] {
				continue
			}
			targetUsed[i] = true
			targetCursor = i + 1
			return targetContexts[i].text, true
		}
		for i := 0; i < len(targetContexts); i++ {
			if targetUsed[i] {
				continue
			}
			targetUsed[i] = true
			return targetContexts[i].text, true
		}
		return "", false
	}

	var b strings.Builder
	for _, part := range sourceDoc.parts {
		if part.key == "" {
			b.WriteString(part.literal)
			continue
		}

		if v, ok := values[part.key]; ok {
			b.WriteString(preserveChunkBoundaryWhitespace(part.source, v))
			sourceCtxIdx++
			continue
		}

		// Only consume fallback translations for keys that are part of source extraction.
		// This avoids injecting fallback text into non-translatable structural segments.
		if _, ok := sourceEntries[part.key]; ok && sourceCtxIdx < len(sourceContexts) {
			if fallback, ok := takeFallback(sourceContexts[sourceCtxIdx]); ok {
				b.WriteString(preserveChunkBoundaryWhitespace(part.source, fallback))
				sourceCtxIdx++
				continue
			}
		}
		if sourceCtxIdx < len(sourceContexts) {
			sourceCtxIdx++
		}
		b.WriteString(part.source)
	}

	return []byte(b.String())
}
