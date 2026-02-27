package lokalise

import (
	"context"
	"fmt"
	"strings"
	"time"

	lokaliseapi "github.com/lokalise/go-lokalise-api/v5"
)

type HTTPClient struct {
	api *lokaliseapi.Api
}

func NewHTTPClient(cfg Config) (*HTTPClient, error) {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	api, err := lokaliseapi.New(
		cfg.APIToken,
		lokaliseapi.WithConnectionTimeout(timeout),
	)
	if err != nil {
		return nil, fmt.Errorf("lokalise client init: %w", err)
	}

	return &HTTPClient{api: api}, nil
}

func (c *HTTPClient) ListKeys(_ context.Context, in ListKeysInput) ([]KeyTranslation, string, error) {
	revision := time.Now().UTC().Format(time.RFC3339Nano)
	allowed := make(map[string]struct{}, len(in.Locales))
	for _, locale := range in.Locales {
		trimmed := strings.TrimSpace(locale)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}

	cursor := ""
	out := make([]KeyTranslation, 0)

	for {
		keysSvc := c.api.Keys()
		keysSvc.SetListOptions(lokaliseapi.KeyListOptions{
			IncludeTranslations: "1",
			Limit:               500,
			CursorPagination:    true,
			Cursor:              cursor,
		})

		resp, err := keysSvc.List(in.ProjectID)
		if err != nil {
			return nil, "", fmt.Errorf("list keys: %w", err)
		}

		for _, key := range resp.Keys {
			keyName := extractKeyName(key.KeyName)
			if keyName == "" {
				continue
			}
			for _, tr := range key.Translations {
				locale := strings.TrimSpace(tr.LanguageISO)
				if locale == "" {
					continue
				}
				if len(allowed) > 0 {
					if _, ok := allowed[locale]; !ok {
						continue
					}
				}
				value := strings.TrimSpace(tr.Translation)
				if value == "" {
					continue
				}
				out = append(out, KeyTranslation{
					Key:     keyName,
					Context: key.Description,
					Locale:  locale,
					Value:   value,
				})
			}
		}

		if !resp.HasNextCursor() {
			break
		}
		cursor = resp.NextCursor()
	}

	return out, revision, nil
}

func (c *HTTPClient) UpsertTranslations(_ context.Context, in UpsertTranslationsInput) (string, error) {
	type groupedKey struct {
		Key     string
		Context string
	}

	byKey := make(map[groupedKey][]lokaliseapi.NewTranslation)
	for _, entry := range in.Entries {
		if strings.TrimSpace(entry.Key) == "" || strings.TrimSpace(entry.Locale) == "" {
			continue
		}
		group := groupedKey{Key: entry.Key, Context: entry.Context}
		byKey[group] = append(byKey[group], lokaliseapi.NewTranslation{
			LanguageISO: entry.Locale,
			Translation: entry.Value,
		})
	}

	newKeys := make([]lokaliseapi.NewKey, 0, len(byKey))
	for group, translations := range byKey {
		platforms := []string{"web"}
		trans := translations
		newKey := lokaliseapi.NewKey{
			KeyName:      map[string]string{"web": group.Key},
			Platforms:    &platforms,
			Translations: &trans,
		}
		if strings.TrimSpace(group.Context) != "" {
			context := group.Context
			newKey.Description = &context
		}
		newKeys = append(newKeys, newKey)
	}

	if len(newKeys) == 0 {
		return time.Now().UTC().Format(time.RFC3339Nano), nil
	}

	if _, err := c.api.Keys().Create(in.ProjectID, newKeys); err != nil {
		return "", fmt.Errorf("create keys: %w", err)
	}

	return time.Now().UTC().Format(time.RFC3339Nano), nil
}

func extractKeyName(platforms lokaliseapi.PlatformStrings) string {
	candidates := []string{platforms.Web, platforms.IOS, platforms.Android, platforms.Other}
	for _, candidate := range candidates {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
