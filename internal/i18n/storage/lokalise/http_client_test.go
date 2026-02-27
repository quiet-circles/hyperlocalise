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
		IOS:     "ios.hello",
		Android: "android.hello",
	})
	if got != "hello" {
		t.Fatalf("unexpected key name: %q", got)
	}
}

func TestExtractKeyNameFallsBack(t *testing.T) {
	got := extractKeyName(lokaliseapi.PlatformStrings{
		IOS: "ios.only",
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
