package translator

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestResponseTextFromBedrock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		want    string
		wantErr bool
	}{
		{
			name: "single text block",
			body: `{
				"output": {
					"message": {
						"content": [
							{"text": "bonjour"}
						]
					}
				},
				"usage": {
					"inputTokens": 11,
					"outputTokens": 7,
					"totalTokens": 18
				}
			}`,
			want: "bonjour",
		},
		{
			name:    "invalid json",
			body:    `{`,
			wantErr: true,
		},
		{
			name:    "empty output",
			body:    `{"output":{"message":{"content":[]}}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, _, err := responseTextFromBedrock([]byte(tc.body))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("responseTextFromBedrock error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("responseTextFromBedrock = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSignBedrockRequest(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest(http.MethodPost, "https://bedrock-runtime.us-east-1.amazonaws.com/model/foo/converse", strings.NewReader(`{"x":"y"}`))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	err = signBedrockRequest(
		req,
		[]byte(`{"x":"y"}`),
		"us-east-1",
		"test-access-key",
		"test-secret-key",
		"test-session-token",
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("signBedrockRequest: %v", err)
	}

	if got := req.Header.Get("Authorization"); got == "" {
		t.Fatalf("missing Authorization header")
	}

	if got := req.Header.Get("X-Amz-Date"); got != "20260102T030405Z" {
		t.Fatalf("unexpected x-amz-date: %q", got)
	}

	if got := req.Header.Get("X-Amz-Security-Token"); got != "test-session-token" {
		t.Fatalf("unexpected x-amz-security-token: %q", got)
	}
}
