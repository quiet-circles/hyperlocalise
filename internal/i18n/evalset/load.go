package evalset

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/jsonc"
)

var csvAllowedHeaders = map[string]struct{}{
	"id":           {},
	"source":       {},
	"targetLocale": {},
	"context":      {},
	"reference":    {},
	"tags":         {},
	"bucket":       {},
	"group":        {},
}

// Load parses and validates an evaluation dataset from path.
func Load(path string) (*Dataset, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open evalset: %w", err)
	}

	var dataset Dataset

	switch strings.ToLower(filepath.Ext(path)) {
	case ".csv":
		dataset, err = decodeCSV(content)
		if err != nil {
			return nil, err
		}
	default:
		dataset, err = decodeJSON(content)
		if err != nil {
			return nil, err
		}
	}

	if err := dataset.Validate(); err != nil {
		return nil, fmt.Errorf("validate evalset: %w", err)
	}

	return &dataset, nil
}

func decodeJSON(content []byte) (Dataset, error) {
	decoder := json.NewDecoder(bytes.NewReader(jsonc.ToJSON(content)))
	decoder.DisallowUnknownFields()

	var dataset Dataset
	if err := decoder.Decode(&dataset); err != nil {
		return Dataset{}, fmt.Errorf("decode evalset: %w", err)
	}

	if err := expectEOF(decoder); err != nil {
		return Dataset{}, err
	}

	return dataset, nil
}

func decodeCSV(content []byte) (Dataset, error) {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return Dataset{}, fmt.Errorf("decode evalset csv: missing header row")
		}

		return Dataset{}, fmt.Errorf("decode evalset csv: %w", err)
	}

	indexByHeader := make(map[string]int, len(headers))
	for i, header := range headers {
		header = strings.TrimSpace(header)
		if header == "" {
			return Dataset{}, fmt.Errorf("decode evalset csv: header[%d]: must not be empty", i)
		}

		if _, ok := csvAllowedHeaders[header]; !ok {
			return Dataset{}, fmt.Errorf("decode evalset csv: unknown header %q", header)
		}

		if _, exists := indexByHeader[header]; exists {
			return Dataset{}, fmt.Errorf("decode evalset csv: duplicate header %q", header)
		}

		indexByHeader[header] = i
	}

	for _, required := range []string{"id", "source", "targetLocale"} {
		if _, ok := indexByHeader[required]; !ok {
			return Dataset{}, fmt.Errorf("decode evalset csv: missing required header %q", required)
		}
	}

	dataset := Dataset{Cases: make([]Case, 0)}

	for rowNum := 2; ; rowNum++ {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return Dataset{}, fmt.Errorf("decode evalset csv: row %d: %w", rowNum, err)
		}

		row := make([]string, len(headers))
		copy(row, record)

		tc := Case{
			ID:           csvField(row, indexByHeader, "id"),
			Source:       csvField(row, indexByHeader, "source"),
			TargetLocale: csvField(row, indexByHeader, "targetLocale"),
			Context:      csvField(row, indexByHeader, "context"),
			Reference:    csvField(row, indexByHeader, "reference"),
			Bucket:       csvField(row, indexByHeader, "bucket"),
			Group:        csvField(row, indexByHeader, "group"),
		}

		if rawTags := csvField(row, indexByHeader, "tags"); rawTags != "" {
			parts := strings.Split(rawTags, ";")
			tags := make([]string, 0, len(parts))
			for _, part := range parts {
				tag := strings.TrimSpace(part)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
			tc.Tags = tags
		}

		dataset.Cases = append(dataset.Cases, tc)
	}

	return dataset, nil
}

func csvField(row []string, indexByHeader map[string]int, header string) string {
	idx, ok := indexByHeader[header]
	if !ok || idx >= len(row) {
		return ""
	}

	return strings.TrimSpace(row[idx])
}

func expectEOF(decoder *json.Decoder) error {
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}

		return fmt.Errorf("decode trailing evalset content: %w", err)
	}

	return fmt.Errorf("decode trailing evalset content: unexpected trailing JSON value")
}
