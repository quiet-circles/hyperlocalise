package crowdin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	sdkcrowdin "github.com/crowdin/crowdin-api-client-go/crowdin"
	"github.com/crowdin/crowdin-api-client-go/crowdin/model"
)

type HTTPClient struct {
	client *sdkcrowdin.Client
}

type partialUpsertError struct {
	appliedCount int
	cause        error
}

func (e *partialUpsertError) Error() string {
	return fmt.Sprintf("partial upsert: applied %d entries before failure: %v", e.appliedCount, e.cause)
}

func (e *partialUpsertError) Unwrap() error { return e.cause }

func appliedCountFromError(err error) int {
	var partial *partialUpsertError
	if errors.As(err, &partial) {
		return partial.appliedCount
	}
	return 0
}

func NewHTTPClient(cfg Config) (*HTTPClient, error) {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client, err := sdkcrowdin.NewClient(
		cfg.APIToken,
		sdkcrowdin.WithHTTPClient(&http.Client{Timeout: timeout}),
	)
	if err != nil {
		return nil, fmt.Errorf("crowdin client init: %w", err)
	}

	return &HTTPClient{client: client}, nil
}

func (c *HTTPClient) ListStrings(ctx context.Context, in ListStringsInput) ([]StringTranslation, string, error) {
	allowed := make(map[string]struct{}, len(in.Locales))
	for _, locale := range in.Locales {
		trimmed := strings.TrimSpace(locale)
		if trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}

	entries := make([]StringTranslation, 0)
	offset := 0
	limit := 500

	for {
		endpoint := fmt.Sprintf("/api/v2/projects/%s/translations", url.PathEscape(in.ProjectID))
		var response struct {
			Data []struct {
				Data struct {
					Identifier string `json:"identifier"`
					Context    string `json:"context"`
					LanguageID string `json:"languageId"`
					Text       string `json:"text"`
				} `json:"data"`
			} `json:"data"`
			Pagination struct {
				Offset int `json:"offset"`
				Limit  int `json:"limit"`
				Total  int `json:"totalCount"`
			} `json:"pagination"`
		}

		_, err := c.client.Get(ctx, endpoint, &model.ListOptions{Limit: limit, Offset: offset}, &response)
		if err != nil {
			return nil, "", fmt.Errorf("crowdin request %s: %w", endpoint, err)
		}

		for _, item := range response.Data {
			locale := strings.TrimSpace(item.Data.LanguageID)
			if locale == "" {
				continue
			}
			if len(allowed) > 0 {
				if _, ok := allowed[locale]; !ok {
					continue
				}
			}
			value := strings.TrimSpace(item.Data.Text)
			if value == "" {
				continue
			}
			entries = append(entries, StringTranslation{Key: item.Data.Identifier, Context: item.Data.Context, Locale: locale, Value: value})
		}

		if response.Pagination.Limit == 0 {
			break
		}
		offset += response.Pagination.Limit
		if response.Pagination.Total > 0 && offset >= response.Pagination.Total {
			break
		}
		if len(response.Data) < response.Pagination.Limit {
			break
		}
	}

	return entries, time.Now().UTC().Format(time.RFC3339Nano), nil
}

func (c *HTTPClient) UpsertTranslations(ctx context.Context, in UpsertTranslationsInput) (string, error) {
	applied := 0

	for _, entry := range in.Entries {
		if strings.TrimSpace(entry.Key) == "" || strings.TrimSpace(entry.Locale) == "" {
			continue
		}

		payload := map[string]string{"identifier": entry.Key, "languageId": entry.Locale, "text": entry.Value}
		if strings.TrimSpace(entry.Context) != "" {
			payload["context"] = entry.Context
		}
		endpoint := fmt.Sprintf("/api/v2/projects/%s/translations", url.PathEscape(in.ProjectID))
		if _, err := c.client.Post(ctx, endpoint, payload, nil); err != nil {
			return "", &partialUpsertError{
				appliedCount: applied,
				cause:        fmt.Errorf("crowdin request %s: %w", endpoint, err),
			}
		}
		applied++
	}

	return time.Now().UTC().Format(time.RFC3339Nano), nil
}
