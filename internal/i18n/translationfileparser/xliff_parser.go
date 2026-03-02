package translationfileparser

import (
	"bytes"
	"encoding/xml"
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
	var captureSource bool
	var captureTarget bool

	for {
		tok, err := decoder.Token()
		if err != nil {
			if isEOFError(err) {
				break
			}
			return nil, fmt.Errorf("xml decode: %w", err)
		}

		if err := consumeXLIFFToken(tok, out, &current, &captureSource, &captureTarget); err != nil {
			return nil, err
		}
	}

	if current != nil {
		finalizeXLIFFUnit(out, *current)
	}

	return out, nil
}

func isEOFError(err error) bool {
	return err != nil && err.Error() == "EOF"
}

func consumeXLIFFToken(tok xml.Token, out map[string]string, current **xliffUnit, captureSource, captureTarget *bool) error {
	switch token := tok.(type) {
	case xml.StartElement:
		handleXLIFFStart(token, out, current, captureSource, captureTarget)
	case xml.EndElement:
		handleXLIFFEnd(token, out, current, captureSource, captureTarget)
	case xml.CharData:
		appendXLIFFText(token, *current, *captureSource, *captureTarget)
	}
	return nil
}

func handleXLIFFStart(token xml.StartElement, out map[string]string, current **xliffUnit, captureSource, captureTarget *bool) {
	switch token.Name.Local {
	case "trans-unit", "unit":
		if *current != nil {
			finalizeXLIFFUnit(out, **current)
		}
		*current = &xliffUnit{key: resolveXLIFFUnitKey(token.Attr)}
	case "source":
		if *current != nil {
			*captureSource = true
		}
	case "target":
		if *current != nil {
			*captureTarget = true
		}
	}
}

func handleXLIFFEnd(token xml.EndElement, out map[string]string, current **xliffUnit, captureSource, captureTarget *bool) {
	switch token.Name.Local {
	case "source":
		*captureSource = false
	case "target":
		*captureTarget = false
	case "trans-unit", "unit":
		if *current != nil {
			finalizeXLIFFUnit(out, **current)
			*current = nil
		}
	}
}

func appendXLIFFText(token xml.CharData, current *xliffUnit, captureSource, captureTarget bool) {
	if current == nil {
		return
	}
	if captureTarget {
		current.target.Write(token)
		return
	}
	if captureSource {
		current.source.Write(token)
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

func finalizeXLIFFUnit(out map[string]string, unit xliffUnit) {
	key := strings.TrimSpace(unit.key)
	if key == "" {
		return
	}

	value := strings.TrimSpace(unit.target.String())
	if value == "" {
		value = strings.TrimSpace(unit.source.String())
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
	decoder := xml.NewDecoder(bytes.NewReader(template))
	var out bytes.Buffer
	encoder := xml.NewEncoder(&out)

	currentUnitKey := ""
	currentUnitHasTarget := false

	type textElementState struct {
		name       string
		replace    bool
		hasValue   bool
		wroteValue bool
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
			switch t.Name.Local {
			case "trans-unit", "unit":
				currentUnitKey = resolveXLIFFUnitKey(t.Attr)
				currentUnitHasTarget = false
				textState = nil
			case "target":
				if currentUnitKey != "" {
					currentUnitHasTarget = true
					v, ok := values[currentUnitKey]
					textState = &textElementState{name: "target", replace: true, hasValue: ok, wroteValue: false}
					if ok && strings.TrimSpace(v) == "" {
						textState.wroteValue = true
					}
				}
			case "source":
				if currentUnitKey != "" {
					v, ok := values[currentUnitKey]
					textState = &textElementState{name: "source", replace: !currentUnitHasTarget, hasValue: ok, wroteValue: false}
					if ok && strings.TrimSpace(v) == "" {
						textState.wroteValue = true
					}
				}
			}
			if err := encoder.EncodeToken(t); err != nil {
				return nil, fmt.Errorf("xml encode start: %w", err)
			}
		case xml.EndElement:
			if textState != nil && t.Name.Local == textState.name {
				if textState.replace && textState.hasValue && !textState.wroteValue {
					if err := encoder.EncodeToken(xml.CharData([]byte(values[currentUnitKey]))); err != nil {
						return nil, fmt.Errorf("xml encode char data: %w", err)
					}
				}
				textState = nil
			}

			if t.Name.Local == "trans-unit" || t.Name.Local == "unit" {
				currentUnitKey = ""
				currentUnitHasTarget = false
				textState = nil
			}

			if err := encoder.EncodeToken(t); err != nil {
				return nil, fmt.Errorf("xml encode end: %w", err)
			}
		case xml.CharData:
			if textState != nil && textState.replace && textState.hasValue {
				if !textState.wroteValue {
					if err := encoder.EncodeToken(xml.CharData([]byte(values[currentUnitKey]))); err != nil {
						return nil, fmt.Errorf("xml encode char data: %w", err)
					}
					textState.wroteValue = true
				}
				continue
			}

			if err := encoder.EncodeToken(t); err != nil {
				return nil, fmt.Errorf("xml encode char data: %w", err)
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

func rewriteXLIFFLocaleAttrs(start xml.StartElement, locale string) xml.StartElement {
	loc := strings.TrimSpace(locale)
	if loc == "" {
		return start
	}

	switch start.Name.Local {
	case "file":
		for i := range start.Attr {
			switch start.Attr[i].Name.Local {
			case "source-language", "target-language":
				start.Attr[i].Value = loc
			}
		}
	case "xliff":
		for i := range start.Attr {
			switch start.Attr[i].Name.Local {
			case "srcLang", "trgLang":
				start.Attr[i].Value = loc
			}
		}
	}
	return start
}
