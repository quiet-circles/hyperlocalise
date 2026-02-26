package poeditor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiBaseURL = "https://api.poeditor.com/v2"

type HTTPClient struct {
	baseURL string
	http    *http.Client
}

func NewHTTPClient(cfg Config) (*HTTPClient, error) {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &HTTPClient{
		baseURL: apiBaseURL,
		http: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *HTTPClient) ListTerms(ctx context.Context, in ListTermsInput) ([]TermTranslation, string, error) {
	var response struct {
		Result struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"result"`
		Terms []struct {
			Term         string `json:"term"`
			Context      string `json:"context"`
			Translations []struct {
				Language string `json:"language"`
				Content  string `json:"content"`
			} `json:"translations"`
		} `json:"terms"`
	}

	values := url.Values{}
	values.Set("api_token", in.APIToken)
	values.Set("id", in.ProjectID)
	if err := c.postForm(ctx, "/terms/list", values, &response); err != nil {
		return nil, "", err
	}

	allowed := make(map[string]struct{}, len(in.Locales))
	for _, locale := range in.Locales {
		allowed[strings.TrimSpace(locale)] = struct{}{}
	}

	out := make([]TermTranslation, 0)
	for _, term := range response.Terms {
		for _, tr := range term.Translations {
			if len(allowed) > 0 {
				if _, ok := allowed[tr.Language]; !ok {
					continue
				}
			}
			out = append(out, TermTranslation{
				Term:    term.Term,
				Context: term.Context,
				Locale:  tr.Language,
				Value:   tr.Content,
			})
		}
	}

	revision := time.Now().UTC().Format(time.RFC3339Nano)
	return out, revision, nil
}

func (c *HTTPClient) UpsertTranslations(ctx context.Context, in UpsertTranslationsInput) (string, error) {
	byLocale := make(map[string][]map[string]string)
	for _, entry := range in.Entries {
		if strings.TrimSpace(entry.Locale) == "" {
			continue
		}
		item := map[string]string{
			"term":        entry.Term,
			"translation": entry.Value,
		}
		if strings.TrimSpace(entry.Context) != "" {
			item["context"] = entry.Context
		}
		byLocale[entry.Locale] = append(byLocale[entry.Locale], item)
	}

	for locale, items := range byLocale {
		raw, err := json.Marshal(items)
		if err != nil {
			return "", fmt.Errorf("marshal poeditor translations payload: %w", err)
		}

		values := url.Values{}
		values.Set("api_token", in.APIToken)
		values.Set("id", in.ProjectID)
		values.Set("language", locale)
		values.Set("data", string(raw))

		var response struct {
			Result struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"result"`
		}
		if err := c.postForm(ctx, "/translations/update", values, &response); err != nil {
			return "", err
		}
	}

	return time.Now().UTC().Format(time.RFC3339Nano), nil
}

func (c *HTTPClient) postForm(ctx context.Context, endpoint string, values url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpoint, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return fmt.Errorf("poeditor request build %s: %w", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("poeditor request %s: %w", endpoint, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("poeditor request %s: status %d: %s", endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("poeditor decode %s response: %w", endpoint, err)
	}

	return nil
}
