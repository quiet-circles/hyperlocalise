package smartling

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
