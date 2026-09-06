package mt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/text/language"
)

// DeepLProBaseURL and DeepLFreeBaseURL are DeepL's API hosts.
const (
	DeepLProBaseURL  = "https://api.deepl.com"
	DeepLFreeBaseURL = "https://api-free.deepl.com"
)

const deeplTranslatePath = "/v2/translate"

const deeplStatusQuotaExceeded = 456

type DeepLClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

var _ Engine = (*DeepLClient)(nil)

// NewDeepLClient creates a DeepL API client.
// Config.BaseURL must select the Pro or Free API host.
func NewDeepLClient(cfg Config) (*DeepLClient, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, &Error{Code: ErrorCodeValidation, Message: "API key is required"}
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, &Error{Code: ErrorCodeValidation, Message: "base URL is required"}
	}
	return &DeepLClient{
		apiKey:     cfg.APIKey,
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: cfg.httpClient(),
	}, nil
}

type deeplTranslateResponse struct {
	Translations []struct {
		Text string `json:"text"`
	} `json:"translations"`
}

type deeplErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (c *DeepLClient) Translate(ctx context.Context, req Request) (Response, error) {
	if err := validateRequest(req); err != nil {
		return Response{}, err
	}

	form := url.Values{}
	for _, s := range req.Sources {
		form.Add("text", s)
	}
	form.Set("source_lang", deeplSourceLanguageCode(req.SourceLocale))
	form.Set("target_lang", deeplTargetLanguageCode(req.TargetLocale))

	var out deeplTranslateResponse
	if err := c.request(ctx, http.MethodPost, c.baseURL+deeplTranslatePath, form, &out); err != nil {
		return Response{}, err
	}

	if len(out.Translations) != len(req.Sources) {
		return Response{}, &Error{
			Code:    ErrorCodeUpstream,
			Message: fmt.Sprintf("DeepL returned %d translations for %d inputs", len(out.Translations), len(req.Sources)),
			Path:    deeplTranslatePath,
		}
	}

	translations := make([]string, len(out.Translations))
	for i, t := range out.Translations {
		translations[i] = t.Text
	}
	return Response{Translations: translations}, nil
}

func (c *DeepLClient) request(ctx context.Context, method, requestURL string, form url.Values, out any) error {
	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("mt: build deepl translate request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Authorization", "DeepL-Auth-Key "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return &Error{
			Code:    ErrorCodeUpstreamUnavailable,
			Message: "could not reach DeepL",
			Path:    deeplTranslatePath,
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
			Message: "could not read DeepL response",
			Path:    deeplTranslatePath,
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return deeplHTTPError(resp.StatusCode, deeplTranslatePath, respBody)
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return &Error{
			Code:       ErrorCodeUpstream,
			Message:    "DeepL returned invalid JSON",
			StatusCode: resp.StatusCode,
			Path:       deeplTranslatePath,
		}
	}
	return nil
}

// DeepL has no stable machine-readable unsupported-language signal,
// so generic 400 responses map to ErrorCodeUpstream.
func deeplHTTPError(statusCode int, path string, body []byte) *Error {
	var parsed deeplErrorResponse
	_ = json.Unmarshal(body, &parsed)
	message := deeplErrorMessage(statusCode, parsed)

	switch {
	case statusCode == http.StatusBadRequest:
		return &Error{Code: ErrorCodeUpstream, Message: message, StatusCode: statusCode, Path: path}
	case statusCode == http.StatusUnauthorized, statusCode == http.StatusForbidden:
		return &Error{Code: ErrorCodeAuthFailed, Message: message, StatusCode: statusCode, Path: path}
	case statusCode == http.StatusTooManyRequests, statusCode == deeplStatusQuotaExceeded:
		return &Error{Code: ErrorCodeRateLimited, Message: message, StatusCode: statusCode, Path: path}
	case statusCode >= 500:
		return &Error{Code: ErrorCodeUpstreamUnavailable, Message: message, StatusCode: statusCode, Path: path}
	default:
		return &Error{Code: ErrorCodeUpstream, Message: message, StatusCode: statusCode, Path: path}
	}
}

func deeplErrorMessage(statusCode int, parsed deeplErrorResponse) string {
	if parsed.Message != "" {
		return fmt.Sprintf("DeepL error (%d): %s", statusCode, parsed.Message)
	}
	return fmt.Sprintf("DeepL error (%d)", statusCode)
}

// deeplSourceLanguageCode drops region and script subtags for DeepL source_lang.
func deeplSourceLanguageCode(locale string) string {
	tag, err := language.Parse(locale)
	if err != nil {
		return locale
	}
	base, _ := tag.Base()
	return strings.ToUpper(base.String())
}

// deeplTargetLanguageCode preserves only explicitly specified target subtags.
// x/text may infer region/script values unless confidence is Exact.
func deeplTargetLanguageCode(locale string) string {
	tag, err := language.Parse(locale)
	if err != nil {
		return locale
	}
	base, _ := tag.Base()
	code := strings.ToUpper(base.String())
	if script, conf := tag.Script(); conf == language.Exact {
		return code + "-" + strings.ToUpper(script.String())
	}
	if region, conf := tag.Region(); conf == language.Exact {
		return code + "-" + region.String()
	}
	return code
}
