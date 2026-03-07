package translator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTranslateWritesPromptDebugLogWhenEnabled(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), ".hyperlocalise", "logs", "prompt.log")
	t.Setenv(envPromptDebugEnabled, "1")
	t.Setenv(envPromptDebugPath, logPath)

	tool := &Tool{providers: map[string]Provider{}}
	if err := tool.Register(fakeProvider{name: ProviderOpenAI, result: "bonjour"}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	_, err := tool.Translate(context.Background(), Request{
		Source:         "hello",
		TargetLanguage: "fr",
		ModelProvider:  ProviderOpenAI,
		Model:          "gpt-5-mini",
		SystemPrompt:   "system prompt",
		UserPrompt:     "user prompt",
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	lines := splitNonEmptyLines(string(data))
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d: %q", len(lines), string(data))
	}

	var callEvent promptDebugEvent
	if err := json.Unmarshal([]byte(lines[0]), &callEvent); err != nil {
		t.Fatalf("unmarshal call event: %v", err)
	}
	if callEvent.Event != "prompt_call" {
		t.Fatalf("call event type = %q, want prompt_call", callEvent.Event)
	}
	if callEvent.SystemPrompt != "system prompt" {
		t.Fatalf("call system prompt = %q, want system prompt", callEvent.SystemPrompt)
	}
	if callEvent.UserPrompt != "user prompt" {
		t.Fatalf("call user prompt = %q, want user prompt", callEvent.UserPrompt)
	}

	var resultEvent promptDebugEvent
	if err := json.Unmarshal([]byte(lines[1]), &resultEvent); err != nil {
		t.Fatalf("unmarshal result event: %v", err)
	}
	if resultEvent.Event != "prompt_result" {
		t.Fatalf("result event type = %q, want prompt_result", resultEvent.Event)
	}
	if resultEvent.Output != "bonjour" {
		t.Fatalf("result output = %q, want bonjour", resultEvent.Output)
	}
	if resultEvent.Error != "" {
		t.Fatalf("result error = %q, want empty", resultEvent.Error)
	}
}

func TestTranslateWritesPromptDebugLogWhenGenericDebugEnabled(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), ".hyperlocalise", "logs", "prompt.log")
	t.Setenv(envPromptDebugEnabled, "")
	t.Setenv(envGenericDebug, "1")
	t.Setenv(envPromptDebugPath, logPath)

	tool := &Tool{providers: map[string]Provider{}}
	if err := tool.Register(fakeProvider{name: ProviderOpenAI, result: "bonjour"}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	_, err := tool.Translate(context.Background(), Request{
		Source:         "hello",
		TargetLanguage: "fr",
		ModelProvider:  ProviderOpenAI,
		Model:          "gpt-5-mini",
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	lines := splitNonEmptyLines(string(data))
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d: %q", len(lines), string(data))
	}
}

func splitNonEmptyLines(s string) []string {
	lines := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' {
			continue
		}
		if i > start {
			lines = append(lines, s[start:i])
		}
		start = i + 1
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
