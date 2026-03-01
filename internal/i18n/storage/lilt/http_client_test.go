package lilt

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPullTranslationsPollsAndDownloads(t *testing.T) {
	polls := 0
	serverURL := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/files/export":
			_, _ = io.WriteString(w, `{"job_id":"job-fr"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/files/export/job-fr":
			polls++
			if polls == 1 {
				_, _ = io.WriteString(w, `{"status":"running"}`)
				return
			}
			_, _ = io.WriteString(w, `{"status":"completed","download_url":"`+serverURL+`/download/fr"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/download/fr":
			_, _ = io.WriteString(w, `[{"key":"hello","value":"bonjour"}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	serverURL = ts.URL

	client := &HTTPClient{baseURL: ts.URL, http: ts.Client(), pollInterval: 1, maxPolls: 3}
	out, err := client.PullTranslations(context.Background(), PullInput{ProjectID: "p1", APIToken: "token", Locales: []string{"fr"}})
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if len(out.FilesByLocale["fr"]) != 1 {
		t.Fatalf("expected one file for fr")
	}
}

func TestPushTranslationsUploadsEachLocale(t *testing.T) {
	uploads := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/files/upload" {
			uploads++
			_, _ = io.WriteString(w, `{"job_id":"job`+string(rune('0'+uploads))+`"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := &HTTPClient{baseURL: ts.URL, http: ts.Client(), pollInterval: 1, maxPolls: 1}
	out, err := client.PushTranslations(context.Background(), PushInput{ProjectID: "p1", APIToken: "token", Files: []UploadFile{{Locale: "fr", Filename: "fr.json", Data: []byte("{}")}, {Locale: "de", Filename: "de.json", Data: []byte("{}")}}})
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if uploads != 2 {
		t.Fatalf("expected 2 uploads, got %d", uploads)
	}
	if len(out.JobIDs) != 2 {
		t.Fatalf("expected 2 job ids, got %d", len(out.JobIDs))
	}
}

func TestParseArtifactSupportsObjectArrayAndZip(t *testing.T) {
	entries, err := ParseArtifact("fr.json", "fr", []byte(`{"home":{"title":"Bonjour"}}`))
	if err != nil {
		t.Fatalf("parse object: %v", err)
	}
	if len(entries) != 1 || entries[0].Key != "home.title" {
		t.Fatalf("unexpected flattened entries: %+v", entries)
	}

	buf := bytes.NewBuffer(nil)
	zw := zip.NewWriter(buf)
	f, err := zw.Create("fr.json")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	_, _ = io.WriteString(f, `[{"key":"hello","value":"bonjour"}]`)
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	files, err := unzipFiles(buf.Bytes())
	if err != nil {
		t.Fatalf("unzip files: %v", err)
	}
	if len(files) != 1 || !strings.Contains(string(files[0].Data), "bonjour") {
		t.Fatalf("unexpected zip contents")
	}
}

func TestParseArtifactPreservesNonStringLeafValues(t *testing.T) {
	entries, err := ParseArtifact("fr.json", "fr", []byte(`{"count":2,"active":true,"meta":["a",1],"nested":{"ratio":1.5}}`))
	if err != nil {
		t.Fatalf("parse object: %v", err)
	}
	byKey := map[string]string{}
	for _, entry := range entries {
		byKey[entry.Key] = entry.Value
	}
	if byKey["count"] != "2" {
		t.Fatalf("expected count to be preserved, got %q", byKey["count"])
	}
	if byKey["active"] != "true" {
		t.Fatalf("expected active to be preserved, got %q", byKey["active"])
	}
	if byKey["meta"] != `["a",1]` {
		t.Fatalf("expected array to be preserved as json, got %q", byKey["meta"])
	}
	if byKey["nested.ratio"] != "1.5" {
		t.Fatalf("expected nested number to be preserved, got %q", byKey["nested.ratio"])
	}
}
