package lilt

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

type fakeClient struct {
	pullOut PullOutput
	pushIn  PushInput
}

func (f *fakeClient) PullTranslations(_ context.Context, _ PullInput) (PullOutput, error) {
	return f.pullOut, nil
}

func (f *fakeClient) PushTranslations(_ context.Context, in PushInput) (PushOutput, error) {
	f.pushIn = in
	return PushOutput{Revision: "rev2", JobIDs: []string{"job1"}}, nil
}

func TestParseConfigUsesEnvToken(t *testing.T) {
	t.Setenv("LILT_API_TOKEN", "secret-token")
	cfg, err := ParseConfig(json.RawMessage(`{"projectID":"p1"}`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.APIToken != "secret-token" {
		t.Fatalf("unexpected token: %q", cfg.APIToken)
	}
}

func TestParseConfigMissingToken(t *testing.T) {
	_ = os.Unsetenv("LILT_API_TOKEN")
	_, err := ParseConfig(json.RawMessage(`{"projectID":"p1"}`))
	if err == nil || !strings.Contains(err.Error(), "API token") {
		t.Fatalf("expected API token error, got %v", err)
	}
}

func TestPullSkipsEmptyAndDedupes(t *testing.T) {
	client := &fakeClient{pullOut: PullOutput{Revision: "rev1", FilesByLocale: map[string][]ExportedFile{"fr": {{Name: "fr.json", Data: []byte(`[{"key":"hello","value":"bonjour"},{"key":"hello","value":"salut"},{"key":"","value":"skip"}]`)}}}}}
	adapter, err := NewWithClient(Config{ProjectID: "p1", APIToken: "t"}, client)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	result, err := adapter.Pull(context.Background(), storage.PullRequest{Locales: []string{"fr"}})
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if len(result.Snapshot.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Snapshot.Entries))
	}
	if got := result.Snapshot.Entries[0].Value; got != "salut" {
		t.Fatalf("expected latest duplicate value, got %q", got)
	}
}

func TestPushBuildsLocalePayloadAndTracksSkipped(t *testing.T) {
	client := &fakeClient{}
	adapter, err := NewWithClient(Config{ProjectID: "p1", APIToken: "t"}, client)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	result, err := adapter.Push(context.Background(), storage.PushRequest{Entries: []storage.Entry{
		{Key: "hello", Locale: "fr", Value: "bonjour"},
		{Key: "hello", Locale: "fr", Value: "salut"},
		{Key: "", Locale: "fr", Value: "invalid"},
	}})
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if len(client.pushIn.Files) != 1 {
		t.Fatalf("expected one locale file, got %d", len(client.pushIn.Files))
	}
	if !strings.Contains(string(client.pushIn.Files[0].Data), "salut") {
		t.Fatalf("expected latest value in payload: %s", string(client.pushIn.Files[0].Data))
	}
	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied id, got %d", len(result.Applied))
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("expected 1 skipped id, got %d", len(result.Skipped))
	}
	if result.Skipped[0] == result.Applied[0] {
		t.Fatalf("entry id must not be both applied and skipped")
	}
	if result.Revision != "rev2" {
		t.Fatalf("unexpected revision: %q", result.Revision)
	}
}
