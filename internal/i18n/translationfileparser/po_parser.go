package translationfileparser

import (
	"fmt"
	"strconv"
	"strings"
)

// POFileParser parses GNU gettext .po translation files.
type POFileParser struct{}

func (p POFileParser) Parse(content []byte) (map[string]string, error) {
	lines := strings.Split(string(content), "\n")
	out := map[string]string{}

	var currentMsgID strings.Builder
	var currentMsgStr strings.Builder
	activeField := ""
	seenMsgID := false
	seenMsgStr := false

	flush := func() {
		if !seenMsgID || !seenMsgStr {
			return
		}
		key := strings.TrimSpace(currentMsgID.String())
		if key == "" {
			return // skip header entry (msgid "")
		}
		out[key] = currentMsgStr.String()
	}

	reset := func() {
		currentMsgID.Reset()
		currentMsgStr.Reset()
		activeField = ""
		seenMsgID = false
		seenMsgStr = false
	}

	for i, raw := range lines {
		line := strings.TrimSpace(raw)

		if line == "" {
			flush()
			reset()
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "msgid "):
			flush()
			reset()
			v, err := parsePOQuoted(strings.TrimPrefix(line, "msgid "))
			if err != nil {
				return nil, fmt.Errorf("line %d: parse msgid: %w", i+1, err)
			}
			currentMsgID.WriteString(v)
			activeField = "msgid"
			seenMsgID = true
		case strings.HasPrefix(line, "msgstr "):
			v, err := parsePOQuoted(strings.TrimPrefix(line, "msgstr "))
			if err != nil {
				return nil, fmt.Errorf("line %d: parse msgstr: %w", i+1, err)
			}
			currentMsgStr.WriteString(v)
			activeField = "msgstr"
			seenMsgStr = true
		case strings.HasPrefix(line, "msgstr["):
			if !strings.HasPrefix(line, "msgstr[0]") {
				continue
			}
			idx := strings.Index(line, "]")
			if idx < 0 || idx+1 >= len(line) {
				return nil, fmt.Errorf("line %d: invalid msgstr[0] format", i+1)
			}
			rest := strings.TrimSpace(line[idx+1:])
			v, err := parsePOQuoted(rest)
			if err != nil {
				return nil, fmt.Errorf("line %d: parse msgstr[0]: %w", i+1, err)
			}
			currentMsgStr.WriteString(v)
			activeField = "msgstr"
			seenMsgStr = true
		case strings.HasPrefix(line, "msgctxt "):
			// Context is currently ignored by the map[string]string strategy output.
			activeField = ""
		case strings.HasPrefix(line, "\""):
			v, err := parsePOQuoted(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: parse continued string: %w", i+1, err)
			}
			switch activeField {
			case "msgid":
				currentMsgID.WriteString(v)
			case "msgstr":
				currentMsgStr.WriteString(v)
			}
		}
	}

	flush()
	return out, nil
}

func parsePOQuoted(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("expected quoted string")
	}
	if !strings.HasPrefix(raw, "\"") {
		return "", fmt.Errorf("expected quoted string, got %q", raw)
	}

	unquoted, err := strconv.Unquote(raw)
	if err != nil {
		return "", err
	}
	return unquoted, nil
}
