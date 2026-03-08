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
	if got.ModelProvider != "openai" || got.Model != "gpt-4.1" {
		t.Fatalf("request model provider/model mismatch: %+v", got)
	}
	if got.SystemPrompt != "system" || got.UserPrompt != "user" {
		t.Fatalf("request prompts mismatch: %+v", got)
	}
	if got.RuntimeContext != "Entry key: checkout.title" {
		t.Fatalf("request runtime context = %q, want %q", got.RuntimeContext, "Entry key: checkout.title")
	}
}

func TestTranslateWithRetryBuildsRequestWithSourceContext(t *testing.T) {
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
		SourceContext: "Checkout submit button",
	}

	_, err := svc.translateWithRetry(context.Background(), task)
	if err != nil {
		t.Fatalf("translateWithRetry returned error: %v", err)
	}

	wantRuntime := "Entry key: checkout.title\n\nSource context:\nCheckout submit button"
	if got.RuntimeContext != wantRuntime {
		t.Fatalf("request runtime context = %q, want %q", got.RuntimeContext, wantRuntime)
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
		SourceContext: "Checkout submit button",
		ContextMemory: "Shared context",
	}

	_, err := svc.translateWithRetry(context.Background(), task)
	if err != nil {
		t.Fatalf("translateWithRetry returned error: %v", err)
	}

	wantRuntime := "Entry key: checkout.title\n\nSource context:\nCheckout submit button\n\nShared memory:\nShared context"
	if got.SystemPrompt != "" {
		t.Fatalf("request system prompt = %q, want empty string", got.SystemPrompt)
	}
	if got.RuntimeContext != wantRuntime {
		t.Fatalf("request runtime context = %q, want %q", got.RuntimeContext, wantRuntime)
	}
	if got.UserPrompt != "" {
		t.Fatalf("request user prompt = %q, want empty string", got.UserPrompt)
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
	if got.SystemPrompt != "" {
		t.Fatalf("request system prompt = %q, want empty string", got.SystemPrompt)
	}
	if got.RuntimeContext != "" {
		t.Fatalf("request runtime context = %q, want empty string", got.RuntimeContext)
	}
	if got.UserPrompt != "" {
		t.Fatalf("request user prompt = %q, want empty string", got.UserPrompt)
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
	wantRuntime := "Shared memory:\nShared context"
	if got.SystemPrompt != "" {
		t.Fatalf("request system prompt = %q, want empty string", got.SystemPrompt)
	}
	if got.RuntimeContext != wantRuntime {
		t.Fatalf("request runtime context = %q, want %q", got.RuntimeContext, wantRuntime)
	}
	if got.UserPrompt != "" {
		t.Fatalf("request user prompt = %q, want empty string", got.UserPrompt)
	}
}

func TestTranslateWithRetryRejectsTranslatedICUKeyword(t *testing.T) {
	svc := &Service{}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "{plan, 选择, pro{专业计划} other{免费计划}}", nil
	}

	_, err := svc.translateWithRetry(context.Background(), Task{
		EntryKey:     "settings.planBadge",
		SourceText:   "{plan, select, pro{Pro plan} other{Free plan}}",
		TargetLocale: "zh-CN",
	})
	if err == nil {
		t.Fatalf("expected invariant validation error")
	}
	if !strings.Contains(err.Error(), "translation invariant violation") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), `source="{plan, select, pro{Pro plan} other{Free plan}}"`) {
		t.Fatalf("expected source context in error, got %v", err)
	}
	if !strings.Contains(err.Error(), `candidate="{plan, 选择, pro{专业计划} other{免费计划}}"`) {
		t.Fatalf("expected candidate context in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "diff=at=") {
		t.Fatalf("expected diff context in error, got %v", err)
	}
}

func TestTranslateWithRetryRejectsTranslatedPlaceholderName(t *testing.T) {
	svc := &Service{}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "Chao mung {ten}", nil
	}

	_, err := svc.translateWithRetry(context.Background(), Task{
		EntryKey:     "auth.welcomeBack",
		SourceText:   "Welcome back {name}",
		TargetLocale: "vi-VN",
	})
	if err == nil {
		t.Fatalf("expected invariant validation error")
	}
	if !strings.Contains(err.Error(), "placeholder parity mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), `source="Welcome back {name}"`) {
		t.Fatalf("expected source context in error, got %v", err)
	}
	if !strings.Contains(err.Error(), `candidate="Chao mung {ten}"`) {
		t.Fatalf("expected candidate context in error, got %v", err)
	}
}

func TestTranslateWithRetryAcceptsPluralPoundForPluralArg(t *testing.T) {
	svc := &Service{}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "{count, plural, =0{无邀请} one{# 邀请} other{# 邀请}}", nil
	}

	got, err := svc.translateWithRetry(context.Background(), Task{
		EntryKey:     "inviteCount",
		SourceText:   "{count, plural, =0{No invites} one{1 invite} other{{count} invites}}",
		TargetLocale: "zh-CN",
	})
	if err != nil {
		t.Fatalf("expected valid plural rewrite, got %v", err)
	}
	if got == "" {
		t.Fatalf("expected translated text")
	}
}

func TestTranslateWithRetryAcceptsPluralPoundWhenSourceOmitsExplicitCount(t *testing.T) {
	svc := &Service{}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "{count, plural, one {# 件} other {# 件}}", nil
	}

	got, err := svc.translateWithRetry(context.Background(), Task{
		EntryKey:     "itemCount",
		SourceText:   "{count, plural, one {a single item} other {many items}}",
		TargetLocale: "zh-CN",
	})
	if err != nil {
		t.Fatalf("expected valid plural rewrite, got %v", err)
	}
	if got == "" {
		t.Fatalf("expected translated text")
	}
}

func TestTranslateWithRetryRejectsDuplicatePluralPoundUsage(t *testing.T) {
	svc := &Service{}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "{count, plural, =0{待审核的评论数量为0} one{# 还有# 待审核的评论} other{# 还有# 待审核的评论}}", nil
	}

	_, err := svc.translateWithRetry(context.Background(), Task{
		EntryKey:     "dashboard.pendingReviews",
		SourceText:   "{count, plural, =0{No reviews pending} one{# review pending} other{# reviews pending}}",
		TargetLocale: "zh-CN",
	})
	if err == nil {
		t.Fatalf("expected invariant validation error")
	}
	if !strings.Contains(err.Error(), "duplicate # tokens in ICU plural/selectordinal branch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTranslateWithRetrySanitizesEntryKeyInSystemPromptContext(t *testing.T) {
	svc := &Service{}
	var got translator.Request
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		got = req
		return "ok", nil
	}

	longKey := strings.Repeat("界", maxScopeIdentifierLen+8)
	task := Task{
		EntryKey:      "  checkout.title\n" + longKey + "\rnext  ",
		SourceText:    "Hello",
		TargetLocale:  "fr",
		ContextMemory: "",
	}

	_, err := svc.translateWithRetry(context.Background(), task)
	if err != nil {
		t.Fatalf("translateWithRetry returned error: %v", err)
	}
	if got.RuntimeContext == "" {
		t.Fatalf("expected runtime context to be populated")
	}
	if strings.Contains(got.RuntimeContext, "\r") || strings.Contains(got.RuntimeContext, "\nnext") {
		t.Fatalf("expected entry key newlines to be stripped in runtime context, got %q", got.RuntimeContext)
	}

	entryLine := strings.TrimPrefix(got.RuntimeContext, "Entry key: ")
	if len([]rune(entryLine)) > maxScopeIdentifierLen {
		t.Fatalf("expected entry key to be capped at %d runes, got %d", maxScopeIdentifierLen, len([]rune(entryLine)))
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

func TestSanitizePromptContextTruncatesWithEllipsis(t *testing.T) {
	got := sanitizePromptContext("abcdefghij", 5)
	if got != "abcd…" {
		t.Fatalf("sanitizePromptContext() = %q, want %q", got, "abcd…")
	}
}

func TestSanitizePromptContextUsesEllipsisWhenMaxLenIsOne(t *testing.T) {
	got := sanitizePromptContext("abcdefghij", 1)
	if got != "…" {
		t.Fatalf("sanitizePromptContext() = %q, want %q", got, "…")
	}
}

func TestSanitizePromptContext(t *testing.T) {
	got := sanitizePromptContext(" line 1\n\n line 2 \r\n", 0)
	if got != "line 1\nline 2" {
		t.Fatalf("sanitizePromptContext() = %q, want %q", got, "line 1\nline 2")
	}
}
