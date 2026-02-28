package smartling

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

type fakeClient struct {
	items        []StringTranslation
	listRevision string
	upsertIn     UpsertTranslationsInput
}

func (f *fakeClient) ListTranslations(_ context.Context, _ ListTranslationsInput) ([]StringTranslation, string, error) {
	return f.items, f.listRevision, nil
}

func (f *fakeClient) UpsertTranslations(_ context.Context, in UpsertTranslationsInput) (string, error) {
	f.upsertIn = in
	return "rev2", nil
}

func TestParseConfigUsesEnvSecret(t *testing.T) {
	t.Setenv("SMARTLING_USER_SECRET", "secret")

	cfg, err := ParseConfig(json.RawMessage(`{"projectID":"123","userIdentifier":"uid"}`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if got := cfg.UserSecret; got != "secret" {
		t.Fatalf("unexpected secret from env: %q", got)
	}
}

func TestParseConfigMissingSecret(t *testing.T) {
	_ = os.Unsetenv("SMARTLING_USER_SECRET")
	_, err := ParseConfig(json.RawMessage(`{"projectID":"123","userIdentifier":"uid"}`))
	if err == nil || !strings.Contains(err.Error(), "user secret") {
		t.Fatalf("expected missing secret error, got %v", err)
	}
}

func TestParseConfigRejectsInlineSecret(t *testing.T) {
	t.Setenv("SMARTLING_USER_SECRET", "env-secret")
	_, err := ParseConfig(json.RawMessage(`{"projectID":"123","userIdentifier":"uid","userSecret":"inline"}`))
	if err == nil || !strings.Contains(err.Error(), "userSecret is not supported") {
		t.Fatalf("expected inline secret rejection, got %v", err)
	}
}

func TestAdapterPullMapsStringContextLanguage(t *testing.T) {
	client := &fakeClient{items: []StringTranslation{{Key: "hello", Context: "home", Locale: "fr", Value: "bonjour"}}, listRevision: "rev1"}
	adapter, err := NewWithClient(Config{ProjectID: "123", UserIdentifier: "uid", UserSecret: "sec"}, client)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	result, err := adapter.Pull(context.Background(), storage.PullRequest{Locales: []string{"fr"}})
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if got := len(result.Snapshot.Entries); got != 1 {
		t.Fatalf("expected 1 entry, got %d", got)
	}
	entry := result.Snapshot.Entries[0]
	if entry.Key != "hello" || entry.Context != "home" || entry.Locale != "fr" || entry.Value != "bonjour" {
		t.Fatalf("unexpected entry mapping: %+v", entry)
	}
}

func TestAdapterPushGroupsEntries(t *testing.T) {
	client := &fakeClient{}
	adapter, err := NewWithClient(Config{ProjectID: "123", UserIdentifier: "uid", UserSecret: "sec"}, client)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	_, err = adapter.Push(context.Background(), storage.PushRequest{Entries: []storage.Entry{{Key: "hello", Context: "home", Locale: "fr", Value: "bonjour"}}})
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if got := len(client.upsertIn.Entries); got != 1 {
		t.Fatalf("expected 1 upsert entry, got %d", got)
	}
}

func TestNewBuildsAdapterFromRawConfig(t *testing.T) {
	t.Setenv("SMARTLING_USER_SECRET", "secret")
	adapter, err := New(json.RawMessage(`{"projectID":"123","userIdentifier":"uid"}`))
	if err != nil {
		t.Fatalf("new adapter from raw config: %v", err)
	}
	if got := adapter.Name(); got != AdapterName {
		t.Fatalf("unexpected adapter name: %q", got)
	}
}
