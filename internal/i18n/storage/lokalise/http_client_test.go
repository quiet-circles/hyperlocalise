package lokalise

import (
	"testing"
	"time"

	lokaliseapi "github.com/lokalise/go-lokalise-api/v5"
)

func TestNewHTTPClientUsesDefaultTimeout(t *testing.T) {
	client, err := NewHTTPClient(Config{APIToken: "token"})
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}
	if client == nil || client.api == nil {
		t.Fatalf("expected initialized api client")
	}
}

func TestExtractKeyNamePrefersWeb(t *testing.T) {
	got := extractKeyName(lokaliseapi.PlatformStrings{
		Web:     "hello",
		Ios:     "ios.hello",
		Android: "android.hello",
	})
	if got != "hello" {
		t.Fatalf("unexpected key name: %q", got)
	}
}

func TestExtractKeyNameFallsBack(t *testing.T) {
	got := extractKeyName(lokaliseapi.PlatformStrings{
		Ios: "ios.only",
	})
	if got != "ios.only" {
		t.Fatalf("unexpected fallback key name: %q", got)
	}
}

func TestUpsertTranslationsNoEntriesReturnsRevision(t *testing.T) {
	client, err := NewHTTPClient(Config{APIToken: "token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}

	revision, err := client.UpsertTranslations(t.Context(), UpsertTranslationsInput{})
	if err != nil {
		t.Fatalf("upsert empty entries: %v", err)
	}
	if revision == "" {
		t.Fatalf("expected non-empty revision")
	}
	if _, err := time.Parse(time.RFC3339Nano, revision); err != nil {
		t.Fatalf("expected revision timestamp, got %q (%v)", revision, err)
	}
}

func TestGroupEntriesByKey(t *testing.T) {
	result := groupEntriesByKey([]KeyTranslation{
		{Key: " hello ", Context: "ctx", Locale: " fr ", Value: "bonjour"},
		{Key: "hello", Context: "ctx", Locale: "de", Value: "hallo"},
		{Key: "", Context: "ctx", Locale: "fr", Value: "x"},
		{Key: "hello", Context: "ctx", Locale: "", Value: "x"},
	})

	group := groupedKey{Key: "hello", Context: "ctx"}
	translations, ok := result[group]
	if !ok {
		t.Fatalf("expected grouped key %+v", group)
	}
	if got := len(translations); got != 2 {
		t.Fatalf("expected 2 grouped translations, got %d", got)
	}
}

func TestBuildNewKey(t *testing.T) {
	newKey := buildNewKey(
		groupedKey{Key: "checkout.submit", Context: "button label"},
		[]lokaliseapi.NewTranslation{{LanguageISO: "fr", Translation: "Valider"}},
	)

	if got, ok := newKey.KeyName.(map[string]string); !ok || got["web"] != "checkout.submit" {
		t.Fatalf("unexpected key name payload: %#v", newKey.KeyName)
	}
	if newKey.Description == nil || *newKey.Description != "button label" {
		t.Fatalf("expected description to be set")
	}
	if newKey.Platforms == nil || len(*newKey.Platforms) != 1 || (*newKey.Platforms)[0] != "web" {
		t.Fatalf("unexpected platforms payload: %#v", newKey.Platforms)
	}
	if newKey.Translations == nil || len(*newKey.Translations) != 1 {
		t.Fatalf("unexpected translations payload: %#v", newKey.Translations)
	}
}
