package translationfileparser

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// XLIFFParser parses XLIFF 1.2 and 2.x translation files.
type XLIFFParser struct{}

func (p XLIFFParser) Parse(content []byte) (map[string]string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(content))

	out := make(map[string]string)
	var current *xliffUnit
	var contentState *xliffContentState

	for {
		tok, err := decoder.Token()
		if err != nil {
			if isEOFError(err) {
				break
			}
			return nil, fmt.Errorf("xml decode: %w", err)
		}

		if err := consumeXLIFFToken(tok, out, &current, &contentState); err != nil {
			return nil, err
		}
	}

	if current != nil {
		finalizeXLIFFUnit(out, *current)
	}

	return out, nil
}

func isEOFError(err error) bool {
	return errors.Is(err, io.EOF)
}

func consumeXLIFFToken(tok xml.Token, out map[string]string, current **xliffUnit, contentState **xliffContentState) error {
	if *contentState != nil {
		finished, err := (*contentState).consume(tok)
		if err != nil {
			return err
		}
		if finished {
			switch (*contentState).name {
			case "source":
				(*current).source.WriteString((*contentState).buffer.String())
			case "target":
				(*current).target.WriteString((*contentState).buffer.String())
			}
			*contentState = nil
		}
		return nil
	}

	switch token := tok.(type) {
	case xml.StartElement:
		handleXLIFFStart(token, out, current, contentState)
	case xml.EndElement:
		handleXLIFFEnd(token, out, current)
	}
	return nil
}

func handleXLIFFStart(token xml.StartElement, out map[string]string, current **xliffUnit, contentState **xliffContentState) {
	switch token.Name.Local {
	case "trans-unit", "unit":
		if *current != nil {
			finalizeXLIFFUnit(out, **current)
		}
		*current = &xliffUnit{key: resolveXLIFFUnitKey(token.Attr)}
	case "source":
		if *current != nil {
			*contentState = newXLIFFContentState("source")
		}
	case "target":
		if *current != nil {
			*contentState = newXLIFFContentState("target")
		}
	}
}

func handleXLIFFEnd(token xml.EndElement, out map[string]string, current **xliffUnit) {
	switch token.Name.Local {
	case "trans-unit", "unit":
		if *current != nil {
			finalizeXLIFFUnit(out, **current)
			*current = nil
		}
	}
}

func resolveXLIFFUnitKey(attrs []xml.Attr) string {
	for _, name := range []string{"id", "name", "resname"} {
		if value := attrValue(attrs, name); value != "" {
			return value
		}
	}
	return ""
}

type xliffUnit struct {
	key    string
	source strings.Builder
	target strings.Builder
}

type xliffContentState struct {
	name    string
	depth   int
	buffer  bytes.Buffer
	encoder *xml.Encoder
}

func newXLIFFContentState(name string) *xliffContentState {
	state := &xliffContentState{name: name}
	state.encoder = xml.NewEncoder(&state.buffer)
	return state
}

func (s *xliffContentState) consume(tok xml.Token) (bool, error) {
	switch token := tok.(type) {
	case xml.StartElement:
		s.depth++
		if err := s.encoder.EncodeToken(token); err != nil {
			return false, fmt.Errorf("xml encode token: %w", err)
		}
	case xml.EndElement:
		if token.Name.Local == s.name && s.depth == 0 {
			if err := s.encoder.Flush(); err != nil {
				return false, fmt.Errorf("xml encode flush: %w", err)
			}
			return true, nil
		}
		if err := s.encoder.EncodeToken(token); err != nil {
			return false, fmt.Errorf("xml encode token: %w", err)
		}
		if s.depth > 0 {
			s.depth--
		}
	case xml.CharData, xml.Comment, xml.Directive, xml.ProcInst:
		if err := s.encoder.EncodeToken(token); err != nil {
			return false, fmt.Errorf("xml encode token: %w", err)
		}
	}
	return false, nil
}

func finalizeXLIFFUnit(out map[string]string, unit xliffUnit) {
	key := strings.TrimSpace(unit.key)
	if key == "" {
		return
	}

	value := unit.target.String()
	if strings.TrimSpace(value) == "" {
		value = unit.source.String()
	}
	if value == "" {
		return
	}

	out[key] = value
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

// MarshalXLIFF rewrites XLIFF source/target text using values keyed by unit id/name/resname.
// If a unit has <target>, only target text is updated; otherwise source text is updated.
func MarshalXLIFF(template []byte, values map[string]string, targetLocale string) ([]byte, error) {
	// TODO: XLIFF 2.x units may contain multiple segments. We currently rewrite at
	// the unit-level by replacing the active source/target element content in stream
	// order, rather than aligning translations per segment.
	unitHasTarget, err := collectXLIFFUnitTargets(template)
	if err != nil {
		return nil, err
	}

	decoder := xml.NewDecoder(bytes.NewReader(template))
	var out bytes.Buffer
	encoder := xml.NewEncoder(&out)

	currentUnitKey := ""

	type textElementState struct {
		name       string
		replace    bool
		hasValue   bool
		wroteValue bool
		depth      int
	}
	var textState *textElementState

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("xml decode: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			t = rewriteXLIFFLocaleAttrs(t, targetLocale)
			if textState != nil {
				textState.depth++
				if textState.replace && textState.hasValue {
					continue
				}
				if err := encoder.EncodeToken(t); err != nil {
					return nil, fmt.Errorf("xml encode start: %w", err)
				}
				continue
			}
			switch t.Name.Local {
			case "trans-unit", "unit":
				currentUnitKey = resolveXLIFFUnitKey(t.Attr)
				textState = nil
			case "target":
				if currentUnitKey != "" {
					v, ok := values[currentUnitKey]
					textState = &textElementState{name: "target", replace: true, hasValue: ok, wroteValue: false}
					if ok && strings.TrimSpace(v) == "" {
						textState.wroteValue = true
					}
				}
			case "source":
				if currentUnitKey != "" {
					v, ok := values[currentUnitKey]
					textState = &textElementState{name: "source", replace: !unitHasTarget[currentUnitKey], hasValue: ok, wroteValue: false}
					if ok && strings.TrimSpace(v) == "" {
						textState.wroteValue = true
					}
				}
			}
			if err := encoder.EncodeToken(t); err != nil {
				return nil, fmt.Errorf("xml encode start: %w", err)
			}
		case xml.EndElement:
			if textState != nil {
				if t.Name.Local == textState.name && textState.depth == 0 {
					if textState.replace && textState.hasValue && !textState.wroteValue {
						if err := encodeXLIFFFragment(encoder, values[currentUnitKey]); err != nil {
							return nil, err
						}
					}
					textState = nil
				} else {
					if textState.replace && textState.hasValue {
						if textState.depth > 0 {
							textState.depth--
						}
						continue
					}
					if err := encoder.EncodeToken(t); err != nil {
						return nil, fmt.Errorf("xml encode end: %w", err)
					}
					if textState.depth > 0 {
						textState.depth--
					}
					continue
				}
			}

			if t.Name.Local == "trans-unit" || t.Name.Local == "unit" {
				currentUnitKey = ""
				textState = nil
			}

			if err := encoder.EncodeToken(t); err != nil {
				return nil, fmt.Errorf("xml encode end: %w", err)
			}
		case xml.CharData:
			if textState != nil && textState.replace && textState.hasValue {
				if !textState.wroteValue {
					if err := encodeXLIFFFragment(encoder, values[currentUnitKey]); err != nil {
						return nil, err
					}
					textState.wroteValue = true
				}
				continue
			}

			if err := encoder.EncodeToken(t); err != nil {
				return nil, fmt.Errorf("xml encode char data: %w", err)
			}
		case xml.Comment, xml.Directive, xml.ProcInst:
			if textState != nil && textState.replace && textState.hasValue {
				continue
			}
			if err := encoder.EncodeToken(t); err != nil {
				return nil, fmt.Errorf("xml encode token: %w", err)
			}
		default:
			if err := encoder.EncodeToken(t); err != nil {
				return nil, fmt.Errorf("xml encode token: %w", err)
			}
		}
	}

	if err := encoder.Flush(); err != nil {
		return nil, fmt.Errorf("xml encode flush: %w", err)
	}
	return out.Bytes(), nil
}

func encodeXLIFFFragment(encoder *xml.Encoder, value string) error {
	wrapped := "<hyperlocalise-root>" + value + "</hyperlocalise-root>"
	decoder := xml.NewDecoder(strings.NewReader(wrapped))
	depth := 0
	var tokens []xml.Token
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				for _, token := range tokens {
					if err := encoder.EncodeToken(token); err != nil {
						return fmt.Errorf("xml encode token: %w", err)
					}
				}
				return nil
			}
			if err := encoder.EncodeToken(xml.CharData([]byte(value))); err != nil {
				return fmt.Errorf("xml encode char data: %w", err)
			}
			return nil
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if depth == 0 && t.Name.Local == "hyperlocalise-root" {
				depth++
				continue
			}
			depth++
		case xml.EndElement:
			depth--
			if depth == 0 && t.Name.Local == "hyperlocalise-root" {
				continue
			}
		}

		if depth > 0 {
			tokens = append(tokens, cloneXMLToken(tok))
		}
	}
}

func collectXLIFFUnitTargets(template []byte) (map[string]bool, error) {
	decoder := xml.NewDecoder(bytes.NewReader(template))
	unitHasTarget := make(map[string]bool)
	currentUnitKey := ""

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return unitHasTarget, nil
			}
			return nil, fmt.Errorf("xml decode: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "trans-unit", "unit":
				currentUnitKey = resolveXLIFFUnitKey(t.Attr)
			case "target":
				if currentUnitKey != "" {
					unitHasTarget[currentUnitKey] = true
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "trans-unit", "unit":
				currentUnitKey = ""
			}
		}
	}
}

func rewriteXLIFFLocaleAttrs(start xml.StartElement, locale string) xml.StartElement {
	loc := strings.TrimSpace(locale)
	if loc == "" {
		return start
	}

	switch start.Name.Local {
	case "file":
		if attrValue(start.Attr, "source-language") != "" {
			start.Attr = upsertXLIFFAttr(start.Attr, "target-language", loc)
		}
	case "xliff":
		if isXLIFF20Root(start.Attr) {
			start.Attr = upsertXLIFFAttr(start.Attr, "trgLang", loc)
		}
	}
	return start
}

func isXLIFF20Root(attrs []xml.Attr) bool {
	version := attrValue(attrs, "version")
	return strings.HasPrefix(version, "2")
}

func upsertXLIFFAttr(attrs []xml.Attr, name, value string) []xml.Attr {
	for i := range attrs {
		if attrs[i].Name.Local == name {
			attrs[i].Value = value
			return attrs
		}
	}
	return append(attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

func cloneXMLToken(tok xml.Token) xml.Token {
	switch t := tok.(type) {
	case xml.StartElement:
		attrs := make([]xml.Attr, len(t.Attr))
		copy(attrs, t.Attr)
		t.Attr = attrs
		return t
	case xml.EndElement:
		return t
	case xml.CharData:
		cloned := make(xml.CharData, len(t))
		copy(cloned, t)
		return cloned
	case xml.Comment:
		cloned := make(xml.Comment, len(t))
		copy(cloned, t)
		return cloned
	case xml.Directive:
		cloned := make(xml.Directive, len(t))
		copy(cloned, t)
		return cloned
	case xml.ProcInst:
		cloned := xml.ProcInst{Target: t.Target}
		cloned.Inst = make([]byte, len(t.Inst))
		copy(cloned.Inst, t.Inst)
		return cloned
	default:
		return tok
	}
}
