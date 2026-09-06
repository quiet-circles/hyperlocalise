package mt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/text/language"
)

const (
	defaultGoogleTranslateBaseURL = "https://translation.googleapis.com"
	googleTranslateV2Path         = "/language/translate/v2"
)

// GoogleClient translates text using Cloud Translation v2.
type GoogleClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

var _ Engine = (*GoogleClient)(nil)

// NewGoogleClient creates a Cloud Translation v2 client.
func NewGoogleClient(cfg Config) (*GoogleClient, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, &Error{Code: ErrorCodeValidation, Message: "API key is required"}
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultGoogleTranslateBaseURL
	}
	return &GoogleClient{
		apiKey:     cfg.APIKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: cfg.httpClient(),
	}, nil
}

type googleTranslateRequest struct {
	Q      []string `json:"q"`
	Source string   `json:"source"`
	Target string   `json:"target"`
	Format string   `json:"format"`
}

type googleTranslateResponse struct {
	Data struct {
		Translations []struct {
			TranslatedText string `json:"translatedText"`
		} `json:"translations"`
	} `json:"data"`
}

type googleErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Errors  []struct {
			Domain string `json:"domain"`
			Reason string `json:"reason"`
		} `json:"errors"`
	} `json:"error"`
}

// Translate implements Engine using Cloud Translation v2.
func (c *GoogleClient) Translate(ctx context.Context, req Request) (Response, error) {
	if err := validateRequest(req); err != nil {
		return Response{}, err
	}

	body := googleTranslateRequest{
		Q:      req.Sources,
		Source: googleLanguageCode(req.SourceLocale),
		Target: googleLanguageCode(req.TargetLocale),
		Format: "text",
	}

	var out googleTranslateResponse
	if err := c.request(ctx, http.MethodPost, c.translateURL(), body, &out); err != nil {
		return Response{}, err
	}

	translations := make([]string, len(out.Data.Translations))
	for i, t := range out.Data.Translations {
		translations[i] = html.UnescapeString(t.TranslatedText)
	}
	return Response{Translations: translations}, nil
}

func (c *GoogleClient) translateURL() string {
	values := url.Values{}
	values.Set("key", c.apiKey)
	return c.baseURL + googleTranslateV2Path + "?" + values.Encode()
}

func (c *GoogleClient) request(ctx context.Context, method, requestURL string, body, out any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("mt: marshal google translate request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("mt: build google translate request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return &Error{
			Code:    ErrorCodeUpstreamUnavailable,
			Message: "could not reach Google Cloud Translation",
			Path:    googleTranslateV2Path,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return &Error{
			Code:    ErrorCodeUpstreamUnavailable,
			Message: "could not read Google Cloud Translation response",
			Path:    googleTranslateV2Path,
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return googleHTTPError(resp.StatusCode, googleTranslateV2Path, respBody)
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return &Error{
			Code:       ErrorCodeUpstream,
			Message:    "Google Cloud Translation returned invalid JSON",
			StatusCode: resp.StatusCode,
			Path:       googleTranslateV2Path,
		}
	}
	return nil
}

// Google v2 does not distinguish unsupported languages from other invalid
// requests, so generic 400 responses map to ErrorCodeUpstream.
func googleHTTPError(statusCode int, path string, body []byte) *Error {
	var parsed googleErrorResponse
	_ = json.Unmarshal(body, &parsed)
	reason, domain := firstGoogleError(parsed)
	message := googleErrorMessage(statusCode, parsed, body)

	switch {
	case statusCode == http.StatusBadRequest && reason == "keyInvalid":
		return &Error{Code: ErrorCodeAuthFailed, Message: message, StatusCode: statusCode, Path: path}
	case statusCode == http.StatusBadRequest:
		return &Error{Code: ErrorCodeUpstream, Message: message, StatusCode: statusCode, Path: path}
	case statusCode == http.StatusUnauthorized:
		return &Error{Code: ErrorCodeAuthFailed, Message: message, StatusCode: statusCode, Path: path}
	case statusCode == http.StatusForbidden && domain == "usageLimits" && (reason == "dailyLimitExceeded" || reason == "userRateLimitExceeded"):
		return &Error{Code: ErrorCodeRateLimited, Message: message, StatusCode: statusCode, Path: path}
	case statusCode == http.StatusForbidden:
		return &Error{Code: ErrorCodeAuthFailed, Message: message, StatusCode: statusCode, Path: path}
	case statusCode == http.StatusTooManyRequests:
		return &Error{Code: ErrorCodeRateLimited, Message: message, StatusCode: statusCode, Path: path}
	case statusCode >= 500:
		return &Error{Code: ErrorCodeUpstreamUnavailable, Message: message, StatusCode: statusCode, Path: path}
	default:
		return &Error{Code: ErrorCodeUpstream, Message: message, StatusCode: statusCode, Path: path}
	}
}

func firstGoogleError(parsed googleErrorResponse) (reason, domain string) {
	if len(parsed.Error.Errors) == 0 {
		return "", ""
	}
	return parsed.Error.Errors[0].Reason, parsed.Error.Errors[0].Domain
}

func googleErrorMessage(statusCode int, parsed googleErrorResponse, body []byte) string {
	if parsed.Error.Message != "" {
		return fmt.Sprintf("Google Cloud Translation error (%d): %s", statusCode, parsed.Error.Message)
	}
	snippet := truncateGoogleErrorBody(body, 300)
	if snippet == "" {
		return fmt.Sprintf("Google Cloud Translation error (%d)", statusCode)
	}
	return fmt.Sprintf("Google Cloud Translation error (%d): %s", statusCode, snippet)
}

func truncateGoogleErrorBody(body []byte, max int) string {
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) <= max {
		return trimmed
	}
	return trimmed[:max] + "..."
}

// googleLanguageCode maps BCP 47 locales to Google v2 language codes.
// Chinese uses the resolved script so regional variants map correctly.
func googleLanguageCode(locale string) string {
	tag, err := language.Parse(locale)
	if err != nil {
		return locale
	}
	base, _ := tag.Base()
	if base.String() == "zh" {
		script, _ := tag.Script()
		if script.String() == "Hant" {
			return "zh-TW"
		}
		return "zh-CN"
	}
	return base.String()
}
