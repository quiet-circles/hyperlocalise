package smartling

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

func TestHTTPClientDoHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadRequest)
	}))
	defer srv.Close()

	client := &HTTPClient{http: srv.Client()}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	err := client.do(req, &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestHTTPClientDoDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "{not-json")
	}))
	defer srv.Close()

	client := &HTTPClient{http: srv.Client()}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	err := client.do(req, &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestHTTPClientAuthenticate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authenticate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
	}))
	defer srv.Close()

	client := &HTTPClient{authBaseURL: srv.URL, http: srv.Client(), userIdentifier: "id", userSecret: "secret"}
	token, err := client.authenticate(context.Background())
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if token != "token" {
		t.Fatalf("unexpected token: %q", token)
	}
}

func TestHTTPClientListTranslationsUsesProjectTranslationsEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/translations":
			assertTranslationsQuery(t, r.URL.Query(), "fr", 500, 0)
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"items":[{"parsedStringText":"welcome.title","stringText":"welcome.title","translation":"  Bienvenue  ","instruction":"home","targetLocaleId":"fr"}]}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &HTTPClient{
		authBaseURL:    srv.URL,
		stringsBaseURL: srv.URL,
		http:           srv.Client(),
		userIdentifier: "id",
		userSecret:     "secret",
	}
	items, _, err := client.ListTranslations(context.Background(), ListTranslationsInput{
		ProjectID: "123",
		Locales:   []string{"fr"},
	})
	if err != nil {
		t.Fatalf("list translations: %v", err)
	}
	if got := len(items); got != 1 {
		t.Fatalf("expected 1 item, got %d", got)
	}
	if items[0].Key != "welcome.title" || items[0].Locale != "fr" || items[0].Value != "  Bienvenue  " || items[0].Context != "home" {
		t.Fatalf("unexpected mapping: %+v", items[0])
	}
}

func TestHTTPClientListTranslationsPaginates(t *testing.T) {
	requestedOffsets := make([]int, 0, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/translations":
			offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
			if err != nil {
				t.Fatalf("offset query: %v", err)
			}
			requestedOffsets = append(requestedOffsets, offset)
			if offset == 0 {
				assertTranslationsQuery(t, r.URL.Query(), "fr", 500, 0)
				writeTranslationsItemsResponse(w, 500, 0, "fr")
				return
			}
			if offset == 500 {
				assertTranslationsQuery(t, r.URL.Query(), "fr", 500, 500)
				writeTranslationsItemsResponse(w, 1, 500, "fr")
				return
			}
			t.Fatalf("unexpected offset: %d", offset)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &HTTPClient{
		authBaseURL:    srv.URL,
		stringsBaseURL: srv.URL,
		http:           srv.Client(),
		userIdentifier: "id",
		userSecret:     "secret",
	}
	items, _, err := client.ListTranslations(context.Background(), ListTranslationsInput{
		ProjectID: "123",
		Locales:   []string{"fr"},
	})
	if err != nil {
		t.Fatalf("list translations: %v", err)
	}
	if got := len(items); got != 501 {
		t.Fatalf("expected 501 items, got %d", got)
	}
	if got := len(requestedOffsets); got != 2 {
		t.Fatalf("expected 2 paged requests, got %d (%v)", got, requestedOffsets)
	}
}

func TestHTTPClientListTranslationsAttemptsAllLocalesAndJoinsErrors(t *testing.T) {
	requestedLocales := make([]string, 0, 2)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token","expiresIn":3600}}`)
		case "/projects/123/translations":
			locale := r.URL.Query().Get("targetLocaleId")
			requestedLocales = append(requestedLocales, locale)
			if locale == "fr" {
				http.Error(w, "fr unavailable", http.StatusInternalServerError)
				return
			}
			if locale == "de" {
				_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"items":[{"stringText":"k1","translation":"hallo","targetLocaleId":"de"}]}}`)
				return
			}
			t.Fatalf("unexpected locale: %s", locale)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &HTTPClient{
		authBaseURL:    srv.URL,
		stringsBaseURL: srv.URL,
		http:           srv.Client(),
		userIdentifier: "id",
		userSecret:     "secret",
	}
	items, _, err := client.ListTranslations(context.Background(), ListTranslationsInput{
		ProjectID: "123",
		Locales:   []string{"fr", "de"},
	})
	if err == nil {
		t.Fatal("expected aggregated locale error, got nil")
	}
	if !strings.Contains(err.Error(), "list translations fr") {
		t.Fatalf("expected fr locale error, got %v", err)
	}
	if got := len(requestedLocales); got != 2 {
		t.Fatalf("expected both locales to be attempted, got %d (%v)", got, requestedLocales)
	}
	if got := len(items); got != 1 {
		t.Fatalf("expected successful locale entries to be returned, got %d", got)
	}
	if items[0].Locale != "de" || items[0].Value != "hallo" {
		t.Fatalf("unexpected successful locale item: %+v", items[0])
	}
}

func TestHTTPClientListTranslationsReusesAuthTokenBeforeExpiry(t *testing.T) {
	authenticateCalls := 0
	translationsCalls := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			authenticateCalls++
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token","expiresIn":3600}}`)
		case "/projects/123/translations":
			translationsCalls++
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"items":[{"stringText":"k1","translation":"bonjour","targetLocaleId":"fr"}]}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &HTTPClient{
		authBaseURL:    srv.URL,
		stringsBaseURL: srv.URL,
		http:           srv.Client(),
		userIdentifier: "id",
		userSecret:     "secret",
	}

	_, _, err := client.ListTranslations(context.Background(), ListTranslationsInput{
		ProjectID: "123",
		Locales:   []string{"fr"},
	})
	if err != nil {
		t.Fatalf("first list translations: %v", err)
	}
	_, _, err = client.ListTranslations(context.Background(), ListTranslationsInput{
		ProjectID: "123",
		Locales:   []string{"fr"},
	})
	if err != nil {
		t.Fatalf("second list translations: %v", err)
	}

	if authenticateCalls != 1 {
		t.Fatalf("expected one authenticate call with cached token, got %d", authenticateCalls)
	}
	if translationsCalls != 2 {
		t.Fatalf("expected two translation calls, got %d", translationsCalls)
	}
}

func TestHTTPClientUpsertLocaleTranslationsAllowsNoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/123/locales/fr/translations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := &HTTPClient{
		stringsBaseURL: srv.URL,
		http:           srv.Client(),
	}
	err := client.upsertLocaleTranslations(context.Background(), "token", "123", "fr", []StringTranslation{
		{Key: "welcome.title", Locale: "fr", Value: "Bienvenue"},
	})
	if err != nil {
		t.Fatalf("upsert locale translations: %v", err)
	}
}

func TestHTTPClientUpsertTranslationsPreservesWhitespace(t *testing.T) {
	var putBody struct {
		Items []StringTranslation `json:"items"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/locales/fr/translations":
			if r.Method != http.MethodPut {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if err := json.Unmarshal(body, &putBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := &HTTPClient{
		authBaseURL:    srv.URL,
		stringsBaseURL: srv.URL,
		http:           srv.Client(),
		userIdentifier: "id",
		userSecret:     "secret",
	}
	value := "  Bonjour  "
	_, err := client.UpsertTranslations(context.Background(), UpsertTranslationsInput{
		ProjectID: "123",
		Entries: []StringTranslation{
			{Key: "welcome.title", Locale: "fr", Value: value},
		},
	})
	if err != nil {
		t.Fatalf("upsert translations: %v", err)
	}
	if got := len(putBody.Items); got != 1 {
		t.Fatalf("expected 1 item in PUT payload, got %d", got)
	}
	if got := putBody.Items[0].Value; got != value {
		t.Fatalf("unexpected payload value: got %q want %q", got, value)
	}
}

func assertTranslationsQuery(t *testing.T, values url.Values, locale string, limit int, offset int) {
	t.Helper()
	if got := values.Get("targetLocaleId"); got != locale {
		t.Fatalf("unexpected targetLocaleId: got %q want %q", got, locale)
	}
	if got := values.Get("limit"); got != strconv.Itoa(limit) {
		t.Fatalf("unexpected limit: got %q want %d", got, limit)
	}
	if got := values.Get("offset"); got != strconv.Itoa(offset) {
		t.Fatalf("unexpected offset: got %q want %d", got, offset)
	}
}

func writeTranslationsItemsResponse(w http.ResponseWriter, count int, start int, locale string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"items":[`)
	for i := 0; i < count; i++ {
		if i > 0 {
			_, _ = fmt.Fprint(w, ",")
		}
		idx := start + i
		_, _ = fmt.Fprintf(
			w,
			`{"stringText":"k%d","translation":"v%d","targetLocaleId":"%s"}`,
			idx,
			idx,
			locale,
		)
	}
	_, _ = fmt.Fprint(w, `]}}`)
}
