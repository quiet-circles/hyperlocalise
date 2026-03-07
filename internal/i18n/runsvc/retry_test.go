package runsvc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "network timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

type nonTimeoutNetError struct{}

func (nonTimeoutNetError) Error() string   { return "connection refused" }
func (nonTimeoutNetError) Timeout() bool   { return false }
func (nonTimeoutNetError) Temporary() bool { return false }

func TestTranslateWithRetryBuildsRequestWithoutContextMemory(t *testing.T) {
	svc := &Service{}
	var got translator.Request
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		got = req
		return "Bonjour", nil
	}

	task := Task{
		EntryKey:      "checkout.title",
		SourceText:    "Checkout",
		TargetLocale:  "fr",
		Provider:      "openai",
		Model:         "gpt-4.1",
		SystemPrompt:  "system",
		UserPrompt:    "user",
		ContextMemory: "   ",
	}

	gotText, err := svc.translateWithRetry(context.Background(), task)
	if err != nil {
		t.Fatalf("translateWithRetry returned error: %v", err)
	}
	if gotText != "Bonjour" {
		t.Fatalf("translateWithRetry result = %q, want Bonjour", gotText)
	}
	if got.Source != "Checkout" {
		t.Fatalf("request source = %q, want Checkout", got.Source)
	}
	if got.TargetLanguage != "fr" {
		t.Fatalf("request target = %q, want fr", got.TargetLanguage)
	}
	if got.Context != "checkout.title" {
		t.Fatalf("request context = %q, want checkout.title", got.Context)
	}
	if got.ModelProvider != "openai" || got.Model != "gpt-4.1" {
		t.Fatalf("request model provider/model mismatch: %+v", got)
	}
	if got.SystemPrompt != "system" || got.UserPrompt != "user" {
		t.Fatalf("request prompts mismatch: %+v", got)
	}
}

func TestTranslateWithRetryBuildsRequestWithContextMemory(t *testing.T) {
	svc := &Service{}
	var got translator.Request
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		got = req
		return "Bonjour", nil
	}

	task := Task{
		EntryKey:      "checkout.title",
		SourceText:    "Checkout",
		TargetLocale:  "fr",
		ContextMemory: "Shared context",
	}

	_, err := svc.translateWithRetry(context.Background(), task)
	if err != nil {
		t.Fatalf("translateWithRetry returned error: %v", err)
	}

	wantContext := "checkout.title\n\nShared memory:\nShared context"
	if got.Context != wantContext {
		t.Fatalf("request context = %q, want %q", got.Context, wantContext)
	}
}

func TestTranslateWithRetryBuildsRequestWithEmptyEntryKey(t *testing.T) {
	svc := &Service{}
	var got translator.Request
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		got = req
		return "ok", nil
	}

	task := Task{
		EntryKey:      "",
		SourceText:    "Hello",
		TargetLocale:  "fr",
		ContextMemory: "",
	}

	_, err := svc.translateWithRetry(context.Background(), task)
	if err != nil {
		t.Fatalf("translateWithRetry returned error: %v", err)
	}
	if got.Context != "" {
		t.Fatalf("request context = %q, want empty string", got.Context)
	}
}

func TestTranslateWithRetryBuildsRequestWithEmptyEntryKeyAndContextMemory(t *testing.T) {
	svc := &Service{}
	var got translator.Request
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		got = req
		return "ok", nil
	}

	task := Task{
		EntryKey:      "",
		SourceText:    "Hello",
		TargetLocale:  "fr",
		ContextMemory: "Shared context",
	}

	_, err := svc.translateWithRetry(context.Background(), task)
	if err != nil {
		t.Fatalf("translateWithRetry returned error: %v", err)
	}
	wantContext := "\n\nShared memory:\nShared context"
	if got.Context != wantContext {
		t.Fatalf("request context = %q, want %q", got.Context, wantContext)
	}
}

func TestTranslateRequestWithRetrySucceedsWithoutRetry(t *testing.T) {
	svc := &Service{}
	attempts := 0
	svc.translate = func(_ context.Context, _ translator.Request) (string, error) {
		attempts++
		return "ok", nil
	}

	originalSleep := sleepWithContext
	t.Cleanup(func() { sleepWithContext = originalSleep })
	sleepWithContext = func(_ context.Context, _ time.Duration) error {
		t.Fatal("sleepWithContext should not be called when first attempt succeeds")
		return nil
	}

	got, err := svc.translateRequestWithRetry(context.Background(), translator.Request{Source: "A"})
	if err != nil {
		t.Fatalf("translateRequestWithRetry returned error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("translateRequestWithRetry result = %q, want ok", got)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestTranslateRequestWithRetryRetriesThenSucceeds(t *testing.T) {
	svc := &Service{}
	attempts := 0
	svc.translate = func(_ context.Context, _ translator.Request) (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("status code 429")
		}
		return "ok", nil
	}

	var gotDelays []time.Duration
	originalSleep := sleepWithContext
	t.Cleanup(func() { sleepWithContext = originalSleep })
	sleepWithContext = func(_ context.Context, d time.Duration) error {
		gotDelays = append(gotDelays, d)
		return nil
	}

	got, err := svc.translateRequestWithRetry(context.Background(), translator.Request{Source: "A"})
	if err != nil {
		t.Fatalf("translateRequestWithRetry returned error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("translateRequestWithRetry result = %q, want ok", got)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	wantDelays := []time.Duration{250 * time.Millisecond, 500 * time.Millisecond}
	if len(gotDelays) != len(wantDelays) {
		t.Fatalf("delays = %v, want %v", gotDelays, wantDelays)
	}
	for i := range wantDelays {
		if gotDelays[i] != wantDelays[i] {
			t.Fatalf("delays[%d] = %v, want %v", i, gotDelays[i], wantDelays[i])
		}
	}
}

func TestTranslateRequestWithRetryStopsAtNonRetryableError(t *testing.T) {
	sentinel := errors.New("invalid request")
	svc := &Service{}
	attempts := 0
	svc.translate = func(_ context.Context, _ translator.Request) (string, error) {
		attempts++
		return "", sentinel
	}

	originalSleep := sleepWithContext
	t.Cleanup(func() { sleepWithContext = originalSleep })
	sleepWithContext = func(_ context.Context, _ time.Duration) error {
		t.Fatal("sleepWithContext should not be called for non-retryable errors")
		return nil
	}

	_, err := svc.translateRequestWithRetry(context.Background(), translator.Request{Source: "A"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("error does not wrap original error: %v", err)
	}
	if !strings.Contains(err.Error(), "translation failed after 1 attempts") {
		t.Fatalf("error = %q, want attempts context", err.Error())
	}
}

func TestTranslateRequestWithRetryStopsAfterMaxAttempts(t *testing.T) {
	sentinel := errors.New("status code 503")
	svc := &Service{}
	attempts := 0
	svc.translate = func(_ context.Context, _ translator.Request) (string, error) {
		attempts++
		return "", sentinel
	}

	var sleepCalls int
	originalSleep := sleepWithContext
	t.Cleanup(func() { sleepWithContext = originalSleep })
	sleepWithContext = func(_ context.Context, _ time.Duration) error {
		sleepCalls++
		return nil
	}

	_, err := svc.translateRequestWithRetry(context.Background(), translator.Request{Source: "A"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != translationRetryMaxAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, translationRetryMaxAttempts)
	}
	if sleepCalls != translationRetryMaxAttempts-1 {
		t.Fatalf("sleepCalls = %d, want %d", sleepCalls, translationRetryMaxAttempts-1)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("error does not wrap original error: %v", err)
	}
	if !strings.Contains(err.Error(), "translation failed after 5 attempts") {
		t.Fatalf("error = %q, want max-attempt context", err.Error())
	}
}

func TestTranslateRequestWithRetryReturnsWhenBackoffInterrupted(t *testing.T) {
	sentinel := errors.New("status code 429")
	svc := &Service{}
	attempts := 0
	svc.translate = func(_ context.Context, _ translator.Request) (string, error) {
		attempts++
		return "", sentinel
	}

	originalSleep := sleepWithContext
	t.Cleanup(func() { sleepWithContext = originalSleep })
	sleepWithContext = func(_ context.Context, _ time.Duration) error {
		return context.Canceled
	}

	_, err := svc.translateRequestWithRetry(context.Background(), translator.Request{Source: "A"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1 (stop when backoff wait fails)", attempts)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error does not wrap context.Canceled: %v", err)
	}
	if !strings.Contains(err.Error(), "translation retry wait interrupted") {
		t.Fatalf("error = %q, want wait interruption context", err.Error())
	}
}

func TestIsRetryableTranslateError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "context canceled", err: context.Canceled, want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: true},
		{name: "net timeout", err: timeoutNetError{}, want: true},
		{name: "net non-timeout", err: nonTimeoutNetError{}, want: false},
		{name: "rate limit literal", err: errors.New("rate limit exceeded"), want: true},
		{name: "rate limit token", err: errors.New("HTTP 429 from provider"), want: true},
		{name: "rate limit phrase", err: errors.New("too many requests"), want: true},
		{name: "timeout word", err: errors.New("request timeout"), want: true},
		{name: "timed out phrase", err: errors.New("upstream timed out"), want: true},
		{name: "status 500", err: errors.New("status code 500"), want: true},
		{name: "status 502", err: errors.New("status code 502"), want: true},
		{name: "status 503", err: errors.New("status code 503"), want: true},
		{name: "status 504", err: errors.New("status code 504"), want: true},
		{name: "service unavailable", err: errors.New("service unavailable"), want: true},
		{name: "temporarily unavailable", err: errors.New("temporarily unavailable"), want: true},
		{name: "wrapped retryable error", err: fmt.Errorf("provider call: %w", errors.New("status code 503")), want: true},
		{name: "wrapped non-retryable error", err: fmt.Errorf("provider call: %w", errors.New("validation failed")), want: false},
		{name: "other", err: errors.New("validation failed"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableTranslateError(tt.err)
			if got != tt.want {
				t.Fatalf("isRetryableTranslateError(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestTranslationRetryDelay(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 0, want: 250 * time.Millisecond},
		{attempt: 1, want: 500 * time.Millisecond},
		{attempt: 2, want: 1 * time.Second},
		{attempt: 3, want: 2 * time.Second},
		{attempt: 4, want: 4 * time.Second},
		{attempt: 5, want: 5 * time.Second},
		{attempt: 8, want: 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			got := translationRetryDelay(tt.attempt)
			if got != tt.want {
				t.Fatalf("translationRetryDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

var (
	_ net.Error = timeoutNetError{}
	_ net.Error = nonTimeoutNetError{}
)
