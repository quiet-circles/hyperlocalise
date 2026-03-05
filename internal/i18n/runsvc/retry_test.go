package runsvc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

func TestTranslateWithRetryAutoRepairRepairsDetectedLeak(t *testing.T) {
	svc := newTestService()
	calls := 0
	svc.translate = func(ctx context.Context, req translator.Request) (string, error) {
		calls++
		if strings.TrimSpace(req.RepairDraft) != "" && strings.TrimSpace(req.RepairSource) != "" {
			translator.SetUsage(ctx, translator.Usage{PromptTokens: 7, CompletionTokens: 3, TotalTokens: 10})
			return "Bonjour le monde", nil
		}
		translator.SetUsage(ctx, translator.Usage{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7})
		return req.Source, nil
	}

	usage := translator.Usage{}
	task := Task{
		SourceText:   "Welcome to the developer dashboard",
		SourceLocale: "en",
		TargetLocale: "fr",
		EntryKey:     "hello",
		Provider:     "openai",
		Model:        "gpt-4.1-mini",
		Prompt:       "Translate from en to fr.",
		AutoRepair:   true,
	}

	got, _, err := svc.translateWithRetry(translator.WithUsageCollector(context.Background(), &usage), task)
	if err != nil {
		t.Fatalf("translate with auto-repair: %v", err)
	}
	if got != "Bonjour le monde" {
		t.Fatalf("unexpected repaired output: %q", got)
	}
	if calls != 2 {
		t.Fatalf("expected two translate calls (pass1 + repair), got %d", calls)
	}
	if usage.PromptTokens != 12 || usage.CompletionTokens != 5 || usage.TotalTokens != 17 {
		t.Fatalf("unexpected aggregated usage: %+v", usage)
	}
}

func TestTranslateWithRetryAutoRepairSkipsRepairWhenNoLeakDetected(t *testing.T) {
	svc := newTestService()
	calls := 0
	svc.translate = func(ctx context.Context, _ translator.Request) (string, error) {
		calls++
		translator.SetUsage(ctx, translator.Usage{PromptTokens: 6, CompletionTokens: 4, TotalTokens: 10})
		return "Bonjour", nil
	}

	usage := translator.Usage{}
	task := Task{
		SourceText:   "Welcome to the developer dashboard",
		SourceLocale: "en",
		TargetLocale: "fr",
		EntryKey:     "hello",
		Provider:     "openai",
		Model:        "gpt-4.1-mini",
		Prompt:       "Translate from en to fr.",
		AutoRepair:   true,
	}

	got, _, err := svc.translateWithRetry(translator.WithUsageCollector(context.Background(), &usage), task)
	if err != nil {
		t.Fatalf("translate with auto-repair: %v", err)
	}
	if got != "Bonjour" {
		t.Fatalf("unexpected output: %q", got)
	}
	if calls != 1 {
		t.Fatalf("expected one translate call when no repair is needed, got %d", calls)
	}
	if usage.PromptTokens != 6 || usage.CompletionTokens != 4 || usage.TotalTokens != 10 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestTranslateWithRetryAutoRepairFailsClosedWhenRepairPassFails(t *testing.T) {
	svc := newTestService()
	calls := 0
	svc.translate = func(ctx context.Context, req translator.Request) (string, error) {
		calls++
		if strings.TrimSpace(req.RepairDraft) != "" && strings.TrimSpace(req.RepairSource) != "" {
			return "", errors.New("repair request rejected")
		}
		translator.SetUsage(ctx, translator.Usage{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7})
		return req.Source, nil
	}

	task := Task{
		SourceText:   "Welcome to the developer dashboard",
		SourceLocale: "en",
		TargetLocale: "fr",
		EntryKey:     "hello",
		Provider:     "openai",
		Model:        "gpt-4.1-mini",
		Prompt:       "Translate from en to fr.",
		AutoRepair:   true,
	}

	_, outcome, err := svc.translateWithRetry(context.Background(), task)
	if err == nil || !strings.Contains(err.Error(), "auto-repair failed") {
		t.Fatalf("expected explicit auto-repair failure, got %v", err)
	}
	if !outcome.Triggered || !outcome.Failed || outcome.Succeeded {
		t.Fatalf("unexpected auto-repair outcome: %+v", outcome)
	}
	if calls != 2 {
		t.Fatalf("expected two calls before fail-closed, got %d", calls)
	}
}

func TestTranslateWithRetryAutoRepairKeepsPass1UsageOnPass1Failure(t *testing.T) {
	svc := newTestService()
	attempts := 0
	svc.translate = func(ctx context.Context, _ translator.Request) (string, error) {
		attempts++
		translator.SetUsage(ctx, translator.Usage{
			PromptTokens:     3 * attempts,
			CompletionTokens: attempts,
			TotalTokens:      (3 * attempts) + attempts,
		})
		return "", errors.New("pass1 failed")
	}

	usage := translator.Usage{}
	task := Task{
		SourceText:   "Welcome to the developer dashboard",
		SourceLocale: "en",
		TargetLocale: "fr",
		EntryKey:     "hello",
		Provider:     "openai",
		Model:        "gpt-4.1-mini",
		Prompt:       "Translate from en to fr.",
		AutoRepair:   true,
	}

	_, _, err := svc.translateWithRetry(translator.WithUsageCollector(context.Background(), &usage), task)
	if err == nil {
		t.Fatalf("expected pass1 error")
	}
	if attempts != 1 {
		t.Fatalf("expected one non-retryable attempt, got %d", attempts)
	}
	if usage.PromptTokens != 3 || usage.CompletionTokens != 1 || usage.TotalTokens != 4 {
		t.Fatalf("unexpected usage preserved from pass1 failure: %+v", usage)
	}
}

func TestShouldAttemptAutoRepair(t *testing.T) {
	tests := []struct {
		name       string
		sourceLoc  string
		targetLoc  string
		source     string
		translated string
		want       bool
	}{
		{
			name:       "exact source copy",
			sourceLoc:  "en",
			targetLoc:  "fr",
			source:     "Welcome to the developer dashboard",
			translated: "Welcome to the developer dashboard",
			want:       true,
		},
		{
			name:       "same-language pair is guarded",
			sourceLoc:  "en-US",
			targetLoc:  "en-GB",
			source:     "Install the package and restart the development server",
			translated: "Install the package and restart the server now",
			want:       false,
		},
		{
			name:       "short text is guarded",
			sourceLoc:  "en",
			targetLoc:  "fr",
			source:     "Hello",
			translated: "Hello",
			want:       false,
		},
		{
			name:       "unknown locale uses strict overlap fallback",
			sourceLoc:  "en",
			targetLoc:  "xx",
			source:     "Install the package and restart the development server",
			translated: "Install the package and restart the server now",
			want:       false,
		},
		{
			name:       "clean target translation",
			sourceLoc:  "en",
			targetLoc:  "fr",
			source:     "Install the package and restart the development server",
			translated: "Installez le paquet et redémarrez le serveur de développement",
			want:       false,
		},
		{
			name:       "borrowed proper nouns should not auto-repair",
			sourceLoc:  "en",
			targetLoc:  "fr",
			source:     "React Redux Webpack",
			translated: "React Redux Webpack",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldAttemptAutoRepair(tt.sourceLoc, tt.targetLoc, tt.source, tt.translated); got != tt.want {
				t.Fatalf("shouldAttemptAutoRepair(%q, %q, %q, %q)=%v, want %v", tt.sourceLoc, tt.targetLoc, tt.source, tt.translated, got, tt.want)
			}
		})
	}
}

func TestTargetLanguageConfidenceCountsShortStopwords(t *testing.T) {
	confidence, known := targetLanguageConfidence("pt-BR", "o sistema e a plataforma com as traducoes")
	if !known {
		t.Fatalf("expected locale to be known")
	}
	if confidence <= 0 {
		t.Fatalf("expected positive confidence from short Portuguese stopwords, got %f", confidence)
	}
}

func TestTranslateRequestWithRetryAccumulatesUsageAcrossAttempts(t *testing.T) {
	svc := newTestService()
	attempts := 0
	originalSleep := sleepWithContext
	t.Cleanup(func() { sleepWithContext = originalSleep })
	sleepWithContext = func(ctx context.Context, _ time.Duration) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	svc.translate = func(ctx context.Context, _ translator.Request) (string, error) {
		attempts++
		translator.SetUsage(ctx, translator.Usage{
			PromptTokens:     10 * attempts,
			CompletionTokens: 5 * attempts,
			TotalTokens:      15 * attempts,
		})
		if attempts < 3 {
			return "", errors.New("status code 429")
		}
		return "Bonjour", nil
	}

	usage := translator.Usage{}
	got, err := svc.translateRequestWithRetry(translator.WithUsageCollector(context.Background(), &usage), translator.Request{
		Source:         "Hello",
		TargetLanguage: "fr",
		ModelProvider:  "openai",
		Model:          "gpt-4.1-mini",
		Prompt:         "Translate",
	})
	if err != nil {
		t.Fatalf("translateRequestWithRetry: %v", err)
	}
	if got != "Bonjour" {
		t.Fatalf("unexpected output: %q", got)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if usage.PromptTokens != 60 || usage.CompletionTokens != 30 || usage.TotalTokens != 90 {
		t.Fatalf("expected summed usage across attempts, got %+v", usage)
	}
}
