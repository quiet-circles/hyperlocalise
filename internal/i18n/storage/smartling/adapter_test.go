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
	fileEntries  []storage.Entry
	fileRevision string
	importIn     ImportFileInput
	listCalls    int
	upsertCalls  int
	exportCalls  int
	importCalls  int
}

func (f *fakeClient) ListTranslations(_ context.Context, _ ListTranslationsInput) ([]StringTranslation, string, error) {
	f.listCalls++
	return f.items, f.listRevision, nil
}

func (f *fakeClient) UpsertTranslations(_ context.Context, in UpsertTranslationsInput) (string, error) {
	f.upsertCalls++
	f.upsertIn = in
	return "rev2", nil
}

func (f *fakeClient) ExportFileEntries(_ context.Context, _ ExportFileInput) ([]storage.Entry, string, error) {
	f.exportCalls++
	return f.fileEntries, f.fileRevision, nil
}

func (f *fakeClient) ImportFileEntries(_ context.Context, in ImportFileInput) (string, error) {
	f.importCalls++
	f.importIn = in
	return "rev-file", nil
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

func TestAdapterPushAppliedOnlyIncludesSentEntries(t *testing.T) {
	client := &fakeClient{}
	adapter, err := NewWithClient(Config{ProjectID: "123", UserIdentifier: "uid", UserSecret: "sec"}, client)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	req := storage.PushRequest{Entries: []storage.Entry{
		{Key: "hello", Context: "home", Locale: "fr", Value: "bonjour"},
		{Key: "goodbye", Context: "home", Locale: "fr", Value: "   "},
		{Key: "   ", Context: "home", Locale: "fr", Value: "au revoir"},
	}}
	result, err := adapter.Push(context.Background(), req)
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if got := len(client.upsertIn.Entries); got != 1 {
		t.Fatalf("expected 1 upsert entry, got %d", got)
	}
	if got := len(result.Applied); got != 1 {
		t.Fatalf("expected 1 applied entry, got %d", got)
	}
	if result.Applied[0] != req.Entries[0].ID() {
		t.Fatalf("unexpected applied entry id: got %v want %v", result.Applied[0], req.Entries[0].ID())
	}
}

func TestAdapterPushPreservesTranslationWhitespace(t *testing.T) {
	client := &fakeClient{}
	adapter, err := NewWithClient(Config{ProjectID: "123", UserIdentifier: "uid", UserSecret: "sec"}, client)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	value := "  Bonjour  "
	_, err = adapter.Push(context.Background(), storage.PushRequest{Entries: []storage.Entry{
		{Key: "hello", Context: "home", Locale: "fr", Value: value},
	}})
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if got := client.upsertIn.Entries[0].Value; got != value {
		t.Fatalf("unexpected pushed value: got %q want %q", got, value)
	}
}

func TestAdapterPushDeduplicatesByEntryID(t *testing.T) {
	client := &fakeClient{}
	adapter, err := NewWithClient(Config{ProjectID: "123", UserIdentifier: "uid", UserSecret: "sec"}, client)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	first := storage.Entry{Key: "hello", Context: "home", Locale: "fr", Value: "bonjour"}
	second := storage.Entry{Key: "hello", Context: "home", Locale: "fr", Value: "salut"}
	third := storage.Entry{Key: "hello", Context: "home", Locale: "de", Value: "hallo"}

	result, err := adapter.Push(context.Background(), storage.PushRequest{Entries: []storage.Entry{
		first,
		second,
		third,
	}})
	if err != nil {
		t.Fatalf("push: %v", err)
	}

	if got := len(client.upsertIn.Entries); got != 2 {
		t.Fatalf("expected 2 upsert entries after dedup, got %d", got)
	}
	if got := client.upsertIn.Entries[0].Value; got != "salut" {
		t.Fatalf("expected latest duplicate value to win, got %q", got)
	}
	if got := len(result.Applied); got != 2 {
		t.Fatalf("expected 2 applied entries after dedup, got %d", got)
	}
	if result.Applied[0] != first.ID() {
		t.Fatalf("unexpected first applied id: got %v want %v", result.Applied[0], first.ID())
	}
	if result.Applied[1] != third.ID() {
		t.Fatalf("unexpected second applied id: got %v want %v", result.Applied[1], third.ID())
	}
}

func TestAdapterModeDefaultsToStrings(t *testing.T) {
	t.Setenv("SMARTLING_USER_SECRET", "secret")
	cfg, err := ParseConfig(json.RawMessage(`{"projectID":"123","userIdentifier":"uid"}`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Mode != ModeStrings {
		t.Fatalf("unexpected default mode: got %q want %q", cfg.Mode, ModeStrings)
	}
}

func TestParseConfigFilesModeRequiresFileURI(t *testing.T) {
	t.Setenv("SMARTLING_USER_SECRET", "secret")
	_, err := ParseConfig(json.RawMessage(`{"projectID":"123","userIdentifier":"uid","mode":"files"}`))
	if err == nil || !strings.Contains(err.Error(), "fileURI") {
		t.Fatalf("expected fileURI requirement error, got %v", err)
	}
}

func TestAdapterModeRoutingPull(t *testing.T) {
	tests := []struct {
		name          string
		mode          string
		wantListCalls int
		wantFileCalls int
	}{
		{name: "strings", mode: ModeStrings, wantListCalls: 1},
		{name: "files", mode: ModeFiles, wantFileCalls: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeClient{
				items:        []StringTranslation{{Key: "hello", Locale: "fr", Value: "bonjour"}},
				listRevision: "rev1",
				fileEntries:  []storage.Entry{{Key: "hello", Locale: "fr", Value: "bonjour"}},
				fileRevision: "rev-file",
			}
			adapter, err := NewWithClient(Config{ProjectID: "123", UserIdentifier: "uid", UserSecret: "sec", Mode: tc.mode, FileURI: "/messages.json"}, client)
			if err != nil {
				t.Fatalf("new adapter: %v", err)
			}

			if _, err := adapter.Pull(context.Background(), storage.PullRequest{Locales: []string{"fr"}}); err != nil {
				t.Fatalf("pull: %v", err)
			}
			if client.listCalls != tc.wantListCalls {
				t.Fatalf("unexpected list calls: got %d want %d", client.listCalls, tc.wantListCalls)
			}
			if client.exportCalls != tc.wantFileCalls {
				t.Fatalf("unexpected file export calls: got %d want %d", client.exportCalls, tc.wantFileCalls)
			}
		})
	}
}

func TestAdapterModeRoutingPush(t *testing.T) {
	tests := []struct {
		name            string
		mode            string
		wantUpsertCalls int
		wantImportCalls int
	}{
		{name: "strings", mode: ModeStrings, wantUpsertCalls: 1},
		{name: "files", mode: ModeFiles, wantImportCalls: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeClient{}
			adapter, err := NewWithClient(Config{ProjectID: "123", UserIdentifier: "uid", UserSecret: "sec", Mode: tc.mode, FileURI: "/messages.json"}, client)
			if err != nil {
				t.Fatalf("new adapter: %v", err)
			}

			_, err = adapter.Push(context.Background(), storage.PushRequest{Entries: []storage.Entry{{Key: "hello", Locale: "fr", Value: "bonjour"}}})
			if err != nil {
				t.Fatalf("push: %v", err)
			}
			if client.upsertCalls != tc.wantUpsertCalls {
				t.Fatalf("unexpected upsert calls: got %d want %d", client.upsertCalls, tc.wantUpsertCalls)
			}
			if client.importCalls != tc.wantImportCalls {
				t.Fatalf("unexpected import calls: got %d want %d", client.importCalls, tc.wantImportCalls)
			}
		})
	}
}

func TestAdapterFileModePullAndPushDeduplicatesAndMatchesStringsSemantics(t *testing.T) {
	client := &fakeClient{
		fileEntries: []storage.Entry{
			{Key: "hello", Locale: "fr", Value: "bonjour"},
			{Key: "goodbye", Locale: "fr", Value: "au revoir"},
			{Key: "", Locale: "fr", Value: "skip"},
		},
		fileRevision: "rev-file",
	}

	adapter, err := NewWithClient(Config{ProjectID: "123", UserIdentifier: "uid", UserSecret: "sec", Mode: ModeFiles, FileURI: "/messages.json"}, client)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	pullRes, err := adapter.Pull(context.Background(), storage.PullRequest{Locales: []string{"fr"}})
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if got := len(pullRes.Snapshot.Entries); got != 2 {
		t.Fatalf("expected 2 pulled file entries, got %d", got)
	}

	pushReq := storage.PushRequest{Entries: []storage.Entry{
		{Key: "hello", Context: "home", Locale: "fr", Value: "bonjour"},
		{Key: "hello", Context: "home", Locale: "fr", Value: "salut"},
		{Key: "goodbye", Context: "home", Locale: "fr", Value: "au revoir"},
		{Key: "skip", Context: "home", Locale: "fr", Value: "   "},
	}}
	pushRes, err := adapter.Push(context.Background(), pushReq)
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if got := len(client.importIn.Entries); got != 2 {
		t.Fatalf("expected 2 imported file entries after dedupe, got %d", got)
	}
	if got := client.importIn.Entries[0].Value; got != "salut" {
		t.Fatalf("expected latest duplicate value to win, got %q", got)
	}
	if got := len(pushRes.Applied); got != 2 {
		t.Fatalf("expected 2 applied file entries, got %d", got)
	}
	if pushRes.Applied[0] != pushReq.Entries[0].ID() {
		t.Fatalf("unexpected first applied id: got %v want %v", pushRes.Applied[0], pushReq.Entries[0].ID())
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
