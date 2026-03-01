package smartling

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	authAPIBaseURL    = "https://api.smartling.com/auth-api/v2"
	stringsAPIBaseURL = "https://api.smartling.com/strings-api/v2"
	translationsLimit = 500
)

type HTTPClient struct {
	authBaseURL    string
	stringsBaseURL string
	http           *http.Client
	userIdentifier string
	userSecret     string

	tokenMu           sync.Mutex
	cachedAccessToken string
	tokenExpiresAt    time.Time
}

func NewHTTPClient(cfg Config) (*HTTPClient, error) {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &HTTPClient{
		authBaseURL:    authAPIBaseURL,
		stringsBaseURL: stringsAPIBaseURL,
		http:           &http.Client{Timeout: timeout},
		userIdentifier: cfg.UserIdentifier,
		userSecret:     cfg.UserSecret,
	}, nil
}

func (c *HTTPClient) ListTranslations(ctx context.Context, in ListTranslationsInput) ([]StringTranslation, string, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, "", err
	}
	revision := time.Now().UTC().Format(time.RFC3339Nano)
	if len(in.Locales) == 0 {
		return nil, revision, nil
	}

	entries := make([]StringTranslation, 0)
	errs := make([]error, 0)
	for _, locale := range in.Locales {
		trimmedLocale := strings.TrimSpace(locale)
		if trimmedLocale == "" {
			continue
		}
		offset := 0
		for {
			batch, hasMore, err := c.listTranslationsPage(ctx, token, in.ProjectID, trimmedLocale, translationsLimit, offset)
			if err != nil {
				errs = append(errs, err)
				break
			}
			entries = append(entries, batch...)
			if !hasMore {
				break
			}
			offset += translationsLimit
		}
	}

	if len(errs) > 0 {
		return entries, revision, errors.Join(errs...)
	}

	return entries, revision, nil
}

func (c *HTTPClient) UpsertTranslations(ctx context.Context, in UpsertTranslationsInput) (string, error) {
	if len(in.Entries) == 0 {
		return time.Now().UTC().Format(time.RFC3339Nano), nil
	}
	token, err := c.accessToken(ctx)
	if err != nil {
		return "", err
	}

	grouped := make(map[string][]StringTranslation)
	for _, entry := range in.Entries {
		key := strings.TrimSpace(entry.Key)
		locale := strings.TrimSpace(entry.Locale)
		if key == "" || locale == "" || strings.TrimSpace(entry.Value) == "" {
			continue
		}
		grouped[locale] = append(grouped[locale], StringTranslation{
			Key:     key,
			Context: strings.TrimSpace(entry.Context),
			Locale:  locale,
			Value:   entry.Value,
		})
	}

	for locale, items := range grouped {
		if err := c.upsertLocaleTranslations(ctx, token, in.ProjectID, locale, items); err != nil {
			return "", err
		}
	}

	return time.Now().UTC().Format(time.RFC3339Nano), nil
}

func (c *HTTPClient) authenticate(ctx context.Context) (string, error) {
	payload := map[string]string{
		"userIdentifier": c.userIdentifier,
		"userSecret":     c.userSecret,
	}

	var resp struct {
		Response struct {
			Code string `json:"code"`
		} `json:"response"`
		Data struct {
			AccessToken string `json:"accessToken"`
			ExpiresIn   int    `json:"expiresIn"`
		} `json:"data"`
	}

	if err := c.postJSON(ctx, c.authBaseURL+"/authenticate", "", payload, &resp); err != nil {
		return "", fmt.Errorf("authenticate: %w", err)
	}
	if strings.TrimSpace(resp.Data.AccessToken) == "" {
		return "", fmt.Errorf("authenticate: empty access token")
	}

	c.tokenMu.Lock()
	c.cachedAccessToken = resp.Data.AccessToken
	if resp.Data.ExpiresIn > 0 {
		expiry := time.Now().UTC().Add(time.Duration(resp.Data.ExpiresIn) * time.Second)
		// Refresh slightly before expiry to avoid edge-of-window failures.
		c.tokenExpiresAt = expiry.Add(-15 * time.Second)
	} else {
		c.tokenExpiresAt = time.Time{}
	}
	c.tokenMu.Unlock()

	return resp.Data.AccessToken, nil
}

func (c *HTTPClient) accessToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	cached := c.cachedAccessToken
	expiresAt := c.tokenExpiresAt
	c.tokenMu.Unlock()

	now := time.Now().UTC()
	if strings.TrimSpace(cached) != "" {
		if expiresAt.IsZero() || now.Before(expiresAt) {
			return cached, nil
		}
	}
	return c.authenticate(ctx)
}

func (c *HTTPClient) listTranslationsPage(ctx context.Context, token string, projectID string, locale string, limit int, offset int) ([]StringTranslation, bool, error) {
	endpoint := fmt.Sprintf("%s/projects/%s/translations", c.stringsBaseURL, url.PathEscape(projectID))
	params := url.Values{}
	params.Set("targetLocaleId", locale)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("offset", fmt.Sprintf("%d", offset))
	endpoint = endpoint + "?" + params.Encode()

	var resp struct {
		Response struct {
			Code string `json:"code"`
		} `json:"response"`
		Data struct {
			Items []struct {
				StringText       string `json:"stringText"`
				ParsedStringText string `json:"parsedStringText"`
				Translation      string `json:"translation"`
				Instruction      string `json:"instruction"`
				FileURI          string `json:"fileUri"`
				TargetLocaleID   string `json:"targetLocaleId"`
			} `json:"items"`
		} `json:"data"`
	}

	if err := c.getJSON(ctx, endpoint, token, &resp); err != nil {
		return nil, false, fmt.Errorf("list translations %s: %w", locale, err)
	}

	out := make([]StringTranslation, 0, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		key := strings.TrimSpace(item.ParsedStringText)
		if key == "" {
			key = strings.TrimSpace(item.StringText)
		}
		if key == "" {
			continue
		}
		contextValue := strings.TrimSpace(item.Instruction)
		if contextValue == "" {
			contextValue = strings.TrimSpace(item.FileURI)
		}
		targetLocale := strings.TrimSpace(item.TargetLocaleID)
		if targetLocale == "" {
			targetLocale = locale
		}
		out = append(out, StringTranslation{
			Key:     key,
			Context: contextValue,
			Locale:  targetLocale,
			Value:   item.Translation,
		})
	}

	return out, len(resp.Data.Items) == limit, nil
}

func (c *HTTPClient) upsertLocaleTranslations(ctx context.Context, token string, projectID string, locale string, entries []StringTranslation) error {
	endpoint := fmt.Sprintf("%s/projects/%s/locales/%s/translations", c.stringsBaseURL, url.PathEscape(projectID), url.PathEscape(locale))
	payload := map[string]any{"items": entries}
	if err := c.putJSON(ctx, endpoint, token, payload, nil); err != nil {
		return fmt.Errorf("upsert translations %s: %w", locale, err)
	}
	return nil
}

func (c *HTTPClient) getJSON(ctx context.Context, endpoint string, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.do(req, out)
}

func (c *HTTPClient) postJSON(ctx context.Context, endpoint string, token string, payload any, out any) error {
	return c.sendJSON(ctx, http.MethodPost, endpoint, token, payload, out)
}

func (c *HTTPClient) putJSON(ctx context.Context, endpoint string, token string, payload any, out any) error {
	return c.sendJSON(ctx, http.MethodPut, endpoint, token, payload, out)
}

func (c *HTTPClient) sendJSON(ctx context.Context, method string, endpoint string, token string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.do(req, out)
}

func (c *HTTPClient) do(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
