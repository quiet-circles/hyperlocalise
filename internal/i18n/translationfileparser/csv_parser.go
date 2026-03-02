package translationfileparser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"slices"
	"strings"
)

// CSVParser parses CSV translation files.
type CSVParser struct {
	KeyColumn   string
	ValueColumn string
	Delimiter   rune
}

func (p CSVParser) Parse(content []byte) (map[string]string, error) {
	records, err := readCSVRecords(content, p.Delimiter)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return map[string]string{}, nil
	}

	headers := normalizeCSVHeaders(records[0])
	keyIdx, valueIdx, err := resolveCSVColumns(headers, p.KeyColumn, p.ValueColumn)
	if err != nil {
		return nil, err
	}

	out := map[string]string{}
	for i, row := range records[1:] {
		if keyIdx >= len(row) {
			continue
		}
		key := strings.TrimSpace(row[keyIdx])
		if key == "" {
			continue
		}
		if valueIdx >= len(row) {
			return nil, fmt.Errorf("csv row %d missing value column", i+2)
		}
		out[key] = row[valueIdx]
	}
	return out, nil
}

func MarshalCSV(template []byte, values map[string]string, parser CSVParser) ([]byte, error) {
	records, err := readCSVRecords(template, parser.Delimiter)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		keyHeader := strings.TrimSpace(parser.KeyColumn)
		if keyHeader == "" {
			keyHeader = "key"
		}
		valueHeader := strings.TrimSpace(parser.ValueColumn)
		if valueHeader == "" {
			valueHeader = "value"
		}
		records = [][]string{{keyHeader, valueHeader}}
	}

	headers := normalizeCSVHeaders(records[0])
	keyIdx, valueIdx, err := resolveCSVColumns(headers, parser.KeyColumn, parser.ValueColumn)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	for i := 1; i < len(records); i++ {
		if keyIdx >= len(records[i]) {
			continue
		}
		key := strings.TrimSpace(records[i][keyIdx])
		if key == "" {
			continue
		}
		if value, ok := values[key]; ok {
			records[i] = ensureCSVLen(records[i], valueIdx+1)
			records[i][valueIdx] = value
		}
		seen[key] = struct{}{}
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		if _, ok := seen[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		row := make([]string, max(keyIdx, valueIdx)+1)
		row[keyIdx] = key
		row[valueIdx] = values[key]
		records = append(records, row)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if parser.Delimiter != 0 {
		w.Comma = parser.Delimiter
	}
	if err := w.WriteAll(records); err != nil {
		return nil, fmt.Errorf("csv write: %w", err)
	}
	return buf.Bytes(), nil
}

func readCSVRecords(content []byte, delimiter rune) ([][]string, error) {
	r := csv.NewReader(bytes.NewReader(content))
	if delimiter != 0 {
		r.Comma = delimiter
	}
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv decode: %w", err)
	}
	return records, nil
}

func normalizeCSVHeaders(headers []string) []string {
	normalized := make([]string, len(headers))
	for i, header := range headers {
		normalized[i] = strings.ToLower(strings.TrimSpace(header))
	}
	return normalized
}

func resolveCSVColumns(headers []string, keyColumn, valueColumn string) (int, int, error) {
	keyIdx := resolveCSVColumn(headers, keyColumn, []string{"key", "id"})
	if keyIdx < 0 {
		return -1, -1, fmt.Errorf("csv key column not found")
	}

	valueFallback := []string{"target", "value", "source"}
	if strings.TrimSpace(valueColumn) == "" {
		for i := range headers {
			if i == keyIdx {
				continue
			}
			valueFallback = append(valueFallback, headers[i])
		}
	}
	valueIdx := resolveCSVColumn(headers, valueColumn, valueFallback)
	if valueIdx < 0 || valueIdx == keyIdx {
		for i := range headers {
			if i != keyIdx {
				valueIdx = i
				break
			}
		}
	}
	if valueIdx < 0 || valueIdx == keyIdx {
		return -1, -1, fmt.Errorf("csv value column not found")
	}

	return keyIdx, valueIdx, nil
}

func resolveCSVColumn(headers []string, preferred string, fallbacks []string) int {
	name := strings.ToLower(strings.TrimSpace(preferred))
	if name != "" {
		for i, header := range headers {
			if header == name {
				return i
			}
		}
		return -1
	}

	for _, candidate := range fallbacks {
		want := strings.ToLower(strings.TrimSpace(candidate))
		if want == "" {
			continue
		}
		for i, header := range headers {
			if header == want {
				return i
			}
		}
	}
	return -1
}

func ensureCSVLen(row []string, n int) []string {
	if len(row) >= n {
		return row
	}
	grown := make([]string, n)
	copy(grown, row)
	return grown
}
