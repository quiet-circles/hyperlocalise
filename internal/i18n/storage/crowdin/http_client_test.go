package crowdin

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/crowdin/crowdin-api-client-go/crowdin/model"
)

func TestParseProjectID(t *testing.T) {
	if got, err := parseProjectID("123"); err != nil || got != 123 {
		t.Fatalf("parseProjectID valid failed: got=%d err=%v", got, err)
	}
	if _, err := parseProjectID("abc"); err == nil {
		t.Fatalf("expected parseProjectID error for non-numeric value")
	}
	if _, err := parseProjectID("0"); err == nil {
		t.Fatalf("expected parseProjectID error for zero value")
	}
}

func TestIndexSourceStringMarksAmbiguousMapping(t *testing.T) {
	byID := make(map[int]sourceStringMeta)
	byKey := make(map[sourceStringKey]int)

	indexSourceString(byID, byKey, &model.SourceString{
		ID:         1,
		Identifier: "hello",
		Context:    "home",
	})
	indexSourceString(byID, byKey, &model.SourceString{
		ID:         2,
		Identifier: "hello",
		Context:    "home",
	})

	if got := byKey[sourceStringKey{key: "hello", context: "home"}]; got != -1 {
		t.Fatalf("expected ambiguous key mapping to -1, got %d", got)
	}
	if got := len(byID); got != 2 {
		t.Fatalf("expected byID to retain both strings, got %d", got)
	}
}

func TestIsRetryableUpsertError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "429",
			err: &model.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusTooManyRequests},
			},
			want: true,
		},
		{
			name: "500",
			err: &model.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusInternalServerError},
			},
			want: true,
		},
		{
			name: "400",
			err: &model.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusBadRequest},
			},
			want: false,
		},
		{
			name: "network",
			err:  &net.DNSError{IsTimeout: true},
			want: true,
		},
		{
			name: "other",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryableUpsertError(tc.err); got != tc.want {
				t.Fatalf("isRetryableUpsertError() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRetryDelayPrefersRetryAfterHeader(t *testing.T) {
	err := &model.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Retry-After": []string{"2"}},
		},
	}

	if got := retryDelay(0, err); got != 2*time.Second {
		t.Fatalf("retryDelay() = %s, want 2s", got)
	}
}
