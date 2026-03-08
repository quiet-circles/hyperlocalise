package translationfileparser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var markdownPlaceholderPattern = regexp.MustCompile("\x1eHLMDPH_[A-Z0-9_]+_(\\d+)\x1f")

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

type MarkdownRenderDiagnostics struct {
	SourceFallbackKeys []string
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
		out = append(out, markdownKeyContext{text: renderMarkdownPart(part, part.source), prevLiteral: prev, nextLiteral: next, partIndex: i})
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
	if len(s) < 2 || (s[0] != '-' && s[0] != '+' && s[0] != '*') || s[1] != ' ' {
		return false
	}
	if (s[0] == '-' || s[0] == '*') && isThematicBreak(s) {
		return false
	}
	return true
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
		// Placeholder sentinels are exposed through Parse() and must survive translation
		// round-trips so renderMarkdownPart can restore protected markdown/JSX literals.
		placeholders[placeholder] = literal
		rendered.WriteString(placeholder)
	}

	for idx := 0; idx < len(segment); {
		if idx == 0 {
			if start, end, ok := findMarkdownReferenceDefinitionDestination(segment); ok {
				rendered.WriteString(segment[idx:start])
				plain.WriteString(segment[idx:start])
				appendPlaceholder(segment[start:end])
				idx = end
				continue
			}
		}

		switch {
		case segment[idx] == '`':
			run := 0
			for idx+run < len(segment) && segment[idx+run] == '`' {
				run++
			}
			end := idx + run
			closing := strings.Repeat("`", run)
			found := false
			for end <= len(segment)-run {
				if segment[end:end+run] == closing {
					end += run
					found = true
					break
				}
				end++
			}
			if !found {
				rendered.WriteString(segment[idx : idx+run])
				plain.WriteString(segment[idx : idx+run])
				idx += run
				continue
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
			// This protection pass is single-line in scope. If an inline JSX tag
			// does not close within the current segment, the rest of the segment is
			// protected here; multi-line inline JSX continuation is handled by the
			// parser state in emitMarkdownLineParts.
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

func findMarkdownReferenceDefinitionDestination(segment string) (int, int, bool) {
	trimmed := strings.TrimLeft(segment, " \t")
	leading := len(segment) - len(trimmed)
	if !strings.HasPrefix(trimmed, "[") {
		return 0, 0, false
	}
	closeBracket := strings.IndexByte(trimmed, ']')
	if closeBracket <= 1 || closeBracket+1 >= len(trimmed) || trimmed[closeBracket+1] != ':' {
		return 0, 0, false
	}

	destStart := closeBracket + 2
	for destStart < len(trimmed) && (trimmed[destStart] == ' ' || trimmed[destStart] == '\t') {
		destStart++
	}
	if destStart >= len(trimmed) {
		return 0, 0, false
	}

	destEnd := destStart
	if trimmed[destStart] == '<' {
		destEnd = strings.IndexByte(trimmed[destStart+1:], '>')
		if destEnd < 0 {
			return 0, 0, false
		}
		destEnd += destStart + 2
	} else {
		for destEnd < len(trimmed) && trimmed[destEnd] != ' ' && trimmed[destEnd] != '\t' {
			destEnd++
		}
	}

	return leading + destStart, leading + destEnd, true
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

func (d markdownDocument) render(values map[string]string) ([]byte, MarkdownRenderDiagnostics) {
	var diags MarkdownRenderDiagnostics
	var b strings.Builder
	for _, part := range d.parts {
		if part.key == "" {
			b.WriteString(part.literal)
			continue
		}
		if v, ok := values[part.key]; ok {
			b.WriteString(renderMarkdownPartWithDiagnostics(part, v, &diags))
			continue
		}
		b.WriteString(renderMarkdownPartWithDiagnostics(part, part.source, &diags))
	}
	return []byte(b.String()), diags
}

func renderMarkdownPart(part markdownPart, translated string) string {
	return renderMarkdownPartWithDiagnostics(part, translated, nil)
}

func renderMarkdownPartWithDiagnostics(part markdownPart, translated string, diags *MarkdownRenderDiagnostics) string {
	rendered := preserveChunkBoundaryWhitespace(part.source, translated)
	rendered = normalizeMarkdownTableRowBoundaries(part, rendered)
	if len(part.placeholders) == 0 {
		return rendered
	}
	rendered = expandMarkdownPlaceholders(rendered, part.placeholders)
	rendered = normalizeMarkdownPlaceholders(rendered, part.placeholders)
	rendered = normalizeUnexpectedMarkdownLinkClosers(part, rendered)
	rendered = restoreSourceReferenceDefinitionDestination(part, rendered)
	if strings.ContainsRune(rendered, '\x1e') || strings.ContainsRune(rendered, '\x1f') {
		// If a translation corrupts placeholder sentinels beyond recovery, emit the
		// original source markdown for this segment instead of leaking control tokens.
		if diags != nil && part.key != "" {
			diags.SourceFallbackKeys = append(diags.SourceFallbackKeys, part.key)
		}
		return expandMarkdownPlaceholders(part.source, part.placeholders)
	}
	return rendered
}

func normalizeMarkdownTableRowBoundaries(part markdownPart, rendered string) string {
	sourceTrimmed := strings.TrimSpace(part.source)
	if !strings.HasPrefix(sourceTrimmed, "|") {
		return rendered
	}
	if strings.Count(sourceTrimmed, "|") < 2 {
		return rendered
	}

	lead := len(rendered) - len(strings.TrimLeftFunc(rendered, unicode.IsSpace))
	trail := len(rendered) - len(strings.TrimRightFunc(rendered, unicode.IsSpace))
	core := strings.TrimSpace(rendered)
	if core == "" {
		return rendered
	}

	if !strings.HasPrefix(core, "|") {
		core = "| " + strings.TrimLeft(core, " ")
	}
	if !strings.HasSuffix(core, "|") {
		core = strings.TrimRight(core, " ") + " |"
	}

	return rendered[:lead] + core + rendered[len(rendered)-trail:]
}

func normalizeUnexpectedMarkdownLinkClosers(part markdownPart, rendered string) string {
	for placeholder, original := range part.placeholders {
		if !strings.HasPrefix(original, "](") {
			continue
		}
		if strings.Contains(part.source, placeholder+"]") {
			continue
		}
		rendered = strings.ReplaceAll(rendered, original+"]", original)
		rendered = strings.ReplaceAll(rendered, original+" ]", original)
	}
	return rendered
}

func restoreSourceReferenceDefinitionDestination(part markdownPart, rendered string) string {
	sourceStart, sourceEnd, ok := findMarkdownReferenceDefinitionDestination(part.source)
	if !ok {
		return rendered
	}
	sourceDestination := part.source[sourceStart:sourceEnd]
	if expanded, ok := part.placeholders[sourceDestination]; ok {
		sourceDestination = expanded
	}

	renderedStart, renderedEnd, ok := findMarkdownReferenceDefinitionDestination(rendered)
	if !ok {
		return rendered
	}
	if rendered[renderedStart:renderedEnd] == sourceDestination {
		return rendered
	}

	return rendered[:renderedStart] + sourceDestination + rendered[renderedEnd:]
}

func expandMarkdownPlaceholders(rendered string, placeholders map[string]string) string {
	for placeholder, original := range placeholders {
		rendered = strings.ReplaceAll(rendered, placeholder, original)
	}
	return rendered
}

func normalizeMarkdownPlaceholders(rendered string, placeholders map[string]string) string {
	if !strings.Contains(rendered, "\x1eHLMDPH_") {
		return rendered
	}
	// Only recover by index when there is exactly one placeholder in the part.
	// With multiple placeholders, index corruption could silently substitute the
	// wrong literal, so we intentionally fail closed to source fallback.
	if len(placeholders) != 1 {
		return rendered
	}
	var (
		expectedIdx int
		original    string
		ok          bool
	)
	for placeholder, v := range placeholders {
		match := markdownPlaceholderPattern.FindStringSubmatch(placeholder)
		if len(match) != 2 {
			return rendered
		}
		idx, err := strconv.Atoi(match[1])
		if err != nil {
			return rendered
		}
		expectedIdx = idx
		original = v
		ok = true
	}
	if !ok {
		return rendered
	}

	return markdownPlaceholderPattern.ReplaceAllStringFunc(rendered, func(token string) string {
		match := markdownPlaceholderPattern.FindStringSubmatch(token)
		if len(match) != 2 {
			return token
		}
		idx, err := strconv.Atoi(match[1])
		if err != nil {
			return token
		}
		if idx == expectedIdx {
			return original
		}
		return token
	})
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
	content, _ := MarshalMarkdownWithDiagnostics(template, values)
	return content
}

func MarshalMarkdownWithDiagnostics(template []byte, values map[string]string) ([]byte, MarkdownRenderDiagnostics) {
	doc, _ := parseMarkdownDocument(template)
	return doc.render(values)
}

func isThematicBreak(s string) bool {
	trimmed := strings.TrimRight(s, "\r\n")
	if trimmed == "" {
		return false
	}
	delim := trimmed[0]
	if delim != '-' && delim != '*' && delim != '_' {
		return false
	}

	count := 0
	for i := 0; i < len(trimmed); i++ {
		switch trimmed[i] {
		case delim:
			count++
		case ' ', '\t':
		default:
			return false
		}
	}
	return count >= 3
}

// MarshalMarkdownWithTargetFallback renders markdown from the source template so new
// sections are included, while preserving existing target translations for entries
// not updated in the current run.
func MarshalMarkdownWithTargetFallback(sourceTemplate, targetTemplate []byte, values map[string]string) []byte {
	content, _ := MarshalMarkdownWithTargetFallbackDiagnostics(sourceTemplate, targetTemplate, values)
	return content
}

func MarshalMarkdownWithTargetFallbackDiagnostics(sourceTemplate, targetTemplate []byte, values map[string]string) ([]byte, MarkdownRenderDiagnostics) {
	sourceDoc, sourceEntries := parseMarkdownDocument(sourceTemplate)
	targetDoc, _ := parseMarkdownDocument(targetTemplate)
	targetContexts := targetDoc.keyContexts()
	targetPartUsed := make([]bool, len(targetDoc.parts))
	targetCtxCursor := 0
	targetPartCursor := 0
	sourceContexts := sourceDoc.keyContexts()
	sourceCtxIdx := 0
	var diags MarkdownRenderDiagnostics

	takeFallback := func(sourceCtx markdownKeyContext) (string, bool) {
		if idx, ok := selectMarkdownContextCandidate(targetContexts, targetPartUsed, sourceCtx, targetCtxCursor, sourceCtxIdx, len(sourceContexts)); ok {
			targetPartUsed[targetContexts[idx].partIndex] = true
			if idx >= targetCtxCursor {
				targetCtxCursor = idx + 1
			}
			if targetContexts[idx].partIndex+1 > targetPartCursor {
				targetPartCursor = targetContexts[idx].partIndex + 1
			}
			return targetContexts[idx].text, true
		}
		for _, startAt := range []int{targetPartCursor, 0} {
			if fallback, nextPartCursor, ok := takeMarkdownFallbackSpan(targetDoc, targetPartUsed, startAt, sourceCtx); ok {
				targetPartCursor = nextPartCursor
				return fallback, true
			}
		}
		for i := targetCtxCursor; i < len(targetContexts); i++ {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			targetPartUsed[targetContexts[i].partIndex] = true
			targetCtxCursor = i + 1
			if targetContexts[i].partIndex+1 > targetPartCursor {
				targetPartCursor = targetContexts[i].partIndex + 1
			}
			return targetContexts[i].text, true
		}
		for i := 0; i < len(targetContexts); i++ {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			targetPartUsed[targetContexts[i].partIndex] = true
			if i >= targetCtxCursor {
				targetCtxCursor = i + 1
			}
			if targetContexts[i].partIndex+1 > targetPartCursor {
				targetPartCursor = targetContexts[i].partIndex + 1
			}
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
			b.WriteString(renderMarkdownPartWithDiagnostics(part, v, &diags))
			sourceCtxIdx++
			continue
		}

		// Only consume fallback translations for keys that are part of source extraction.
		// This avoids injecting fallback text into non-translatable structural segments.
		if _, ok := sourceEntries[part.key]; ok && sourceCtxIdx < len(sourceContexts) {
			if fallback, ok := takeFallback(sourceContexts[sourceCtxIdx]); ok {
				b.WriteString(renderMarkdownPartWithDiagnostics(part, fallback, &diags))
				sourceCtxIdx++
				continue
			}
		}
		if sourceCtxIdx < len(sourceContexts) {
			sourceCtxIdx++
		}
		b.WriteString(renderMarkdownPartWithDiagnostics(part, part.source, &diags))
	}

	return []byte(b.String()), diags
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
	targetCtxCursor := 0
	targetPartCursor := 0
	sourceContexts := sourceDoc.keyContexts()
	sourceCtxIdx := 0
	aligned := make(map[string]string, len(sourceEntries))

	takeFallback := func(sourceCtx markdownKeyContext) (string, bool) {
		if idx, ok := selectMarkdownContextCandidate(targetContexts, targetPartUsed, sourceCtx, targetCtxCursor, sourceCtxIdx, len(sourceContexts)); ok {
			targetPartUsed[targetContexts[idx].partIndex] = true
			if idx >= targetCtxCursor {
				targetCtxCursor = idx + 1
			}
			if targetContexts[idx].partIndex+1 > targetPartCursor {
				targetPartCursor = targetContexts[idx].partIndex + 1
			}
			return targetContexts[idx].text, true
		}
		for _, startAt := range []int{targetPartCursor, 0} {
			if fallback, nextPartCursor, ok := takeMarkdownFallbackSpan(targetDoc, targetPartUsed, startAt, sourceCtx); ok {
				targetPartCursor = nextPartCursor
				return fallback, true
			}
		}
		for i := targetCtxCursor; i < len(targetContexts); i++ {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			targetPartUsed[targetContexts[i].partIndex] = true
			targetCtxCursor = i + 1
			if targetContexts[i].partIndex+1 > targetPartCursor {
				targetPartCursor = targetContexts[i].partIndex + 1
			}
			return targetContexts[i].text, true
		}
		for i := range targetContexts {
			if targetPartUsed[targetContexts[i].partIndex] {
				continue
			}
			targetPartUsed[targetContexts[i].partIndex] = true
			if i >= targetCtxCursor {
				targetCtxCursor = i + 1
			}
			if targetContexts[i].partIndex+1 > targetPartCursor {
				targetPartCursor = targetContexts[i].partIndex + 1
			}
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

func selectMarkdownContextCandidate(targetContexts []markdownKeyContext, targetPartUsed []bool, sourceCtx markdownKeyContext, targetCtxCursor, sourceCtxIdx, sourceTotal int) (int, bool) {
	best := -1
	bestScore := math.MaxFloat64
	for i := range targetContexts {
		if targetPartUsed[targetContexts[i].partIndex] {
			continue
		}
		if targetContexts[i].prevLiteral != sourceCtx.prevLiteral || targetContexts[i].nextLiteral != sourceCtx.nextLiteral {
			continue
		}
		score := markdownRelativeIndexDistance(i, len(targetContexts), sourceCtxIdx, sourceTotal)
		if i < targetCtxCursor {
			score += 0.25
		}
		if score < bestScore {
			best = i
			bestScore = score
		}
	}
	if best < 0 {
		return 0, false
	}
	return best, true
}

func markdownRelativeIndexDistance(targetIdx, targetTotal, sourceIdx, sourceTotal int) float64 {
	if targetTotal <= 1 || sourceTotal <= 1 {
		return 0
	}
	targetPos := float64(targetIdx) / float64(targetTotal-1)
	sourcePos := float64(sourceIdx) / float64(sourceTotal-1)
	return math.Abs(targetPos - sourcePos)
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
		b.WriteString(renderMarkdownPart(targetDoc.parts[i], targetDoc.parts[i].source))
	}

	nextCursor := startAt
	if spanEnd > nextCursor {
		nextCursor = spanEnd
	}
	return b.String(), nextCursor, true
}
