package translationfileparser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

type markdownPart struct {
	literal      string
	key          string
	source       string
	placeholders map[string]string
}

type markdownDocument struct {
	parts []markdownPart
}

type markdownParseState struct {
	inJSXTag      bool
	jsxQuote      byte
	jsxEscaped    bool
	jsxBraceDepth int
}

type markdownKeyContext struct {
	text        string
	prevLiteral string
	nextLiteral string
	partIndex   int
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
		out = append(out, markdownKeyContext{text: part.source, prevLiteral: prev, nextLiteral: next, partIndex: i})
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
	state := markdownParseState{}

	appendKey := func(part markdownPart) {
		key := markdownSegmentKey(part.source, hashOccurrences)
		part.key = key
		doc.parts = append(doc.parts, part)
		entries[key] = part.source
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inFrontmatter {
			if i > 0 && trimmed == "---" {
				doc.parts = append(doc.parts, markdownPart{literal: line})
				inFrontmatter = false
				continue
			}
			emitFrontmatterLineParts(line, &doc, func(segment string) {
				appendKey(markdownPart{source: segment})
			})
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

		emitMarkdownLineParts(line, &doc, appendKey, &state)
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

func emitMarkdownLineParts(line string, doc *markdownDocument, appendKey func(markdownPart), state *markdownParseState) {
	prefix := ""
	body := line
	if !state.inJSXTag {
		prefix, body = splitMarkdownLinePrefix(line)
	}
	if prefix != "" {
		doc.parts = append(doc.parts, markdownPart{literal: prefix})
	}
	if body == "" {
		return
	}

	newline := ""
	if strings.HasSuffix(body, "\n") {
		newline = "\n"
		body = strings.TrimSuffix(body, "\n")
	}

	var literal string
	var consumed bool
	literal, body, consumed = consumeLeadingJSXLiteral(body, state)
	if consumed {
		doc.parts = append(doc.parts, markdownPart{literal: literal})
	}

	body, trailingLiterals := stripTrailingJSXClosingLiterals(body)
	for {
		literal, body, consumed = consumeLeadingJSXLiteral(body, state)
		if !consumed {
			break
		}
		doc.parts = append(doc.parts, markdownPart{literal: literal})
	}
	if body == "" {
		for _, literal := range trailingLiterals {
			doc.parts = append(doc.parts, markdownPart{literal: literal})
		}
		if newline != "" {
			doc.parts = append(doc.parts, markdownPart{literal: newline})
		}
		return
	}

	placeholdered, placeholders, plainText := protectMarkdownInlineSyntax(body)
	if !isTranslatableChunk(plainText) {
		doc.parts = append(doc.parts, markdownPart{literal: body})
		for _, literal := range trailingLiterals {
			doc.parts = append(doc.parts, markdownPart{literal: literal})
		}
		if newline != "" {
			doc.parts = append(doc.parts, markdownPart{literal: newline})
		}
		return
	}
	appendKey(markdownPart{source: placeholdered, placeholders: placeholders})
	for _, literal := range trailingLiterals {
		doc.parts = append(doc.parts, markdownPart{literal: literal})
	}
	if newline != "" {
		doc.parts = append(doc.parts, markdownPart{literal: newline})
	}
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

func splitMarkdownLinePrefix(line string) (string, string) {
	idx := 0
	for idx < len(line) && (line[idx] == ' ' || line[idx] == '\t') {
		idx++
	}

	// Preserve one or more blockquote markers as structural prefix.
	for idx < len(line) && line[idx] == '>' {
		idx++
		for idx < len(line) && line[idx] == ' ' {
			idx++
		}
	}

	switch {
	case hasHeadingPrefix(line[idx:]):
		for idx < len(line) && line[idx] == '#' {
			idx++
		}
		if idx < len(line) && line[idx] == ' ' {
			idx++
		}
	case hasBulletPrefix(line[idx:]):
		idx += 2
	case hasOrderedPrefix(line[idx:]):
		for idx < len(line) && line[idx] >= '0' && line[idx] <= '9' {
			idx++
		}
		if idx < len(line) && line[idx] == '.' {
			idx++
		}
		if idx < len(line) && line[idx] == ' ' {
			idx++
		}
	}

	return line[:idx], line[idx:]
}

func hasHeadingPrefix(s string) bool {
	if s == "" || s[0] != '#' {
		return false
	}
	count := 0
	for count < len(s) && s[count] == '#' {
		count++
	}
	return count > 0 && count <= 6 && count < len(s) && s[count] == ' '
}

func hasBulletPrefix(s string) bool {
	return len(s) >= 2 && (s[0] == '-' || s[0] == '+' || s[0] == '*') && s[1] == ' '
}

func hasOrderedPrefix(s string) bool {
	if s == "" || s[0] < '0' || s[0] > '9' {
		return false
	}
	idx := 0
	for idx < len(s) && s[idx] >= '0' && s[idx] <= '9' {
		idx++
	}
	return idx+1 < len(s) && s[idx] == '.' && s[idx+1] == ' '
}

func protectMarkdownInlineSyntax(segment string) (string, map[string]string, string) {
	var rendered strings.Builder
	var plain strings.Builder
	placeholders := map[string]string{}
	placeholderCount := 0

	appendPlaceholder := func(literal string) {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", placeholderCount, literal)))
		placeholder := fmt.Sprintf("\x1eHLMDPH_%s_%d\x1f", strings.ToUpper(hex.EncodeToString(sum[:])[:12]), placeholderCount)
		placeholderCount++
		placeholders[placeholder] = literal
		rendered.WriteString(placeholder)
	}

	for idx := 0; idx < len(segment); {
		switch {
		case segment[idx] == '`':
			end := idx + 1
			for end < len(segment) && segment[end] != '`' {
				end++
			}
			if end < len(segment) {
				end++
			}
			appendPlaceholder(segment[idx:end])
			idx = end
		case strings.HasPrefix(segment[idx:], "]("):
			end := findMarkdownLinkDestinationEnd(segment, idx+2)
			appendPlaceholder(segment[idx:end])
			idx = end
		case segment[idx] == '{':
			end := findBraceExpressionEnd(segment, idx)
			appendPlaceholder(segment[idx:end])
			idx = end
		case segment[idx] == '<' && looksLikeJSXTagStart(segment, idx):
			end := findJSXTagEnd(segment, idx)
			appendPlaceholder(segment[idx:end])
			idx = end
		default:
			rendered.WriteByte(segment[idx])
			plain.WriteByte(segment[idx])
			idx++
		}
	}

	if len(placeholders) == 0 {
		return segment, nil, segment
	}
	return rendered.String(), placeholders, plain.String()
}

func consumeLeadingJSXLiteral(body string, state *markdownParseState) (string, string, bool) {
	if body == "" {
		return "", body, false
	}
	start := 0
	if !state.inJSXTag {
		for start < len(body) && (body[start] == ' ' || body[start] == '\t') {
			start++
		}
		if start >= len(body) || body[start] != '<' || !looksLikeJSXTagStart(body, start) {
			return "", body, false
		}
	}

	end, closed := scanJSXTagFragment(body, start, state)
	if closed {
		state.inJSXTag = false
		state.jsxQuote = 0
		state.jsxEscaped = false
		state.jsxBraceDepth = 0
		return body[:end], body[end:], true
	}
	state.inJSXTag = true
	return body, "", true
}

func scanJSXTagFragment(line string, start int, state *markdownParseState) (int, bool) {
	loopStart := start + 1
	if !state.inJSXTag {
		state.jsxQuote = 0
		state.jsxEscaped = false
		state.jsxBraceDepth = 0
	} else {
		loopStart = start
	}

	for idx := loopStart; idx < len(line); idx++ {
		ch := line[idx]
		if state.jsxQuote != 0 {
			if state.jsxEscaped {
				state.jsxEscaped = false
				continue
			}
			if ch == '\\' {
				state.jsxEscaped = true
				continue
			}
			if ch == state.jsxQuote {
				state.jsxQuote = 0
			}
			continue
		}

		if ch == '\'' || ch == '"' {
			state.jsxQuote = ch
			continue
		}

		switch ch {
		case '{':
			state.jsxBraceDepth++
		case '}':
			if state.jsxBraceDepth > 0 {
				state.jsxBraceDepth--
			}
		case '>':
			if state.jsxBraceDepth == 0 {
				return idx + 1, true
			}
		}
	}

	return len(line), false
}

func stripTrailingJSXClosingLiterals(body string) (string, []string) {
	trailing := []string{}
	for {
		end := len(body)
		for end > 0 && (body[end-1] == ' ' || body[end-1] == '\t') {
			end--
		}
		start := strings.LastIndex(body[:end], "</")
		if start < 0 || !looksLikeJSXTagStart(body, start) {
			return body, trailing
		}
		tagEnd := findJSXTagEnd(body, start)
		if tagEnd != end {
			return body, trailing
		}
		trailing = append([]string{body[start:end]}, trailing...)
		body = body[:start]
	}
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
			b.WriteString(renderMarkdownPart(part, v))
			continue
		}
		b.WriteString(renderMarkdownPart(part, part.source))
	}
	return []byte(b.String())
}

func renderMarkdownPart(part markdownPart, translated string) string {
	rendered := preserveChunkBoundaryWhitespace(part.source, translated)
	if len(part.placeholders) == 0 {
		return rendered
	}
	for placeholder, original := range part.placeholders {
		rendered = strings.ReplaceAll(rendered, placeholder, original)
	}
	return rendered
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
	targetPartUsed := make([]bool, len(targetDoc.parts))
	targetCursor := 0
	sourceContexts := sourceDoc.keyContexts()
	sourceCtxIdx := 0

	takeFallback := func(sourceCtx markdownKeyContext) (string, bool) {
		for i := targetCursor; i < len(targetContexts); i++ {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			if targetContexts[i].prevLiteral == sourceCtx.prevLiteral && targetContexts[i].nextLiteral == sourceCtx.nextLiteral {
				targetPartUsed[targetContexts[i].partIndex] = true
				targetCursor = i + 1
				return targetContexts[i].text, true
			}
		}
		for i := 0; i < len(targetContexts); i++ {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			if targetContexts[i].prevLiteral == sourceCtx.prevLiteral && targetContexts[i].nextLiteral == sourceCtx.nextLiteral {
				targetPartUsed[targetContexts[i].partIndex] = true
				if i >= targetCursor {
					targetCursor = i + 1
				}
				return targetContexts[i].text, true
			}
		}
		for _, startAt := range []int{targetCursor, 0} {
			if fallback, nextCursor, ok := takeMarkdownFallbackSpan(targetDoc, targetPartUsed, startAt, sourceCtx); ok {
				targetCursor = nextCursor
				return fallback, true
			}
		}
		for i := targetCursor; i < len(targetContexts); i++ {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			targetPartUsed[targetContexts[i].partIndex] = true
			targetCursor = i + 1
			return targetContexts[i].text, true
		}
		for i := 0; i < len(targetContexts); i++ {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			targetPartUsed[targetContexts[i].partIndex] = true
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
			b.WriteString(renderMarkdownPart(part, v))
			sourceCtxIdx++
			continue
		}

		// Only consume fallback translations for keys that are part of source extraction.
		// This avoids injecting fallback text into non-translatable structural segments.
		if _, ok := sourceEntries[part.key]; ok && sourceCtxIdx < len(sourceContexts) {
			if fallback, ok := takeFallback(sourceContexts[sourceCtxIdx]); ok {
				b.WriteString(renderMarkdownPart(part, fallback))
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

// AlignMarkdownTargetToSource maps translated target segments back to source-derived markdown keys.
// This is useful for status/reporting where source key identity must remain stable across locales.
func AlignMarkdownTargetToSource(sourceTemplate, targetTemplate []byte) map[string]string {
	sourceDoc, sourceEntries := parseMarkdownDocument(sourceTemplate)
	targetDoc, _ := parseMarkdownDocument(targetTemplate)
	return alignMarkdownFallback(sourceDoc, sourceEntries, targetDoc)
}

func alignMarkdownFallback(sourceDoc markdownDocument, sourceEntries map[string]string, targetDoc markdownDocument) map[string]string {
	targetContexts := targetDoc.keyContexts()
	targetPartUsed := make([]bool, len(targetDoc.parts))
	targetCursor := 0
	sourceContexts := sourceDoc.keyContexts()
	sourceCtxIdx := 0
	aligned := make(map[string]string, len(sourceEntries))

	takeFallback := func(sourceCtx markdownKeyContext) (string, bool) {
		for i := targetCursor; i < len(targetContexts); i++ {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			if targetContexts[i].prevLiteral == sourceCtx.prevLiteral && targetContexts[i].nextLiteral == sourceCtx.nextLiteral {
				targetPartUsed[targetContexts[i].partIndex] = true
				targetCursor = i + 1
				return targetContexts[i].text, true
			}
		}
		for i := range targetContexts {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			if targetContexts[i].prevLiteral == sourceCtx.prevLiteral && targetContexts[i].nextLiteral == sourceCtx.nextLiteral {
				targetPartUsed[targetContexts[i].partIndex] = true
				if i >= targetCursor {
					targetCursor = i + 1
				}
				return targetContexts[i].text, true
			}
		}
		for _, startAt := range []int{targetCursor, 0} {
			if fallback, nextCursor, ok := takeMarkdownFallbackSpan(targetDoc, targetPartUsed, startAt, sourceCtx); ok {
				targetCursor = nextCursor
				return fallback, true
			}
		}
		for i := targetCursor; i < len(targetContexts); i++ {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			targetPartUsed[targetContexts[i].partIndex] = true
			targetCursor = i + 1
			return targetContexts[i].text, true
		}
		for i := range targetContexts {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			targetPartUsed[targetContexts[i].partIndex] = true
			return targetContexts[i].text, true
		}
		return "", false
	}

	for _, part := range sourceDoc.parts {
		if part.key == "" {
			continue
		}

		// Only consume fallback translations for keys that are part of source extraction.
		// This avoids injecting fallback text into non-translatable structural segments.
		if _, ok := sourceEntries[part.key]; ok && sourceCtxIdx < len(sourceContexts) {
			if fallback, ok := takeFallback(sourceContexts[sourceCtxIdx]); ok {
				aligned[part.key] = renderMarkdownPart(part, fallback)
				sourceCtxIdx++
				continue
			}
		}
		if sourceCtxIdx < len(sourceContexts) {
			sourceCtxIdx++
		}
		if _, ok := sourceEntries[part.key]; ok {
			aligned[part.key] = ""
		}
	}

	return aligned
}

func takeMarkdownFallbackSpan(targetDoc markdownDocument, targetPartUsed []bool, startAt int, sourceCtx markdownKeyContext) (string, int, bool) {
	findSpan := func(searchStart int) (int, int, bool) {
		start := searchStart
		if sourceCtx.prevLiteral != "" {
			foundPrev := false
			for i := searchStart; i < len(targetDoc.parts); i++ {
				part := targetDoc.parts[i]
				if part.key == "" && part.literal == sourceCtx.prevLiteral {
					start = i + 1
					foundPrev = true
					break
				}
			}
			if !foundPrev {
				return 0, 0, false
			}
		}

		end := len(targetDoc.parts)
		if sourceCtx.nextLiteral != "" {
			foundNext := false
			for i := start; i < len(targetDoc.parts); i++ {
				part := targetDoc.parts[i]
				if part.key == "" && part.literal == sourceCtx.nextLiteral {
					end = i
					foundNext = true
					break
				}
			}
			if !foundNext {
				return 0, 0, false
			}
		} else {
			for i := start; i < len(targetDoc.parts); i++ {
				part := targetDoc.parts[i]
				if part.key == "" {
					end = i
					break
				}
			}
		}

		if end <= start {
			return 0, 0, false
		}

		for i := start; i < end; i++ {
			if targetPartUsed[i] {
				return 0, 0, false
			}
		}
		return start, end, true
	}

	spanStart, spanEnd, ok := findSpan(startAt)
	if !ok {
		return "", startAt, false
	}

	var b strings.Builder
	for i := spanStart; i < spanEnd; i++ {
		targetPartUsed[i] = true
		if targetDoc.parts[i].key == "" {
			b.WriteString(targetDoc.parts[i].literal)
			continue
		}
		b.WriteString(targetDoc.parts[i].source)
	}

	nextCursor := startAt
	if spanEnd > nextCursor {
		nextCursor = spanEnd
	}
	return b.String(), nextCursor, true
}
