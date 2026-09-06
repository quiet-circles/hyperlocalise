package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hyperlocalise/hyperlocalise/internal/i18n/storage/smartling"
)

type fakeSmartlingDiscoveryClient struct {
	files    []smartling.FileListItem
	status   smartling.FileStatus
	locales  []smartling.LocaleListItem
	projects []smartling.ProjectListItem
	err      error
	assert   func(kind string, in any)
}

func (f *fakeSmartlingDiscoveryClient) ListFiles(_ context.Context, in smartling.FileListInput) ([]smartling.FileListItem, error) {
	if f.assert != nil {
		f.assert("files", in)
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.files, nil
}

func (f *fakeSmartlingDiscoveryClient) GetFileStatus(_ context.Context, in smartling.FileStatusInput) (smartling.FileStatus, error) {
	if f.assert != nil {
		f.assert("status", in)
	}
	if f.err != nil {
		return smartling.FileStatus{}, f.err
	}
	return f.status, nil
}

func (f *fakeSmartlingDiscoveryClient) ListLocales(_ context.Context, in smartling.LocaleListInput) ([]smartling.LocaleListItem, error) {
	if f.assert != nil {
		f.assert("locales", in)
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.locales, nil
}

func (f *fakeSmartlingDiscoveryClient) ListProjects(_ context.Context, in smartling.ProjectListInput) ([]smartling.ProjectListItem, error) {
	if f.assert != nil {
		f.assert("projects", in)
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.projects, nil
}

func TestSmartlingHelpListsDiscoveryCommands(t *testing.T) {
	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}
	text := out.String()
	for _, name := range []string{"files", "locales", "projects", "upload", "download"} {
		if !strings.Contains(text, name) {
			t.Fatalf("smartling help missing %q:\n%s", name, text)
		}
	}
}

func TestSmartlingFilesListTextAndJSON(t *testing.T) {
	t.Setenv("SMARTLING_USER_IDENTIFIER", "uid")
	t.Setenv("SMARTLING_USER_SECRET", "secret")
	orig := newSmartlingDiscoveryClient
	t.Cleanup(func() { newSmartlingDiscoveryClient = orig })
	newSmartlingDiscoveryClient = func(_ smartling.Config) (smartlingDiscoveryClient, error) {
		return &fakeSmartlingDiscoveryClient{
			files: []smartling.FileListItem{
				{FileURI: "locales/en.json", FileType: "json", LastUploaded: "2017-09-06T20:29:15Z"},
			},
			assert: func(kind string, in any) {
				got := in.(smartling.FileListInput)
				if got.URIMask != "en.json" {
					t.Fatalf("uri mask=%q", got.URIMask)
				}
			},
		}, nil
	}

	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "files", "list", "--project-id", "123", "--uri-mask", "en.json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	if got := out.String(); got != "uri=locales/en.json type=json last_uploaded=2017-09-06T20:29:15Z\n" {
		t.Fatalf("text output=%q", got)
	}

	out.Reset()
	root.SetArgs([]string{"smartling", "files", "list", "--project-id", "123", "--uri-mask", "en.json", "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("json list: %v", err)
	}
	if !strings.Contains(out.String(), `"fileUri": "locales/en.json"`) {
		t.Fatalf("json output=%s", out.String())
	}
}

func TestSmartlingFilesListEmptyAndInvalidOutput(t *testing.T) {
	t.Setenv("SMARTLING_USER_IDENTIFIER", "uid")
	t.Setenv("SMARTLING_USER_SECRET", "secret")
	orig := newSmartlingDiscoveryClient
	t.Cleanup(func() { newSmartlingDiscoveryClient = orig })
	newSmartlingDiscoveryClient = func(_ smartling.Config) (smartlingDiscoveryClient, error) {
		return &fakeSmartlingDiscoveryClient{files: nil}, nil
	}

	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "files", "list", "--project-id", "123"})
	if err := root.Execute(); err != nil {
		t.Fatalf("empty text: %v", err)
	}
	if out.String() != "" {
		t.Fatalf("empty text should print no rows, got %q", out.String())
	}

	out.Reset()
	root.SetArgs([]string{"smartling", "files", "list", "--project-id", "123", "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("empty json: %v", err)
	}
	if strings.TrimSpace(out.String()) != "[]" {
		t.Fatalf("empty json=%q", out.String())
	}

	out.Reset()
	root.SetArgs([]string{"smartling", "files", "list", "--project-id", "123", "--output", "xml"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), `unsupported output format "xml"`) {
		t.Fatalf("expected invalid output error, got %v", err)
	}
}

func TestSmartlingFilesListPrintsManyFiles(t *testing.T) {
	t.Setenv("SMARTLING_USER_IDENTIFIER", "uid")
	t.Setenv("SMARTLING_USER_SECRET", "secret")
	files := make([]smartling.FileListItem, 250)
	for i := range files {
		files[i] = smartling.FileListItem{FileURI: fmt.Sprintf("file-%03d.json", i), FileType: "json", LastUploaded: "t"}
	}
	orig := newSmartlingDiscoveryClient
	t.Cleanup(func() { newSmartlingDiscoveryClient = orig })
	newSmartlingDiscoveryClient = func(_ smartling.Config) (smartlingDiscoveryClient, error) {
		return &fakeSmartlingDiscoveryClient{files: files}, nil
	}

	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "files", "list", "--project-id", "123"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	if got := strings.Count(out.String(), "uri="); got != 250 {
		t.Fatalf("text rows=%d", got)
	}

	out.Reset()
	root.SetArgs([]string{"smartling", "files", "list", "--project-id", "123", "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("json: %v", err)
	}
	var decoded []smartling.FileListItem
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(decoded) != 250 {
		t.Fatalf("json items=%d", len(decoded))
	}
}

func TestSmartlingFilesStatusRequiresFileURIAndZeroTotals(t *testing.T) {
	t.Setenv("SMARTLING_USER_IDENTIFIER", "uid")
	t.Setenv("SMARTLING_USER_SECRET", "secret")

	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "files", "status", "--project-id", "123"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "--file-uri") {
		t.Fatalf("expected --file-uri error, got %v / %s", err, out.String())
	}

	orig := newSmartlingDiscoveryClient
	t.Cleanup(func() { newSmartlingDiscoveryClient = orig })
	newSmartlingDiscoveryClient = func(_ smartling.Config) (smartlingDiscoveryClient, error) {
		return &fakeSmartlingDiscoveryClient{
			status: smartling.FileStatus{
				FileURI:          "locales/en.json",
				FileType:         "json",
				LastUploaded:     "t",
				TotalStringCount: 0,
				TotalWordCount:   0,
				Items:            []smartling.FileStatusLocale{{LocaleID: "fr-FR"}},
			},
		}, nil
	}

	out.Reset()
	root.SetArgs([]string{"smartling", "files", "status", "--project-id", "123", "--file-uri", "locales/en.json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("status: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "locale=fr-FR") || !strings.Contains(text, "total_strings=0") || !strings.Contains(text, "percent=0") {
		t.Fatalf("status output=%s", text)
	}
	if strings.Contains(strings.Split(text, "\n")[0], "total_strings=") {
		t.Fatalf("file header should omit totals: %s", text)
	}
}

func TestSmartlingFilesStatusPercentFromAuthorizedCounts(t *testing.T) {
	t.Setenv("SMARTLING_USER_IDENTIFIER", "uid")
	t.Setenv("SMARTLING_USER_SECRET", "secret")
	orig := newSmartlingDiscoveryClient
	t.Cleanup(func() { newSmartlingDiscoveryClient = orig })
	newSmartlingDiscoveryClient = func(_ smartling.Config) (smartlingDiscoveryClient, error) {
		return &fakeSmartlingDiscoveryClient{
			status: smartling.FileStatus{
				FileURI:          "locales/en.json",
				FileType:         "json",
				LastUploaded:     "t",
				TotalStringCount: 0,
				TotalWordCount:   0,
				Items: []smartling.FileStatusLocale{{
					LocaleID:              "fr-FR",
					CompletedStringCount:  5,
					CompletedWordCount:    8,
					AuthorizedStringCount: 10,
					AuthorizedWordCount:   20,
				}},
			},
		}, nil
	}

	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "files", "status", "--project-id", "123", "--file-uri", "locales/en.json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("status: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "completed_strings=5") || !strings.Contains(text, "total_strings=10") || !strings.Contains(text, "percent=50") {
		t.Fatalf("status output=%s", text)
	}
	if !strings.Contains(text, "completed_words=8") || !strings.Contains(text, "total_words=20") {
		t.Fatalf("status output=%s", text)
	}
}

func TestSmartlingFilesStatusDoesNotDoubleWrapHTTPError(t *testing.T) {
	t.Setenv("SMARTLING_USER_IDENTIFIER", "uid")
	t.Setenv("SMARTLING_USER_SECRET", "secret")
	orig := newSmartlingDiscoveryClient
	t.Cleanup(func() { newSmartlingDiscoveryClient = orig })
	newSmartlingDiscoveryClient = func(_ smartling.Config) (smartlingDiscoveryClient, error) {
		return &fakeSmartlingDiscoveryClient{err: fmt.Errorf("smartling files status: status 404: missing")}, nil
	}

	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "files", "status", "--project-id", "123", "--file-uri", "missing.json"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Count(err.Error(), "smartling files status:") != 1 {
		t.Fatalf("double-wrapped error: %v", err)
	}
}

func TestSmartlingFilesStatusPassesThroughHTTPError(t *testing.T) {
	t.Setenv("SMARTLING_USER_IDENTIFIER", "uid")
	t.Setenv("SMARTLING_USER_SECRET", "secret")
	orig := newSmartlingDiscoveryClient
	t.Cleanup(func() { newSmartlingDiscoveryClient = orig })
	newSmartlingDiscoveryClient = func(_ smartling.Config) (smartlingDiscoveryClient, error) {
		return &fakeSmartlingDiscoveryClient{err: fmt.Errorf("status 404: missing")}, nil
	}

	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "files", "status", "--project-id", "123", "--file-uri", "missing.json"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("expected HTTP error, got %v", err)
	}
}

func TestSmartlingLocalesListAndProjectsList(t *testing.T) {
	t.Setenv("SMARTLING_USER_IDENTIFIER", "uid")
	t.Setenv("SMARTLING_USER_SECRET", "secret")
	enabled := false
	orig := newSmartlingDiscoveryClient
	t.Cleanup(func() { newSmartlingDiscoveryClient = orig })
	newSmartlingDiscoveryClient = func(_ smartling.Config) (smartlingDiscoveryClient, error) {
		return &fakeSmartlingDiscoveryClient{
			locales: []smartling.LocaleListItem{
				{LocaleID: "en-US", Source: true},
				{LocaleID: "de-DE", Source: false, Enabled: &enabled},
			},
			projects: []smartling.ProjectListItem{
				{ProjectID: "proj-1", ProjectName: "App", SourceLocaleID: "en-US"},
			},
		}, nil
	}

	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "locales", "list", "--project-id", "123"})
	if err := root.Execute(); err != nil {
		t.Fatalf("locales: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "locale=en-US source=true") || !strings.Contains(text, "locale=de-DE source=false enabled=false") {
		t.Fatalf("locales output=%s", text)
	}

	out.Reset()
	root.SetArgs([]string{"smartling", "projects", "list", "--account-uid", "acct-1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("projects: %v", err)
	}
	if !strings.Contains(out.String(), "project_id=proj-1 name=App source_locale=en-US") {
		t.Fatalf("projects output=%s", out.String())
	}
}

func TestSmartlingProjectsListRequiresAccountUID(t *testing.T) {
	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "projects", "list"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "--account-uid is required") || !strings.Contains(err.Error(), "glossary download") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSmartlingFilesListRequiresCredentials(t *testing.T) {
	t.Setenv("SMARTLING_USER_IDENTIFIER", "")
	t.Setenv("SMARTLING_USER_SECRET", "")
	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"smartling", "files", "list", "--project-id", "123"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "smartling files list: credentials are required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSmartlingDownloadTranslationsRejectsInvalidRetrievalType(t *testing.T) {
	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{
		"smartling", "download", "translations",
		"--project-id", "123",
		"--target-locale", "fr-FR",
		"--file-uri", "locales/en.json",
		"--retrieval-type", "contextMatchingInstrumented",
		"--dry-run",
	})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "pending, published, or pseudo") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSmartlingUploadSourcesDirectiveDryRunAndInvalid(t *testing.T) {
	root := newRootCmd("test")
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{
		"smartling", "upload", "sources",
		"--project-id", "123",
		"--file", "test.json",
		"--directive", "source_key_paths=/key",
		"--dry-run",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("directive dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "directives=smartling.source_key_paths=/key") {
		t.Fatalf("output=%s", out.String())
	}

	out.Reset()
	root.SetArgs([]string{
		"smartling", "upload", "sources",
		"--project-id", "123",
		"--file", "test.json",
		"--directive", "novalue",
		"--dry-run",
	})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "--directive must be key=value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseSmartlingUploadDirectives(t *testing.T) {
	got, err := parseSmartlingUploadDirectives([]string{"foo=bar", "smartling.baz=qux=1"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got["smartling.foo"] != "bar" || got["smartling.baz"] != "qux=1" {
		t.Fatalf("got %+v", got)
	}
	_, err = parseSmartlingUploadDirectives([]string{"nope"})
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = parseSmartlingUploadDirectives([]string{"source_key_paths=/a", "smartling.source_key_paths=/b"})
	if err == nil || !strings.Contains(err.Error(), "duplicate --directive smartling.source_key_paths") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	summary := formatSmartlingDirectiveSummary(map[string]string{"smartling.foo": "bar"})
	if summary != "smartling.foo=bar" {
		t.Fatalf("summary=%q", summary)
	}
}
