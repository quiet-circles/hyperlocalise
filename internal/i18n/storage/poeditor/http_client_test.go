package poeditor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPClientUsesDefaultTimeout(t *testing.T) {
	client, err := NewHTTPClient(Config{})
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}
	if got, want := client.http.Timeout, 30*time.Second; got != want {
		t.Fatalf("unexpected default timeout: got %v want %v", got, want)
	}
}

func TestHTTPClientListTermsFiltersLocales(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/terms/list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/x-www-form-urlencoded") {
			t.Fatalf("unexpected content-type: %s", ct)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("parse form body: %v", err)
		}
		if got := values.Get("api_token"); got != "token" {
			t.Fatalf("unexpected api_token: %q", got)
		}
		if got := values.Get("id"); got != "123" {
			t.Fatalf("unexpected project id: %q", got)
		}

		_, _ = fmt.Fprint(w, `{
			"result":{"code":"200","message":"OK"},
			"terms":[
				{"term":"hello","context":"home","translations":[
					{"language":"fr","content":"bonjour"},
					{"language":"de","content":"hallo"}
				]},
				{"term":"bye","context":"","translations":[
					{"language":"fr","content":"au revoir"}
				]}
			]
		}`)
	}))
	defer srv.Close()

	client := &HTTPClient{
		baseURL: srv.URL,
		http:    srv.Client(),
	}

	entries, revision, err := client.ListTerms(context.Background(), ListTermsInput{
		ProjectID: "123",
		APIToken:  "token",
		Locales:   []string{"fr"},
	})
	if err != nil {
		t.Fatalf("list terms: %v", err)
	}
	if revision == "" {
		t.Fatalf("expected revision")
	}
	if got := len(entries); got != 2 {
		t.Fatalf("expected 2 filtered entries, got %d", got)
	}
	for _, e := range entries {
		if e.Locale != "fr" {
			t.Fatalf("expected filtered locale fr, got %+v", e)
		}
	}
}

func TestHTTPClientUpsertTranslationsSendsGroupedPayload(t *testing.T) {
	var calls []struct {
		Path   string
		Values url.Values
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		calls = append(calls, struct {
			Path   string
			Values url.Values
		}{Path: r.URL.Path, Values: values})

		_, _ = fmt.Fprint(w, `{"result":{"code":"200","message":"OK"}}`)
	}))
	defer srv.Close()

	client := &HTTPClient{
		baseURL: srv.URL,
		http:    srv.Client(),
	}

	revision, err := client.UpsertTranslations(context.Background(), UpsertTranslationsInput{
		ProjectID: "123",
		APIToken:  "token",
		Entries: []TermTranslation{
			{Term: "hello", Context: "home", Locale: "fr", Value: "bonjour"},
			{Term: "bye", Locale: "fr", Value: "au revoir"},
			{Term: "skip", Locale: "", Value: "x"},
			{Term: "hello", Locale: "de", Value: "hallo"},
		},
	})
	if err != nil {
		t.Fatalf("upsert translations: %v", err)
	}
	if revision == "" {
		t.Fatalf("expected revision")
	}
	if got := len(calls); got != 2 {
		t.Fatalf("expected 2 locale-grouped calls, got %d", got)
	}

	for _, call := range calls {
		if call.Path != "/translations/update" {
			t.Fatalf("unexpected path: %s", call.Path)
		}
		if call.Values.Get("api_token") != "token" || call.Values.Get("id") != "123" {
			t.Fatalf("unexpected auth/id form values: %+v", call.Values)
		}
		raw := call.Values.Get("data")
		if raw == "" {
			t.Fatalf("missing data payload")
		}
		var payload []map[string]string
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode data payload: %v", err)
		}
		if len(payload) == 0 {
			t.Fatalf("expected non-empty payload")
		}
	}
}

func TestHTTPClientPostFormHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadRequest)
	}))
	defer srv.Close()

	client := &HTTPClient{
		baseURL: srv.URL,
		http:    srv.Client(),
	}

	err := client.postForm(context.Background(), "/x", url.Values{"a": {"1"}}, &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestHTTPClientPostFormDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "{not-json")
	}))
	defer srv.Close()

	client := &HTTPClient{
		baseURL: srv.URL,
		http:    srv.Client(),
	}

	err := client.postForm(context.Background(), "/x", url.Values{"a": {"1"}}, &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "decode /x response") {
		t.Fatalf("expected decode error, got %v", err)
	}
}
