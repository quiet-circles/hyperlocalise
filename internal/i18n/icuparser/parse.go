package icuparser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func Parse(input string, opts *ParseOptions) ([]Element, error) {
	if opts == nil {
		opts = &ParseOptions{}
	}
	p := astParser{
		src:  input,
		opts: *opts,
	}
	elems, err := p.parseMessage(parseCtx{}, false)
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.src) {
		return nil, fmt.Errorf("unexpected trailing content at %d", p.pos)
	}
	return elems, nil
}

type parseCtx struct {
	inPlural bool
}

type astParser struct {
	src  string
	pos  int
	opts ParseOptions
}

func (p *astParser) parseMessage(ctx parseCtx, untilBrace bool) ([]Element, error) {
	var out []Element
	var text strings.Builder

	flushText := func() {
		if text.Len() == 0 {
			return
		}
		out = append(out, LiteralElement{Value: text.String()})
		text.Reset()
	}

	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case '{':
			flushText()
			el, err := p.parseArgumentLike()
			if err != nil {
				return nil, err
			}
			out = append(out, el)
		case '}':
			if !untilBrace {
				return nil, fmt.Errorf("unexpected closing brace at %d", p.pos)
			}
			flushText()
			return out, nil
		case '#':
			if ctx.inPlural {
				flushText()
				out = append(out, PoundElement{})
				p.pos++
				continue
			}
			text.WriteByte(p.src[p.pos])
			p.pos++
		case '<':
			if p.opts.IgnoreTag {
				text.WriteByte(p.src[p.pos])
				p.pos++
				continue
			}
			tag, ok, err := p.tryParseTag(ctx)
			if err != nil {
				return nil, err
			}
			if ok {
				flushText()
				out = append(out, tag)
				continue
			}
			text.WriteByte(p.src[p.pos])
			p.pos++
		case '\'':
			// Simplified ICU apostrophe handling; keeps literal content parse-safe.
			p.consumeQuotedInto(&text)
		default:
			text.WriteByte(p.src[p.pos])
			p.pos++
		}
	}

	flushText()
	if untilBrace {
		return nil, fmt.Errorf("unclosed brace at %d", p.pos)
	}
	return out, nil
}

func (p *astParser) parseArgumentLike() (Element, error) {
	if !p.consume('{') {
		return nil, fmt.Errorf("expected '{' at %d", p.pos)
	}
	p.skipSpaces()
	arg, ok := p.readIdentifierLike()
	if !ok {
		return nil, fmt.Errorf("expected argument name at %d", p.pos)
	}
	p.skipSpaces()
	if p.consume('}') {
		return ArgumentElement{Value: arg}, nil
	}
	if !p.consume(',') {
		return nil, fmt.Errorf("expected ',' or '}' at %d", p.pos)
	}
	p.skipSpaces()
	kind, ok := p.readIdentifierLike()
	if !ok {
		return nil, fmt.Errorf("expected format type at %d", p.pos)
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	p.skipSpaces()

	switch kind {
	case "number", "date", "time":
		style, err := p.parseSimpleStyle()
		if err != nil {
			return nil, err
		}
		switch kind {
		case "number":
			return NumberElement{Value: arg, Style: style}, nil
		case "date":
			return DateElement{Value: arg, Style: style}, nil
		default:
			return TimeElement{Value: arg, Style: style}, nil
		}
	case "select":
		if !p.consume(',') {
			return nil, fmt.Errorf("expected ',' before select options at %d", p.pos)
		}
		p.skipSpaces()
		opts, err := p.parseSelectOptions(parseCtx{})
		if err != nil {
			return nil, err
		}
		if !p.consume('}') {
			return nil, fmt.Errorf("expected closing brace for select at %d", p.pos)
		}
		return SelectElement{Value: arg, Options: opts}, nil
	case "plural", "selectordinal":
		if !p.consume(',') {
			return nil, fmt.Errorf("expected ',' before plural options at %d", p.pos)
		}
		p.skipSpaces()
		offset, opts, err := p.parsePluralOptions()
		if err != nil {
			return nil, err
		}
		if !p.consume('}') {
			return nil, fmt.Errorf("expected closing brace for plural at %d", p.pos)
		}
		return PluralElement{
			Value:      arg,
			Options:    opts,
			Offset:     offset,
			Ordinal:    kind == "selectordinal",
			PluralType: map[bool]ElementType{true: TypeSelectOrdinal, false: TypePlural}[kind == "selectordinal"],
		}, nil
	default:
		// Generic formatter/custom format. Preserve style text but don't parse skeletons yet.
		style, err := p.parseSimpleStyle()
		if err != nil {
			return nil, err
		}
		_ = style
		return ArgumentElement{Value: arg}, nil
	}
}

func (p *astParser) parseSimpleStyle() (string, error) {
	if p.consume('}') {
		return "", nil
	}
	if !p.consume(',') {
		return "", fmt.Errorf("expected ',' or '}' at %d", p.pos)
	}
	start := p.pos
	depth := 0
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case '{':
			depth++
			p.pos++
		case '}':
			if depth == 0 {
				style := strings.TrimSpace(p.src[start:p.pos])
				p.pos++ // consume closing brace
				return style, nil
			}
			depth--
			p.pos++
		case '\'':
			p.skipQuotedLiteral()
		default:
			p.pos++
		}
	}
	return "", fmt.Errorf("unclosed simple formatter style")
}

func (p *astParser) parseSelectOptions(ctx parseCtx) ([]SelectOption, error) {
	var out []SelectOption
	for {
		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unclosed brace at %d", p.pos)
		}
		p.skipSpaces()
		if p.peek() == '}' {
			if len(out) == 0 {
				return nil, fmt.Errorf("select argument missing options at %d", p.pos)
			}
			return out, nil
		}
		sel, ok := p.readSelector()
		if !ok {
			return nil, fmt.Errorf("expected select selector at %d", p.pos)
		}
		p.skipSpaces()
		if !p.consume('{') {
			return nil, fmt.Errorf("expected select option body at %d", p.pos)
		}
		body, err := p.parseMessage(ctx, true)
		if err != nil {
			return nil, err
		}
		if !p.consume('}') {
			return nil, fmt.Errorf("expected closing brace for select option at %d", p.pos)
		}
		out = append(out, SelectOption{Selector: sel, Value: body})
	}
}

func (p *astParser) parsePluralOptions() (int, []PluralOption, error) {
	offset := 0
	var out []PluralOption
	for {
		if p.pos >= len(p.src) {
			return 0, nil, fmt.Errorf("unclosed brace at %d", p.pos)
		}
		p.skipSpaces()
		if p.peek() == '}' {
			if len(out) == 0 {
				return 0, nil, fmt.Errorf("ICU argument missing options at %d", p.pos)
			}
			return offset, out, nil
		}
		sel, ok := p.readSelector()
		if !ok {
			return 0, nil, fmt.Errorf("expected ICU selector at %d", p.pos)
		}
		p.skipSpaces()
		selLower := strings.ToLower(sel)
		if strings.HasPrefix(selLower, "offset:") {
			n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(selLower, "offset:")))
			if err != nil {
				return 0, nil, fmt.Errorf("invalid plural offset %q", sel)
			}
			offset = n
			continue
		}
		if !p.consume('{') {
			return 0, nil, fmt.Errorf("expected ICU option body at %d", p.pos)
		}
		body, err := p.parseMessage(parseCtx{inPlural: true}, true)
		if err != nil {
			return 0, nil, err
		}
		if !p.consume('}') {
			return 0, nil, fmt.Errorf("expected closing brace for ICU option at %d", p.pos)
		}
		out = append(out, PluralOption{Selector: sel, Value: body})
	}
}

func (p *astParser) tryParseTag(ctx parseCtx) (TagElement, bool, error) {
	start := p.pos
	if !p.consume('<') {
		return TagElement{}, false, nil
	}
	if p.peek() == '/' || p.peek() == '!' || p.peek() == '?' {
		p.pos = start
		return TagElement{}, false, nil
	}
	name, ok := p.readTagName()
	if !ok {
		p.pos = start
		return TagElement{}, false, nil
	}
	p.skipSpaces()
	if p.consume('/') {
		if !p.consume('>') {
			return TagElement{}, false, fmt.Errorf("expected '/>' for self-closing tag at %d", p.pos)
		}
		return TagElement{Value: name, SelfClosing: true}, true, nil
	}
	if !p.consume('>') {
		p.pos = start
		return TagElement{}, false, nil
	}
	children, err := p.parseUntilClosingTag(name, ctx)
	if err != nil {
		return TagElement{}, false, err
	}
	return TagElement{Value: name, Children: children}, true, nil
}

func (p *astParser) parseUntilClosingTag(name string, ctx parseCtx) ([]Element, error) {
	var out []Element
	var text strings.Builder
	flushText := func() {
		if text.Len() == 0 {
			return
		}
		out = append(out, LiteralElement{Value: text.String()})
		text.Reset()
	}

	for p.pos < len(p.src) {
		if strings.HasPrefix(p.src[p.pos:], "</") {
			save := p.pos
			p.pos += 2
			closeName, ok := p.readTagName()
			if !ok {
				p.pos = save
				text.WriteByte(p.src[p.pos])
				p.pos++
				continue
			}
			p.skipSpaces()
			if !p.consume('>') {
				return nil, fmt.Errorf("expected closing '>' for tag %q", closeName)
			}
			if closeName != name {
				return nil, fmt.Errorf("mismatched closing tag: got %q want %q", closeName, name)
			}
			flushText()
			return out, nil
		}

		switch p.peek() {
		case '{':
			flushText()
			el, err := p.parseArgumentLike()
			if err != nil {
				return nil, err
			}
			out = append(out, el)
		case '#':
			if ctx.inPlural {
				flushText()
				out = append(out, PoundElement{})
				p.pos++
				continue
			}
			text.WriteByte('#')
			p.pos++
		case '<':
			tag, ok, err := p.tryParseTag(ctx)
			if err != nil {
				return nil, err
			}
			if ok {
				flushText()
				out = append(out, tag)
				continue
			}
			text.WriteByte('<')
			p.pos++
		case '\'':
			p.consumeQuotedInto(&text)
		default:
			text.WriteByte(p.src[p.pos])
			p.pos++
		}
	}
	return nil, fmt.Errorf("unclosed tag %q", name)
}

func (p *astParser) consumeQuotedInto(b *strings.Builder) {
	p.pos++ // opening '
	if p.pos < len(p.src) && p.src[p.pos] == '\'' {
		b.WriteByte('\'')
		p.pos++
		return
	}
	for p.pos < len(p.src) {
		if p.src[p.pos] != '\'' {
			b.WriteByte(p.src[p.pos])
			p.pos++
			continue
		}
		if p.pos+1 < len(p.src) && p.src[p.pos+1] == '\'' {
			b.WriteByte('\'')
			p.pos += 2
			continue
		}
		p.pos++
		return
	}
}

func (p *astParser) skipSpaces() {
	for p.pos < len(p.src) {
		r, w := utf8.DecodeRuneInString(p.src[p.pos:])
		if !unicode.IsSpace(r) {
			break
		}
		p.pos += w
	}
}

func (p *astParser) consume(ch byte) bool {
	if p.pos < len(p.src) && p.src[p.pos] == ch {
		p.pos++
		return true
	}
	return false
}

func (p *astParser) peek() byte {
	if p.pos >= len(p.src) {
		return 0
	}
	return p.src[p.pos]
}

func (p *astParser) readIdentifierLike() (string, bool) {
	start := p.pos
	for p.pos < len(p.src) {
		r, w := utf8.DecodeRuneInString(p.src[p.pos:])
		if unicode.IsSpace(r) || r == ',' || r == '{' || r == '}' {
			break
		}
		p.pos += w
	}
	if p.pos == start {
		return "", false
	}
	return strings.TrimSpace(p.src[start:p.pos]), true
}

func (p *astParser) readSelector() (string, bool) {
	start := p.pos
	if p.pos < len(p.src) && p.src[p.pos] == '=' {
		p.pos++
		for p.pos < len(p.src) && isASCIIDigit(p.src[p.pos]) {
			p.pos++
		}
		return strings.TrimSpace(p.src[start:p.pos]), p.pos > start+1
	}
	for p.pos < len(p.src) {
		r, w := utf8.DecodeRuneInString(p.src[p.pos:])
		if unicode.IsSpace(r) || r == '{' || r == '}' || r == ',' {
			break
		}
		p.pos += w
	}
	if p.pos == start {
		return "", false
	}
	return strings.TrimSpace(p.src[start:p.pos]), true
}

func (p *astParser) readTagName() (string, bool) {
	start := p.pos
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' || ch == '.' {
			p.pos++
			continue
		}
		break
	}
	if p.pos == start {
		return "", false
	}
	return p.src[start:p.pos], true
}

func (p *astParser) skipQuotedLiteral() {
	p.pos++
	for p.pos < len(p.src) {
		if p.src[p.pos] != '\'' {
			p.pos++
			continue
		}
		if p.pos+1 < len(p.src) && p.src[p.pos+1] == '\'' {
			p.pos += 2
			continue
		}
		p.pos++
		return
	}
}
