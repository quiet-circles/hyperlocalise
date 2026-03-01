package lilt

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

const apiBaseURL = "https://api.lilt.com/v2"

type HTTPClient struct {
	baseURL      string
	http         *http.Client
	pollInterval time.Duration
	maxPolls     int
}

func NewHTTPClient(cfg Config) (*HTTPClient, error) {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	poll := time.Duration(cfg.PollIntervalMS) * time.Millisecond
	if poll <= 0 {
		poll = time.Second
	}
	maxPolls := cfg.MaxPolls
	if maxPolls <= 0 {
		maxPolls = 60
	}
	return &HTTPClient{baseURL: apiBaseURL, http: &http.Client{Timeout: timeout}, pollInterval: poll, maxPolls: maxPolls}, nil
}

type exportStartResponse struct {
	ID    string `json:"id"`
	JobID string `json:"job_id"`
}

type exportStatusResponse struct {
	Status      string `json:"status"`
	DownloadURL string `json:"download_url"`
	Message     string `json:"message"`
}

type importResponse struct {
	ID     string `json:"id"`
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

func (c *HTTPClient) PullTranslations(ctx context.Context, in PullInput) (PullOutput, error) {
	filesByLocale := map[string][]ExportedFile{}
	for _, locale := range in.Locales {
		locale = strings.TrimSpace(locale)
		if locale == "" {
			continue
		}
		jobID, err := c.startExport(ctx, in, locale)
		if err != nil {
			return PullOutput{}, err
		}
		downloadURL, err := c.waitForExport(ctx, in, locale, jobID)
		if err != nil {
			return PullOutput{}, err
		}
		artifact, err := c.downloadArtifact(ctx, in.APIToken, downloadURL)
		if err != nil {
			return PullOutput{}, err
		}
		files := []ExportedFile{{Name: fmt.Sprintf("%s.json", locale), Data: artifact}}
		if looksLikeZip(artifact) {
			files, err = unzipFiles(artifact)
			if err != nil {
				return PullOutput{}, fmt.Errorf("pull locale %s: unzip: %w", locale, err)
			}
		}
		filesByLocale[locale] = files
	}
	revision := time.Now().UTC().Format(time.RFC3339Nano)
	return PullOutput{FilesByLocale: filesByLocale, Revision: revision}, nil
}

func (c *HTTPClient) PushTranslations(ctx context.Context, in PushInput) (PushOutput, error) {
	jobIDs := make([]string, 0, len(in.Files))
	for _, file := range in.Files {
		jobID, err := c.uploadFile(ctx, in, file)
		if err != nil {
			return PushOutput{}, err
		}
		if strings.TrimSpace(jobID) != "" {
			jobIDs = append(jobIDs, jobID)
		}
	}
	return PushOutput{Revision: time.Now().UTC().Format(time.RFC3339Nano), JobIDs: jobIDs}, nil
}

func (c *HTTPClient) startExport(ctx context.Context, in PullInput, locale string) (string, error) {
	payload := map[string]string{"project_id": in.ProjectID, "locale": locale}
	var resp exportStartResponse
	if err := c.sendJSON(ctx, http.MethodPost, c.baseURL+"/files/export", in.APIToken, payload, &resp); err != nil {
		return "", fmt.Errorf("pull locale %s: start export: %w", locale, err)
	}
	jobID := strings.TrimSpace(resp.JobID)
	if jobID == "" {
		jobID = strings.TrimSpace(resp.ID)
	}
	if jobID == "" {
		return "", fmt.Errorf("pull locale %s: missing export job id", locale)
	}
	return jobID, nil
}

func (c *HTTPClient) waitForExport(ctx context.Context, in PullInput, locale, jobID string) (string, error) {
	for i := 0; i < c.maxPolls; i++ {
		var resp exportStatusResponse
		endpoint := c.baseURL + "/files/export/" + url.PathEscape(jobID)
		if err := c.sendJSON(ctx, http.MethodGet, endpoint, in.APIToken, nil, &resp); err != nil {
			return "", fmt.Errorf("pull locale %s: poll export %s: %w", locale, jobID, err)
		}
		status := strings.ToLower(strings.TrimSpace(resp.Status))
		switch status {
		case "completed", "complete", "success", "succeeded":
			downloadURL := strings.TrimSpace(resp.DownloadURL)
			if downloadURL == "" {
				return "", fmt.Errorf("pull locale %s: export %s missing download URL", locale, jobID)
			}
			return downloadURL, nil
		case "failed", "error", "canceled", "cancelled":
			return "", fmt.Errorf("pull locale %s: export %s failed: %s", locale, jobID, strings.TrimSpace(resp.Message))
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(c.pollInterval):
		}
	}
	return "", fmt.Errorf("pull locale %s: export %s polling exceeded %d attempts", locale, jobID, c.maxPolls)
}

func (c *HTTPClient) downloadArtifact(ctx context.Context, token, downloadURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("download artifact request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download artifact: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("download artifact status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download artifact body: %w", err)
	}
	return body, nil
}

func (c *HTTPClient) uploadFile(ctx context.Context, in PushInput, file UploadFile) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("project_id", in.ProjectID)
	_ = writer.WriteField("locale", file.Locale)
	part, err := writer.CreateFormFile("file", file.Filename)
	if err != nil {
		return "", fmt.Errorf("upload %s: create multipart file: %w", file.Locale, err)
	}
	if _, err := part.Write(file.Data); err != nil {
		return "", fmt.Errorf("upload %s: write multipart file: %w", file.Locale, err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("upload %s: close multipart writer: %w", file.Locale, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/files/upload", &body)
	if err != nil {
		return "", fmt.Errorf("upload %s: build request: %w", file.Locale, err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+in.APIToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload %s: do request: %w", file.Locale, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("upload %s: status %d: %s", file.Locale, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	var out importResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("upload %s: decode response: %w", file.Locale, err)
	}
	jobID := strings.TrimSpace(out.JobID)
	if jobID == "" {
		jobID = strings.TrimSpace(out.ID)
	}
	return jobID, nil
}

func (c *HTTPClient) sendJSON(ctx context.Context, method, endpoint, token string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

type flatRecord struct {
	Key     string `json:"key"`
	Context string `json:"context,omitempty"`
	Value   string `json:"value"`
}

func ParseArtifact(name, locale string, data []byte) ([]storage.Entry, error) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".json" && ext != "" {
		return nil, fmt.Errorf("unsupported artifact type %q", ext)
	}
	trimmedLocale := strings.TrimSpace(locale)
	if trimmedLocale == "" {
		return nil, fmt.Errorf("locale is required")
	}
	var arr []flatRecord
	if err := json.Unmarshal(data, &arr); err == nil {
		out := make([]storage.Entry, 0, len(arr))
		for _, item := range arr {
			out = append(out, storage.Entry{Key: item.Key, Context: item.Context, Locale: trimmedLocale, Value: item.Value})
		}
		return out, nil
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	out := make([]storage.Entry, 0)
	flattenObject(trimmedLocale, "", obj, &out)
	return out, nil
}

func flattenObject(locale, prefix string, obj map[string]any, out *[]storage.Entry) {
	for key, val := range obj {
		nextKey := key
		if prefix != "" {
			nextKey = prefix + "." + key
		}

		switch typed := val.(type) {
		case map[string]any:
			flattenObject(locale, nextKey, typed, out)
		default:
			encoded, ok := flattenValue(typed)
			if !ok {
				continue
			}
			*out = append(*out, storage.Entry{Key: nextKey, Locale: locale, Value: encoded})
		}
	}
}

func flattenValue(v any) (string, bool) {
	switch typed := v.(type) {
	case nil:
		return "", false
	case string:
		return typed, true
	case bool, float64, int, int64, uint64:
		return fmt.Sprint(typed), true
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return "", false
		}
		return string(raw), true
	}
}

func looksLikeZip(data []byte) bool {
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04
}

func unzipFiles(data []byte) ([]ExportedFile, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	out := make([]ExportedFile, 0, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		payload, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			return nil, readErr
		}
		out = append(out, ExportedFile{Name: file.Name, Data: payload})
	}
	return out, nil
}
