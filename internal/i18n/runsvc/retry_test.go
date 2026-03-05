package runsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

func TestTranslateWithRetryAutoRepairRepairsDetectedLeak(t *testing.T) {
	svc := newTestService()
	calls := 0
	svc.translate = func(ctx context.Context, req translator.Request) (string, error) {
		calls++
		if strings.Contains(req.Source, "TRANSLATION_DRAFT:") {
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
		if strings.Contains(req.Source, "TRANSLATION_DRAFT:") {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldAttemptAutoRepair(tt.sourceLoc, tt.targetLoc, tt.source, tt.translated); got != tt.want {
				t.Fatalf("shouldAttemptAutoRepair(%q, %q, %q, %q)=%v, want %v", tt.sourceLoc, tt.targetLoc, tt.source, tt.translated, got, tt.want)
			}
		})
	}
}
