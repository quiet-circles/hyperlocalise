package translationfileparser

import (
	"bytes"
	"encoding/xml"
	"fmt"
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
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("xml decode: %w", err)
		}

		switch token := tok.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "trans-unit", "unit":
				if current != nil {
					finalizeXLIFFUnit(out, *current)
				}
				current = &xliffUnit{key: attrValue(token.Attr, "id")}
				if current.key == "" {
					current.key = attrValue(token.Attr, "name")
				}
				if current.key == "" {
					current.key = attrValue(token.Attr, "resname")
				}
			case "source":
				if current != nil {
					captureSource = true
				}
			case "target":
				if current != nil {
					captureTarget = true
				}
			}
		case xml.EndElement:
			switch token.Name.Local {
			case "source":
				captureSource = false
			case "target":
				captureTarget = false
			case "trans-unit", "unit":
				if current != nil {
					finalizeXLIFFUnit(out, *current)
					current = nil
				}
			}
		case xml.CharData:
			if current == nil {
				continue
			}
			if captureTarget {
				current.target.Write(token)
			} else if captureSource {
				current.source.Write(token)
			}
		}
	}

	if current != nil {
		finalizeXLIFFUnit(out, *current)
	}

	return out, nil
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
