package mt

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type deeplRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f deeplRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newDeepLTestClient(t *testing.T, handler http.HandlerFunc) *DeepLClient {
	t.Helper()
	cfg := newTestServer(t, handler)
	cfg.APIKey = "test-key"
	client, err := NewDeepLClient(cfg)
	require.NoError(t, err)
	return client
}

func TestNewDeepLClientRequiresAPIKey(t *testing.T) {
	client, err := NewDeepLClient(Config{BaseURL: DeepLProBaseURL})
	require.Nil(t, client)
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeValidation, typed.Code)
}

func TestNewDeepLClientRequiresBaseURL(t *testing.T) {
	client, err := NewDeepLClient(Config{APIKey: "test-key"})
	require.Nil(t, client)
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeValidation, typed.Code)
}

func TestNewDeepLClientUsesConfiguredBaseURL(t *testing.T) {
	for _, base := range []string{DeepLProBaseURL, DeepLFreeBaseURL} {
		t.Run(base, func(t *testing.T) {
			var gotURL string
			client, err := NewDeepLClient(Config{
				APIKey:  "test-key",
				BaseURL: base,
				HTTPClient: &http.Client{
					Transport: deeplRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
						gotURL = req.URL.String()
						return nil, errors.New("no network in this test")
					}),
				},
			})
			require.NoError(t, err)

			_, _ = client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
			require.True(t, len(gotURL) >= len(base) && gotURL[:len(base)] == base, "expected URL %q to start with %q", gotURL, base)
		})
	}
}

func TestDeepLClientTranslateSuccess(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	var gotForm url.Values
	client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		require.NoError(t, r.ParseForm())
		gotForm = r.PostForm
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"translations":[{"text":"Bonjour"},{"text":"Salut"}]}`))
	})

	resp, err := client.Translate(t.Context(), Request{
		SourceLocale: "en",
		TargetLocale: "fr",
		Sources:      []string{"Hello", "Hi"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"Bonjour", "Salut"}, resp.Translations)

	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, deeplTranslatePath, gotPath)
	require.Equal(t, "DeepL-Auth-Key test-key", gotAuth)
	require.Equal(t, []string{"Hello", "Hi"}, gotForm["text"])
	require.Equal(t, "EN", gotForm.Get("source_lang"))
	require.Equal(t, "FR", gotForm.Get("target_lang"))
}

func TestDeepLClientTranslateRawErrorBodyNotEchoed(t *testing.T) {
	const rawBody = "Internal Server Error: stack trace and sensitive upstream details"

	client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(rawBody))
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeUpstreamUnavailable, typed.Code)
	require.Equal(t, http.StatusInternalServerError, typed.StatusCode)
	require.Equal(t, "DeepL error (500)", typed.Message)
	require.NotContains(t, typed.Message, rawBody)
	require.NotContains(t, err.Error(), rawBody)
}

func TestDeepLClientTranslateAuthFailed(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"message":"Authorization failed"}`))
			})

			_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
			typed, ok := AsError(err)
			require.True(t, ok)
			require.Equal(t, ErrorCodeAuthFailed, typed.Code)
			require.Equal(t, status, typed.StatusCode)
		})
	}
}

func TestDeepLClientTranslateRateLimited(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, deeplStatusQuotaExceeded} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"message":"Too many requests"}`))
			})

			_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
			typed, ok := AsError(err)
			require.True(t, ok)
			require.Equal(t, ErrorCodeRateLimited, typed.Code)
			require.Equal(t, status, typed.StatusCode)
		})
	}
}

func TestDeepLClientTranslateBadRequestMapsToUpstreamError(t *testing.T) {
	messages := []string{
		`{"message":"Value for 'target_lang' not supported."}`,
		`{"message":"invalid_content_type"}`,
	}
	for _, body := range messages {
		t.Run(body, func(t *testing.T) {
			client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(body))
			})

			_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
			typed, ok := AsError(err)
			require.True(t, ok)
			require.Equal(t, ErrorCodeUpstream, typed.Code)
			require.Equal(t, http.StatusBadRequest, typed.StatusCode)
		})
	}
}

func TestDeepLClientTranslateUpstreamUnavailable(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusGatewayTimeout} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"message":"Internal error"}`))
			})

			_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
			typed, ok := AsError(err)
			require.True(t, ok)
			require.Equal(t, ErrorCodeUpstreamUnavailable, typed.Code)
		})
	}
}

func TestDeepLClientTranslateTranslationCountMismatchMapsToUpstreamError(t *testing.T) {
	client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"translations":[{"text":"Bonjour"}]}`))
	})

	_, err := client.Translate(t.Context(), Request{
		SourceLocale: "en",
		TargetLocale: "fr",
		Sources:      []string{"Hello", "Hi"},
	})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeUpstream, typed.Code)
}

func TestDeepLClientTranslateContextCanceled(t *testing.T) {
	client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"translations":[]}`))
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := client.Translate(ctx, Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
	_, ok := AsError(err)
	require.False(t, ok)
}

func TestDeepLClientTranslateContextDeadlineExceeded(t *testing.T) {
	client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"translations":[]}`))
	})

	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()

	_, err := client.Translate(ctx, Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	require.Error(t, err)
	require.True(t, errors.Is(err, context.DeadlineExceeded))
	_, ok := AsError(err)
	require.False(t, ok)
}

func TestDeepLClientTranslateTransportErrorDoesNotLeakAPIKey(t *testing.T) {
	const apiKey = "super-secret-deepl-key-12345"

	client, err := NewDeepLClient(Config{
		BaseURL: DeepLProBaseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Transport: deeplRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("dial tcp: connection refused, headers: " + req.Header.Get("Authorization"))
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
	require.Equal(t, deeplTranslatePath, typed.Path)
	require.NotContains(t, typed.Message, apiKey)
	require.NotContains(t, err.Error(), apiKey)
}

func TestDeepLClientTranslateEmptyInputMakesNoRequest(t *testing.T) {
	client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected HTTP call for empty input")
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: nil})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeValidation, typed.Code)
}

func TestDeepLClientTranslateInvalidJSONResponse(t *testing.T) {
	client := newDeepLTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	})

	_, err := client.Translate(t.Context(), Request{SourceLocale: "en", TargetLocale: "fr", Sources: []string{"hi"}})
	typed, ok := AsError(err)
	require.True(t, ok)
	require.Equal(t, ErrorCodeUpstream, typed.Code)
}

func TestDeepLSourceLanguageCode(t *testing.T) {
	cases := []struct {
		locale string
		want   string
	}{
		{"en", "EN"},
		{"en-US", "EN"},
		{"en-GB", "EN"},
		{"pt-BR", "PT"},
		{"zh-Hant", "ZH"},
		{"zh-Hans", "ZH"},
	}
	for _, tc := range cases {
		t.Run(tc.locale, func(t *testing.T) {
			require.Equal(t, tc.want, deeplSourceLanguageCode(tc.locale))
		})
	}
}

func TestDeepLTargetLanguageCode(t *testing.T) {
	cases := []struct {
		locale string
		want   string
	}{
		{"en", "EN"},
		{"pt", "PT"},
		{"zh", "ZH"},
		{"en-US", "EN-US"},
		{"en-GB", "EN-GB"},
		{"en-AU", "EN"},
		{"pt-BR", "PT-BR"},
		{"pt-PT", "PT-PT"},
		{"zh-Hans", "ZH-HANS"},
		{"zh-Hant", "ZH-HANT"},
		{"zh-Hant-HK", "ZH-HANT"},
		{"fr-FR", "FR-FR"},
		{"fr-CA", "FR-CA"},
		{"fr-BE", "FR"},
		{"de-DE", "DE-DE"},
		{"de-CH", "DE-CH"},
		{"de-AT", "DE"},
		{"es-419", "ES-419"},
		{"es-MX", "ES"},
		{"ja-JP", "JA"},
		{"vi-VN", "VI"},
	}
	for _, tc := range cases {
		t.Run(tc.locale, func(t *testing.T) {
			require.Equal(t, tc.want, deeplTargetLanguageCode(tc.locale))
		})
	}
}
