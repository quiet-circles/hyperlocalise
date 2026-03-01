package crowdin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdkcrowdin "github.com/crowdin/crowdin-api-client-go/crowdin"
	"github.com/crowdin/crowdin-api-client-go/crowdin/model"
)

type HTTPClient struct {
	client *sdkcrowdin.Client
}

const (
	maxUpsertRetries = 3
	retryBaseDelay   = 250 * time.Millisecond
	pageLimit        = 500
)

type partialUpsertError struct {
	sentIndexes []int
	cause       error
}

func (e *partialUpsertError) Error() string {
	return fmt.Sprintf("partial upsert: sent %d entries before failure: %v", len(e.sentIndexes), e.cause)
}

func (e *partialUpsertError) Unwrap() error { return e.cause }

func sentIndexesFromError(err error) []int {
	var partial *partialUpsertError
	if errors.As(err, &partial) {
		out := make([]int, 0, len(partial.sentIndexes))
		out = append(out, partial.sentIndexes...)
		return out
	}
	return nil
}

func retryDelay(attempt int, err error) time.Duration {
	var apiErr *model.ErrorResponse
	if errors.As(err, &apiErr) && apiErr.Response != nil {
		retryAfter := strings.TrimSpace(apiErr.Response.Header.Get("Retry-After"))
		if retryAfter != "" {
			if seconds, convErr := strconv.Atoi(retryAfter); convErr == nil && seconds > 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}

	delay := retryBaseDelay
	for i := 0; i < attempt; i++ {
		delay *= 2
	}
	return delay
}

func isRetryableUpsertError(err error) bool {
	var apiErr *model.ErrorResponse
	if errors.As(err, &apiErr) && apiErr.Response != nil {
		code := apiErr.Response.StatusCode
		return code == http.StatusTooManyRequests || code >= http.StatusInternalServerError
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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

type sourceStringKey struct {
	key     string
	context string
}

type sourceStringMeta struct {
	key     string
	context string
}

type translationLookupKey struct {
	stringID int
	locale   string
}

func parseProjectID(raw string) (int, error) {
	projectID, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || projectID <= 0 {
		return 0, fmt.Errorf("invalid projectID %q: expected positive integer", raw)
	}
	return projectID, nil
}

func indexSourceString(byID map[int]sourceStringMeta, byKey map[sourceStringKey]int, src *model.SourceString) {
	if src == nil || src.ID <= 0 {
		return
	}
	key := strings.TrimSpace(src.Identifier)
	if key == "" {
		return
	}
	context := strings.TrimSpace(src.Context)
	byID[src.ID] = sourceStringMeta{key: key, context: context}
	indexKey := sourceStringKey{key: key, context: context}
	if existingID, exists := byKey[indexKey]; exists && existingID != src.ID {
		// Ambiguous mapping across multiple source strings.
		byKey[indexKey] = -1
		return
	}

	if _, exists := byKey[indexKey]; !exists {
		byKey[indexKey] = src.ID
	}
}

func (c *HTTPClient) listSourceStrings(
	ctx context.Context,
	projectID int,
) (map[int]sourceStringMeta, map[sourceStringKey]int, error) {
	byID := make(map[int]sourceStringMeta)
	byKey := make(map[sourceStringKey]int)
	offset := 0

	for {
		strs, _, err := c.client.SourceStrings.List(ctx, projectID, &model.SourceStringsListOptions{
			ListOptions: model.ListOptions{
				Limit:  pageLimit,
				Offset: offset,
			},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list source strings: %w", err)
		}

		for _, src := range strs {
			indexSourceString(byID, byKey, src)
		}

		if len(strs) < pageLimit {
			break
		}
		offset += pageLimit
	}

	return byID, byKey, nil
}

func (c *HTTPClient) listTranslationTexts(
	ctx context.Context,
	projectID, stringID int,
	locale string,
) (map[string]struct{}, error) {
	texts := make(map[string]struct{})
	offset := 0

	for {
		translations, _, err := c.client.StringTranslations.ListStringTranslations(
			ctx,
			projectID,
			&model.StringTranslationsListOptions{
				StringID:   stringID,
				LanguageID: locale,
				ListOptions: model.ListOptions{
					Limit:  pageLimit,
					Offset: offset,
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("list string translations: %w", err)
		}

		for _, tr := range translations {
			if tr == nil {
				continue
			}
			texts[tr.Text] = struct{}{}
		}

		if len(translations) < pageLimit {
			break
		}
		offset += pageLimit
	}

	return texts, nil
}

func (c *HTTPClient) resolveLocales(ctx context.Context, projectID int, inLocales []string) ([]string, error) {
	out := make([]string, 0, len(inLocales))
	seen := make(map[string]struct{}, len(inLocales))
	for _, locale := range inLocales {
		trimmed := strings.TrimSpace(locale)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) > 0 {
		return out, nil
	}

	project, _, err := c.client.Projects.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("get project: empty response")
	}

	for _, locale := range project.TargetLanguageIDs {
		trimmed := strings.TrimSpace(locale)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out, nil
}

func (c *HTTPClient) ListStrings(ctx context.Context, in ListStringsInput) ([]StringTranslation, string, error) {
	projectID, err := parseProjectID(in.ProjectID)
	if err != nil {
		return nil, "", err
	}

	sourceByID, _, err := c.listSourceStrings(ctx, projectID)
	if err != nil {
		return nil, "", err
	}
	locales, err := c.resolveLocales(ctx, projectID, in.Locales)
	if err != nil {
		return nil, "", err
	}

	entries := make([]StringTranslation, 0)
	for _, locale := range locales {
		offset := 0
		for {
			translations, _, listErr := c.client.StringTranslations.ListLanguageTranslations(
				ctx,
				projectID,
				locale,
				&model.LanguageTranslationsListOptions{
					ListOptions: model.ListOptions{
						Limit:  pageLimit,
						Offset: offset,
					},
				},
			)
			if listErr != nil {
				return nil, "", fmt.Errorf("list language translations (%s): %w", locale, listErr)
			}

			for _, tr := range translations {
				if tr == nil || tr.StringID <= 0 || tr.Text == nil {
					continue
				}
				value := strings.TrimSpace(*tr.Text)
				if value == "" {
					continue
				}
				source, exists := sourceByID[tr.StringID]
				if !exists {
					continue
				}
				entries = append(entries, StringTranslation{
					Key:     source.key,
					Context: source.context,
					Locale:  locale,
					Value:   value,
				})
			}

			if len(translations) < pageLimit {
				break
			}
			offset += pageLimit
		}
	}

	return entries, time.Now().UTC().Format(time.RFC3339Nano), nil
}

func (c *HTTPClient) UpsertTranslations(ctx context.Context, in UpsertTranslationsInput) (string, error) {
	projectID, err := parseProjectID(in.ProjectID)
	if err != nil {
		return "", err
	}
	_, sourceByKey, err := c.listSourceStrings(ctx, projectID)
	if err != nil {
		return "", err
	}
	translationsByTarget := make(map[translationLookupKey]map[string]struct{}, len(in.Entries))

	sentIndexes := make([]int, 0, len(in.Entries))

	for idx, entry := range in.Entries {
		key := strings.TrimSpace(entry.Key)
		locale := strings.TrimSpace(entry.Locale)
		if key == "" || locale == "" {
			continue
		}
		stringID, exists := sourceByKey[sourceStringKey{
			key:     key,
			context: strings.TrimSpace(entry.Context),
		}]
		if !exists {
			return "", &partialUpsertError{
				sentIndexes: sentIndexes,
				cause: fmt.Errorf(
					"source string not found for key=%q context=%q",
					key,
					strings.TrimSpace(entry.Context),
				),
			}
		}
		if stringID < 0 {
			return "", &partialUpsertError{
				sentIndexes: sentIndexes,
				cause: fmt.Errorf(
					"ambiguous source string for key=%q context=%q",
					key,
					strings.TrimSpace(entry.Context),
				),
			}
		}

		target := translationLookupKey{stringID: stringID, locale: locale}
		knownTexts, exists := translationsByTarget[target]
		if !exists {
			knownTexts, err = c.listTranslationTexts(ctx, projectID, stringID, locale)
			if err != nil {
				return "", &partialUpsertError{
					sentIndexes: sentIndexes,
					cause:       err,
				}
			}
			translationsByTarget[target] = knownTexts
		}
		if _, exists = knownTexts[entry.Value]; exists {
			continue
		}

		var lastErr error
		for attempt := 0; attempt <= maxUpsertRetries; attempt++ {
			_, _, reqErr := c.client.StringTranslations.AddTranslation(
				ctx,
				projectID,
				&model.TranslationAddRequest{
					StringID:   stringID,
					LanguageID: locale,
					Text:       entry.Value,
				},
			)
			if reqErr == nil {
				lastErr = nil
				break
			}
			lastErr = reqErr

			if !isRetryableUpsertError(lastErr) || attempt == maxUpsertRetries {
				break
			}
			if err := waitForRetry(ctx, retryDelay(attempt, lastErr)); err != nil {
				lastErr = err
				break
			}
		}
		if lastErr != nil {
			return "", &partialUpsertError{
				sentIndexes: sentIndexes,
				cause:       fmt.Errorf("add translation: %w", lastErr),
			}
		}
		knownTexts[entry.Value] = struct{}{}
		sentIndexes = append(sentIndexes, idx)
	}

	return time.Now().UTC().Format(time.RFC3339Nano), nil
}
