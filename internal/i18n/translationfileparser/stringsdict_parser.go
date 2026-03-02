package translationfileparser

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
)

// AppleStringsdictParser parses Apple .stringsdict pluralization files.
type AppleStringsdictParser struct{}

type stringsdictEntry struct {
	key         string
	sourceValue string
	valueRaw    string
	valueStart  int
	valueEnd    int
}

type stringsdictDocument struct {
	template string
	entries  []stringsdictEntry
}

func (p AppleStringsdictParser) Parse(content []byte) (map[string]string, error) {
	doc, err := parseStringsdictDocument(content)
	if err != nil {
		return nil, err
	}

	out := map[string]string{}
	for _, entry := range doc.entries {
		out[entry.key] = entry.sourceValue
	}
	return out, nil
}

func MarshalAppleStringsdict(template []byte, values map[string]string) ([]byte, error) {
	doc, err := parseStringsdictDocument(template)
	if err != nil {
		return nil, err
	}
	return doc.render(values), nil
}

func (d stringsdictDocument) render(values map[string]string) []byte {
	if len(d.entries) == 0 {
		return []byte(d.template)
	}

	entries := append([]stringsdictEntry(nil), d.entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].valueStart < entries[j].valueStart })

	var b strings.Builder
	cursor := 0
	for _, entry := range entries {
		if entry.valueStart < cursor || entry.valueStart > len(d.template) || entry.valueEnd > len(d.template) {
			continue
		}
		b.WriteString(d.template[cursor:entry.valueStart])
		if translated, ok := values[entry.key]; ok {
			if translated == entry.sourceValue {
				b.WriteString(entry.valueRaw)
			} else {
				b.WriteString(escapeXMLText(translated))
			}
		} else {
			b.WriteString(entry.valueRaw)
		}
		cursor = entry.valueEnd
	}
	b.WriteString(d.template[cursor:])
	return []byte(b.String())
}

func parseStringsdictDocument(content []byte) (stringsdictDocument, error) {
	text := string(content)
	doc := stringsdictDocument{template: text, entries: []stringsdictEntry{}}

	decoder := xml.NewDecoder(bytes.NewReader(content))
	type dictFrame struct {
		path       []string
		pendingKey string
	}
	dictStack := []dictFrame{}

	captureKey := false
	var keyBuilder strings.Builder

	inValueString := false
	valuePath := ""
	valueStart := -1
	valueEnd := -1
	var valueBuilder strings.Builder

	for {
		tok, err := decoder.Token()
		if err != nil {
			if isEOFError(err) {
				break
			}
			return stringsdictDocument{}, fmt.Errorf("xml decode: %w", err)
		}

		switch token := tok.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "key":
				captureKey = true
				keyBuilder.Reset()
			case "dict":
				frame := dictFrame{path: []string{}}
				if len(dictStack) > 0 {
					parent := &dictStack[len(dictStack)-1]
					frame.path = append(frame.path, parent.path...)
					if parent.pendingKey != "" {
						frame.path = append(frame.path, parent.pendingKey)
						parent.pendingKey = ""
					}
				}
				dictStack = append(dictStack, frame)
			case "string":
				if len(dictStack) == 0 {
					continue
				}
				frame := &dictStack[len(dictStack)-1]
				if frame.pendingKey == "" {
					continue
				}

				path := append([]string{}, frame.path...)
				path = append(path, frame.pendingKey)
				frame.pendingKey = ""
				valuePath = strings.Join(path, ".")
				inValueString = true
				valueBuilder.Reset()
				valueStart = -1
				valueEnd = -1
			}
		case xml.EndElement:
			switch token.Name.Local {
			case "key":
				captureKey = false
				if len(dictStack) > 0 {
					dictStack[len(dictStack)-1].pendingKey = strings.TrimSpace(keyBuilder.String())
				}
			case "dict":
				if len(dictStack) > 0 {
					dictStack = dictStack[:len(dictStack)-1]
				}
			case "string":
				if inValueString {
					if valueStart >= 0 && valueEnd >= valueStart {
						doc.entries = append(doc.entries, stringsdictEntry{
							key:         valuePath,
							sourceValue: valueBuilder.String(),
							valueRaw:    text[valueStart:valueEnd],
							valueStart:  valueStart,
							valueEnd:    valueEnd,
						})
					}
					inValueString = false
					valuePath = ""
				}
			}
		case xml.CharData:
			if captureKey {
				keyBuilder.Write(token)
			}
			if inValueString {
				valueBuilder.Write(token)
				tokenEnd := int(decoder.InputOffset())
				tokenStart := tokenEnd - len(token)
				if valueStart == -1 || tokenStart < valueStart {
					valueStart = tokenStart
				}
				if tokenEnd > valueEnd {
					valueEnd = tokenEnd
				}
			}
		}
	}

	return doc, nil
}

func escapeXMLText(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(s)
}
