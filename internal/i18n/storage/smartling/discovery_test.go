package smartling

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestHTTPClientListFilesPagesAcrossThreeResponses(t *testing.T) {
	const total = 201
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/files/list":
			if r.URL.Query().Get("orderBy") != "fileUri" {
				t.Fatalf("expected orderBy=fileUri, got %q", r.URL.Query().Get("orderBy"))
			}
			if r.URL.Query().Get("limit") != "100" {
				t.Fatalf("expected limit=100, got %q", r.URL.Query().Get("limit"))
			}
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
			writeFilesListPage(w, offset, 100, total)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	files, err := client.ListFiles(context.Background(), FileListInput{ProjectID: "123"})
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != total {
		t.Fatalf("got %d files, want %d", len(files), total)
	}
	if files[0].FileURI != "file-000.json" || files[200].FileURI != "file-200.json" {
		t.Fatalf("unexpected first/last URIs: %q %q", files[0].FileURI, files[200].FileURI)
	}
}

func TestHTTPClientListFilesStopsOnEmptyPageWithLyingTotalCount(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/files/list":
			calls++
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS","data":{"totalCount":99999,"items":[]}}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	files, err := client.ListFiles(context.Background(), FileListInput{ProjectID: "123", URIMask: "en.json"})
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("got %d files, want 0", len(files))
	}
	if calls != 1 {
		t.Fatalf("expected 1 list call after empty page, got %d", calls)
	}
}

func TestHTTPClientListFilesStopsOnShortLastPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/files/list":
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
			writeFilesListPage(w, offset, 100, 103)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	files, err := client.ListFiles(context.Background(), FileListInput{ProjectID: "123"})
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 103 {
		t.Fatalf("got %d files, want 103", len(files))
	}
}

func TestHTTPClientListFilesStopsWhenOffsetDoesNotAdvance(t *testing.T) {
	origLimit := filesListPageLimit
	origMax := filesListMaxPages
	filesListPageLimit = 100
	filesListMaxPages = 3
	t.Cleanup(func() {
		filesListPageLimit = origLimit
		filesListMaxPages = origMax
	})

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/files/list":
			calls++
			// Ignore offset so the client would loop unless the page cap fires.
			writeFilesListPageWithTotal(w, 0, 100, 100, 99999)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	_, err := client.ListFiles(context.Background(), FileListInput{ProjectID: "123"})
	if err == nil || !strings.Contains(err.Error(), "exceeded maximum page count") {
		t.Fatalf("expected page cap error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 list calls, got %d", calls)
	}
}

func TestHTTPClientListFilesPassesURIMask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/files/list":
			if got := r.URL.Query().Get("uriMask"); got != "en.json" {
				t.Fatalf("uriMask=%q", got)
			}
			writeFilesListPage(w, 0, 100, 1)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	files, err := client.ListFiles(context.Background(), FileListInput{ProjectID: "123", URIMask: "en.json"})
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files", len(files))
	}
}

func TestHTTPClientGetFileStatusZeroTotals(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/file/status":
			if r.URL.Query().Get("fileUri") != "locales/en.json" {
				t.Fatalf("fileUri=%q", r.URL.Query().Get("fileUri"))
			}
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS","data":{"fileUri":"locales/en.json","fileType":"json","lastUploaded":"2017-09-06T20:29:15Z","totalStringCount":0,"totalWordCount":0,"items":[{"localeId":"fr-FR","completedStringCount":5,"completedWordCount":8,"authorizedStringCount":10,"authorizedWordCount":20}]}}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	status, err := client.GetFileStatus(context.Background(), FileStatusInput{ProjectID: "123", FileURI: "locales/en.json"})
	if err != nil {
		t.Fatalf("GetFileStatus: %v", err)
	}
	if status.TotalStringCount != 0 || status.TotalWordCount != 0 {
		t.Fatalf("unexpected file totals: %+v", status)
	}
	if status.Items[0].AuthorizedStringCount != 10 || LocaleStatusPercent(status.Items[0]) != 50 {
		t.Fatalf("unexpected locale progress: %+v percent=%d", status.Items[0], LocaleStatusPercent(status.Items[0]))
	}
}

func TestHTTPClientGetFileStatusHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/file/status":
			http.Error(w, `{"response":{"code":"VALIDATION_ERROR"}}`, http.StatusNotFound)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	_, err := client.GetFileStatus(context.Background(), FileStatusInput{ProjectID: "123", FileURI: "missing.json"})
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("expected HTTP error, got %v", err)
	}
}

func TestHTTPClientListLocalesIncludesDisabledTargets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/proj-1":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS","data":{"projectId":"proj-1","sourceLocaleId":"en-US","sourceLocaleDescription":"English","targetLocales":[{"localeId":"fr-FR","description":"French","enabled":true},{"localeId":"de-DE","description":"German","enabled":false}]}}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	locales, err := client.ListLocales(context.Background(), LocaleListInput{ProjectID: "proj-1"})
	if err != nil {
		t.Fatalf("ListLocales: %v", err)
	}
	if len(locales) != 3 {
		t.Fatalf("got %d locales: %+v", len(locales), locales)
	}
	if !locales[0].Source || locales[0].LocaleID != "en-US" {
		t.Fatalf("source locale: %+v", locales[0])
	}
	if locales[2].LocaleID != "de-DE" || locales[2].Enabled == nil || *locales[2].Enabled {
		t.Fatalf("disabled locale dropped or wrong: %+v", locales[2])
	}
}

func TestHTTPClientListProjectsPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/accounts/acct-1/projects":
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
			writeProjectsListPage(w, offset, 100, 101)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	projects, err := client.ListProjects(context.Background(), ProjectListInput{AccountUID: "acct-1"})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 101 {
		t.Fatalf("got %d projects, want 101", len(projects))
	}
}

func TestHTTPClientDownloadTranslationFileRetrievalType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/locales/fr-FR/file":
			if r.URL.Query().Get("fileUri") != "locales/en.json" {
				t.Fatalf("fileUri=%q", r.URL.Query().Get("fileUri"))
			}
			if r.URL.Query().Get("retrievalType") != "published" {
				t.Fatalf("retrievalType=%q", r.URL.Query().Get("retrievalType"))
			}
			_, _ = io.WriteString(w, `{"hello":"bonjour"}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	result, err := client.DownloadTranslationFile(context.Background(), TranslationDownloadInput{
		ProjectID:     "123",
		FileURI:       "locales/en.json",
		LocaleID:      "fr-FR",
		RetrievalType: "published",
	})
	if err != nil {
		t.Fatalf("DownloadTranslationFile: %v", err)
	}
	if string(result.Content) != `{"hello":"bonjour"}` {
		t.Fatalf("content=%q", result.Content)
	}
}

func TestHTTPClientDownloadTranslationFileOmitsRetrievalTypeWhenUnset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/locales/fr-FR/file":
			if _, ok := r.URL.Query()["retrievalType"]; ok {
				t.Fatalf("retrievalType should be omitted, query=%s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"hello":"bonjour"}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	_, err := client.DownloadTranslationFile(context.Background(), TranslationDownloadInput{
		ProjectID: "123",
		FileURI:   "locales/en.json",
		LocaleID:  "fr-FR",
	})
	if err != nil {
		t.Fatalf("DownloadTranslationFile: %v", err)
	}
}

func TestNormalizeRetrievalTypeRejectsUnknown(t *testing.T) {
	_, err := NormalizeRetrievalType("contextMatchingInstrumented")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = NormalizeRetrievalType("published")
	if err != nil {
		t.Fatalf("published: %v", err)
	}
}

func TestHTTPClientUploadSourceFileDirectives(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authenticate":
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"accessToken":"token"}}`)
		case "/projects/123/file":
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("parse multipart: %v", err)
			}
			if got := r.FormValue("smartling.source_key_paths"); got != "/key" {
				t.Fatalf("source_key_paths=%q", got)
			}
			if got := r.FormValue("smartling.placeholder_format_custom"); got != "\\{.+?\\}" {
				t.Fatalf("placeholder=%q", got)
			}
			_, _ = fmt.Fprint(w, `{"response":{"code":"SUCCESS"},"data":{"overWritten":false,"stringCount":1,"wordCount":1}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := discoveryTestClient(srv)
	tempFile, err := os.CreateTemp("", "dir-*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, _ = tempFile.WriteString(`{"k":"v"}`)
	_ = tempFile.Close()

	_, err = client.UploadSourceFile(context.Background(), SourceUploadInput{
		ProjectID: "123",
		FileURI:   "test.json",
		FilePath:  tempFile.Name(),
		FileType:  "json",
		Authorize: true,
		Directives: map[string]string{
			"source_key_paths":                    "/key",
			"smartling.placeholder_format_custom": `\{.+?\}`,
		},
	})
	if err != nil {
		t.Fatalf("UploadSourceFile: %v", err)
	}
}

func discoveryTestClient(srv *httptest.Server) *HTTPClient {
	return &HTTPClient{
		authBaseURL:     srv.URL,
		filesBaseURL:    srv.URL,
		projectsBaseURL: srv.URL,
		accountsBaseURL: srv.URL,
		http:            srv.Client(),
		userIdentifier:  "id",
		userSecret:      "secret",
	}
}

func writeFilesListPage(w http.ResponseWriter, offset, limit, total int) {
	writeFilesListPageWithTotal(w, offset, limit, total, total)
}

func writeFilesListPageWithTotal(w http.ResponseWriter, offset, limit, itemTotal, reportedTotal int) {
	type item struct {
		FileURI      string `json:"fileUri"`
		FileType     string `json:"fileType"`
		LastUploaded string `json:"lastUploaded"`
	}
	end := offset + limit
	if end > itemTotal {
		end = itemTotal
	}
	items := make([]item, 0)
	if offset < itemTotal {
		for i := offset; i < end; i++ {
			items = append(items, item{
				FileURI:      fmt.Sprintf("file-%03d.json", i),
				FileType:     "json",
				LastUploaded: "2017-09-06T20:29:15Z",
			})
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"response": map[string]any{
			"code": "SUCCESS",
			"data": map[string]any{
				"totalCount": reportedTotal,
				"items":      items,
			},
		},
	})
}

func writeProjectsListPage(w http.ResponseWriter, offset, limit, total int) {
	end := offset + limit
	if end > total {
		end = total
	}
	items := make([]ProjectListItem, 0)
	if offset < total {
		for i := offset; i < end; i++ {
			items = append(items, ProjectListItem{
				ProjectID:      fmt.Sprintf("proj-%03d", i),
				ProjectName:    fmt.Sprintf("Project %d", i),
				SourceLocaleID: "en-US",
			})
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"response": map[string]any{
			"code": "SUCCESS",
			"data": map[string]any{
				"totalCount": total,
				"items":      items,
			},
		},
	})
}

func TestCanonicalDirectiveField(t *testing.T) {
	if got := CanonicalDirectiveField("source_key_paths"); got != "smartling.source_key_paths" {
		t.Fatalf("got %q", got)
	}
	if got := CanonicalDirectiveField("smartling.foo"); got != "smartling.foo" {
		t.Fatalf("got %q", got)
	}
}
