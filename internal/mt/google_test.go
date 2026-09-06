package mt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newGoogleTestClient(t *testing.T, handler http.HandlerFunc) *GoogleClient {
	t.Helper()
	cfg := newTestServer(t, handler)
	cfg.APIKey = "test-key"
	client, err := NewGoogleClient(cfg)
	require.NoError(t, err)
	return client
}

func TestGoogleClientTranslateSuccess(t *testing.T) {
	var gotBody googleTranslateRequest
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, googleTranslateV2Path, r.URL.Path)
		require.Equal(t, "test-key", r.URL.Query().Get("key"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"translations":[{"translatedText":"Bonjour"},{"translatedText":"Salut"}]}}`))
	})

	resp, err := client.Translate(t.Context(), Request{
		SourceLocale: "en",
		TargetLocale: "fr",
		Sources:      []string{"Hello", "Hi"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"Bonjour", "Salut"}, resp.Translations)

	require.Equal(t, []string{"Hello", "Hi"}, gotBody.Q)
	require.Equal(t, "en", gotBody.Source)
	require.Equal(t, "fr", gotBody.Target)
	require.Equal(t, "text", gotBody.Format)
}

func TestGoogleClientTranslateMapsChineseScriptVariants(t *testing.T) {
	var gotBody googleTranslateRequest
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"translations":[{"translatedText":"x"}]}}`))
	})

	_, err := client.Translate(t.Context(), Request{
		SourceLocale: "zh-Hans",
		TargetLocale: "zh-Hant-HK",
		Sources:      []string{"你好"},
	})
	require.NoError(t, err)
	require.Equal(t, "zh-CN", gotBody.Source)
	require.Equal(t, "zh-TW", gotBody.Target)
}

func TestGoogleClientTranslateAuthFailed401(t *testing.T) {
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":401,"message":"Request had invalid authentication credentials."}}`))
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeAuthFailed, typed.Code)
	require.Equal(t, http.StatusUnauthorized, typed.StatusCode)
}

func TestGoogleClientTranslateAuthFailed403(t *testing.T) {
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"errors":[{"domain":"global","reason":"forbidden","message":"The caller does not have permission"}],"message":"The caller does not have permission"}}`))
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeAuthFailed, typed.Code)
}

func TestGoogleClientTranslateAuthFailedInvalidAPIKey(t *testing.T) {
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":400,"errors":[{"domain":"global","reason":"keyInvalid","message":"API key not valid. Please pass a valid API key."}],"message":"API key not valid. Please pass a valid API key."}}`))
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeAuthFailed, typed.Code)
}

func TestGoogleClientTranslateRateLimited429(t *testing.T) {
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":429,"message":"Quota exceeded"}}`))
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeRateLimited, typed.Code)
}

func TestGoogleClientTranslateRateLimited403Quota(t *testing.T) {
	for _, reason := range []string{"dailyLimitExceeded", "userRateLimitExceeded"} {
		t.Run(reason, func(t *testing.T) {
			client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				body, err := json.Marshal(map[string]any{
					"error": map[string]any{
						"code": 403,
						"errors": []map[string]any{
							{"domain": "usageLimits", "reason": reason, "message": "Rate Limit Exceeded"},
						},
						"message": "Rate Limit Exceeded",
					},
				})
				require.NoError(t, err)
				_, _ = w.Write(body)
			})

			_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
			typed, ok := AsError(err)
			require.True(t, ok)
			require.Equal(t, ErrorCodeRateLimited, typed.Code)
		})
	}
}

// Google v2 does not distinguish unsupported languages from other invalid
// requests, so generic 400s map to ErrorCodeUpstream.
func TestGoogleClientTranslateUnsupportedLanguageMapsToUpstreamError(t *testing.T) {
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":400,"errors":[{"domain":"global","reason":"invalid","message":"Invalid Value"}],"message":"Invalid Value"}}`))
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "de", Sources: []string{"hi"}})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeUpstream, typed.Code)
}

func TestGoogleClientTranslateTransportErrorDoesNotLeakAPIKey(t *testing.T) {
	const apiKey = "super-secret-api-key-12345"

	client, err := NewGoogleClient(Config{
		BaseURL: defaultGoogleTranslateBaseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("Post %q: connection refused", req.URL.String())
			}),
		},
	})
	require.NoError(t, err)

	_, err = client.Translate(t.Context(), Request{
		SourceLocale: "en",
		TargetLocale: "fr",
		Sources:      []string{"hi"},
	})
	require.Error(t, err)

	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeUpstreamUnavailable, typed.Code)
	require.Equal(t, googleTranslateV2Path, typed.Path)
	require.NotContains(t, typed.Message, apiKey)
	require.NotContains(t, err.Error(), apiKey)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGoogleClientTranslateUpstreamUnavailable(t *testing.T) {
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"code":503,"message":"Service Unavailable"}}`))
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeUpstreamUnavailable, typed.Code)
}

func TestGoogleClientTranslateInvalidJSONResponse(t *testing.T) {
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeUpstream, typed.Code)
}

func TestGoogleClientTranslateEmptyInputMakesNoRequest(t *testing.T) {
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected HTTP call for empty input")
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: nil})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeValidation, typed.Code)
}

func TestGoogleClientTranslateContextDeadlineExceeded(t *testing.T) {
	client := newGoogleTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"translations":[]}}`))
	})

	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()

	_, err := client.Translate(ctx, Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	require.Error(t, err)
	require.True(t, errors.Is(err, context.DeadlineExceeded))
	_, ok := AsError(err)
	require.False(t, ok)
}

func TestNewGoogleClientRequiresAPIKey(t *testing.T) {
	client, err := NewGoogleClient(Config{})
	require.Nil(t, client)
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeValidation, typed.Code)
}

func TestGoogleLanguageCode(t *testing.T) {
	cases := []struct {
		locale string
		want   string
	}{
		{"en", "en"},
		{"pt-BR", "pt"},
		{"en-GB", "en"},
		{"zh", "zh-CN"},
		{"zh-CN", "zh-CN"},
		{"zh-Hans", "zh-CN"},
		{"zh-Hans-CN", "zh-CN"},
		{"zh-TW", "zh-TW"},
		{"zh-HK", "zh-TW"},
		{"zh-Hant", "zh-TW"},
		{"zh-Hant-HK", "zh-TW"},
		{"zh-Hant-TW", "zh-TW"},
		{"he", "he"},
		{"iw", "he"},
		{"id", "id"},
		{"in", "id"},
	}
	for _, tc := range cases {
		t.Run(tc.locale, func(t *testing.T) {
			require.Equal(t, tc.want, googleLanguageCode(tc.locale))
		})
	}
}
