package smartling

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	projectsAPIBaseURL = "https://api.smartling.com/projects-api/v2"
	accountsAPIBaseURL = "https://api.smartling.com/accounts-api/v2"
)

var (
	filesListPageLimit    = 100
	filesListMaxPages     = 1000
	projectsListPageLimit = 100
	projectsListMaxPages  = 1000
)

// FileListInput lists source files in a Smartling project.
type FileListInput struct {
	ProjectID string
	URIMask   string
}

// FileListItem is one file from GET /files-api/v2/projects/{id}/files/list.
type FileListItem struct {
	FileURI      string `json:"fileUri"`
	FileType     string `json:"fileType"`
	LastUploaded string `json:"lastUploaded"`
	Created      string `json:"created,omitempty"`
}

// FileStatusInput fetches translation status for one file URI.
type FileStatusInput struct {
	ProjectID string
	FileURI   string
}

// FileStatus is GET /files-api/v2/projects/{id}/file/status.
type FileStatus struct {
	FileURI          string             `json:"fileUri"`
	FileType         string             `json:"fileType"`
	LastUploaded     string             `json:"lastUploaded"`
	TotalStringCount int                `json:"totalStringCount"`
	TotalWordCount   int                `json:"totalWordCount"`
	Items            []FileStatusLocale `json:"items"`
}

// FileStatusLocale is per-locale progress on a file.
// Live Smartling status often leaves FileStatus.TotalStringCount at 0.
// Per-locale work uses authorizedStringCount / authorizedWordCount as the denominator.
type FileStatusLocale struct {
	LocaleID              string `json:"localeId"`
	CompletedStringCount  int    `json:"completedStringCount"`
	CompletedWordCount    int    `json:"completedWordCount"`
	AuthorizedStringCount int    `json:"authorizedStringCount,omitempty"`
	AuthorizedWordCount   int    `json:"authorizedWordCount,omitempty"`
}

// LocaleStatusPercent is completed strings over authorized strings for one locale.
// Zero authorized strings (or a missing field) returns 0 so callers never divide by zero.
func LocaleStatusPercent(item FileStatusLocale) int {
	if item.AuthorizedStringCount <= 0 {
		return 0
	}
	return (item.CompletedStringCount * 100) / item.AuthorizedStringCount
}

// LocaleListInput lists source and target locales for a project.
type LocaleListInput struct {
	ProjectID string
}

// LocaleListItem is a source or target locale from project details.
type LocaleListItem struct {
	LocaleID    string `json:"localeId"`
	Description string `json:"description,omitempty"`
	Source      bool   `json:"source"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

// ProjectListInput lists projects on a Smartling account.
type ProjectListInput struct {
	AccountUID string
}

// ProjectListItem is one account project.
type ProjectListItem struct {
	ProjectID      string `json:"projectId"`
	ProjectName    string `json:"projectName"`
	SourceLocaleID string `json:"sourceLocaleId,omitempty"`
	Archived       bool   `json:"archived,omitempty"`
}

type filesListPayload struct {
	TotalCount int            `json:"totalCount"`
	Items      []FileListItem `json:"items"`
}

type projectDetailsPayload struct {
	ProjectID               string `json:"projectId"`
	ProjectName             string `json:"projectName"`
	SourceLocaleID          string `json:"sourceLocaleId"`
	SourceLocaleDescription string `json:"sourceLocaleDescription"`
	TargetLocales           []struct {
		LocaleID    string `json:"localeId"`
		Description string `json:"description"`
		Enabled     *bool  `json:"enabled"`
	} `json:"targetLocales"`
}

type projectsListPayload struct {
	TotalCount int               `json:"totalCount"`
	Items      []ProjectListItem `json:"items"`
}

type smartlingDataEnvelope struct {
	Response struct {
		Code string          `json:"code"`
		Data json.RawMessage `json:"data"`
	} `json:"response"`
	Data json.RawMessage `json:"data"`
}

func (e smartlingDataEnvelope) code() string {
	return strings.TrimSpace(e.Response.Code)
}

func (e smartlingDataEnvelope) data() json.RawMessage {
	if len(e.Response.Data) > 0 && string(e.Response.Data) != "null" {
		return e.Response.Data
	}
	return e.Data
}

func decodeSmartlingData(envelope smartlingDataEnvelope, action string, out any) error {
	code := envelope.code()
	if code != "" && !strings.EqualFold(code, "SUCCESS") {
		return fmt.Errorf("%s: unexpected response code %s", action, code)
	}
	raw := envelope.data()
	if len(raw) == 0 || string(raw) == "null" {
		return fmt.Errorf("%s: empty response data", action)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s: decode response data: %w", action, err)
	}
	return nil
}

// ListFiles pages GET /files-api/v2/projects/{id}/files/list with limit=100 and orderBy=fileUri.
func (c *HTTPClient) ListFiles(ctx context.Context, in FileListInput) ([]FileListItem, error) {
	if strings.TrimSpace(in.ProjectID) == "" {
		return nil, fmt.Errorf("smartling files list: project id is required")
	}

	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	limit := filesListPageLimit
	if limit <= 0 {
		limit = 100
	}
	maxPages := filesListMaxPages
	if maxPages <= 0 {
		maxPages = 1000
	}

	var all []FileListItem
	offset := 0
	for page := 1; ; page++ {
		if page > maxPages {
			return nil, fmt.Errorf("smartling files list: exceeded maximum page count")
		}
		params := url.Values{}
		params.Set("limit", fmt.Sprintf("%d", limit))
		params.Set("offset", fmt.Sprintf("%d", offset))
		params.Set("orderBy", "fileUri")
		if mask := strings.TrimSpace(in.URIMask); mask != "" {
			params.Set("uriMask", mask)
		}
		endpoint := fmt.Sprintf("%s/projects/%s/files/list?%s", c.filesBaseURL, url.PathEscape(strings.TrimSpace(in.ProjectID)), params.Encode())

		var envelope smartlingDataEnvelope
		if err := c.getJSON(ctx, endpoint, token, &envelope); err != nil {
			return nil, fmt.Errorf("smartling files list: %w", err)
		}
		var payload filesListPayload
		if err := decodeSmartlingData(envelope, "smartling files list", &payload); err != nil {
			return nil, err
		}
		if len(payload.Items) == 0 {
			break
		}
		all = append(all, payload.Items...)
		if len(payload.Items) < limit {
			break
		}
		if payload.TotalCount > 0 && len(all) >= payload.TotalCount {
			break
		}
		next := offset + limit
		if next <= offset {
			return nil, fmt.Errorf("smartling files list: pagination offset did not advance")
		}
		offset = next
	}
	return all, nil
}

// GetFileStatus fetches GET /files-api/v2/projects/{id}/file/status for one URI.
func (c *HTTPClient) GetFileStatus(ctx context.Context, in FileStatusInput) (FileStatus, error) {
	if strings.TrimSpace(in.ProjectID) == "" {
		return FileStatus{}, fmt.Errorf("smartling files status: project id is required")
	}
	if strings.TrimSpace(in.FileURI) == "" {
		return FileStatus{}, fmt.Errorf("smartling files status: file uri is required")
	}

	token, err := c.accessToken(ctx)
	if err != nil {
		return FileStatus{}, err
	}

	params := url.Values{}
	params.Set("fileUri", strings.TrimSpace(in.FileURI))
	endpoint := fmt.Sprintf("%s/projects/%s/file/status?%s", c.filesBaseURL, url.PathEscape(strings.TrimSpace(in.ProjectID)), params.Encode())

	var envelope smartlingDataEnvelope
	if err := c.getJSON(ctx, endpoint, token, &envelope); err != nil {
		return FileStatus{}, fmt.Errorf("smartling files status: %w", err)
	}
	var payload FileStatus
	if err := decodeSmartlingData(envelope, "smartling files status", &payload); err != nil {
		return FileStatus{}, err
	}
	return payload, nil
}

// ListLocales returns the project source locale plus target locales.
func (c *HTTPClient) ListLocales(ctx context.Context, in LocaleListInput) ([]LocaleListItem, error) {
	if strings.TrimSpace(in.ProjectID) == "" {
		return nil, fmt.Errorf("smartling locales list: project id is required")
	}

	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/projects/%s", c.projectsBaseURL, url.PathEscape(strings.TrimSpace(in.ProjectID)))
	var envelope smartlingDataEnvelope
	if err := c.getJSON(ctx, endpoint, token, &envelope); err != nil {
		return nil, fmt.Errorf("smartling locales list: %w", err)
	}
	var payload projectDetailsPayload
	if err := decodeSmartlingData(envelope, "smartling locales list", &payload); err != nil {
		return nil, err
	}

	var out []LocaleListItem
	sourceID := strings.TrimSpace(payload.SourceLocaleID)
	if sourceID != "" {
		out = append(out, LocaleListItem{
			LocaleID:    sourceID,
			Description: strings.TrimSpace(payload.SourceLocaleDescription),
			Source:      true,
		})
	}
	for _, locale := range payload.TargetLocales {
		id := strings.TrimSpace(locale.LocaleID)
		if id == "" || id == sourceID {
			continue
		}
		out = append(out, LocaleListItem{
			LocaleID:    id,
			Description: strings.TrimSpace(locale.Description),
			Source:      false,
			Enabled:     locale.Enabled,
		})
	}
	return out, nil
}

// ListProjects pages GET /accounts-api/v2/accounts/{accountUid}/projects.
func (c *HTTPClient) ListProjects(ctx context.Context, in ProjectListInput) ([]ProjectListItem, error) {
	if strings.TrimSpace(in.AccountUID) == "" {
		return nil, fmt.Errorf("smartling projects list: account uid is required")
	}

	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	limit := projectsListPageLimit
	if limit <= 0 {
		limit = 100
	}
	maxPages := projectsListMaxPages
	if maxPages <= 0 {
		maxPages = 1000
	}

	var all []ProjectListItem
	offset := 0
	for page := 1; ; page++ {
		if page > maxPages {
			return nil, fmt.Errorf("smartling projects list: exceeded maximum page count")
		}
		params := url.Values{}
		params.Set("limit", fmt.Sprintf("%d", limit))
		params.Set("offset", fmt.Sprintf("%d", offset))
		endpoint := fmt.Sprintf("%s/accounts/%s/projects?%s", c.accountsBaseURL, url.PathEscape(strings.TrimSpace(in.AccountUID)), params.Encode())

		var envelope smartlingDataEnvelope
		if err := c.getJSON(ctx, endpoint, token, &envelope); err != nil {
			return nil, fmt.Errorf("smartling projects list: %w", err)
		}
		var payload projectsListPayload
		if err := decodeSmartlingData(envelope, "smartling projects list", &payload); err != nil {
			return nil, err
		}
		if len(payload.Items) == 0 {
			break
		}
		all = append(all, payload.Items...)
		if len(payload.Items) < limit {
			break
		}
		if payload.TotalCount > 0 && len(all) >= payload.TotalCount {
			break
		}
		next := offset + limit
		if next <= offset {
			return nil, fmt.Errorf("smartling projects list: pagination offset did not advance")
		}
		offset = next
	}
	return all, nil
}
