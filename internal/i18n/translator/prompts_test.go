package translator

import (
	"strings"
	"testing"
)

func TestBuildSystemPromptPrefersSystemPrompt(t *testing.T) {
	t.Parallel()

	got := buildSystemPrompt(Request{
		SystemPrompt: "custom system",
	})

	if got != "custom system" {
		t.Fatalf("expected system_prompt to be used, got %q", got)
	}
}

func TestBuildSystemPromptUsesDefaultPolicyWhenNoPromptProvided(t *testing.T) {
	t.Parallel()

	got := buildSystemPrompt(Request{TargetLanguage: "vi-VN"})
	if !strings.Contains(got, "Return only the translated text") {
		t.Fatalf("expected default policy suffix, got %q", got)
	}
	if !strings.Contains(got, "Translate the user-provided source text") {
		t.Fatalf("expected default translation instruction, got %q", got)
	}
	if !strings.Contains(got, "Do not translate programmatic identifiers inside placeholders or ICU message syntax") {
		t.Fatalf("expected ICU preservation guidance, got %q", got)
	}
	if !strings.Contains(got, "plural, select, selectordinal") {
		t.Fatalf("expected ICU keyword list in default system prompt, got %q", got)
	}
	if !strings.Contains(got, "Target language: vi-VN") {
		t.Fatalf("expected target language in default system prompt, got %q", got)
	}
}

func TestBuildSystemPromptUsesDefaultPolicyWhenSystemPromptIsWhitespace(t *testing.T) {
	t.Parallel()

	got := buildSystemPrompt(Request{SystemPrompt: "   \n\t  "})
	if !strings.Contains(got, "Return only the translated text") {
		t.Fatalf("expected default policy suffix for whitespace prompt, got %q", got)
	}
}

func TestBuildUserPromptPrefersUserPrompt(t *testing.T) {
	t.Parallel()

	got := buildUserPrompt(Request{
		Source:         "hello",
		TargetLanguage: "fr",
		UserPrompt:     "custom user",
	})
	if got != "custom user" {
		t.Fatalf("expected custom user prompt, got %q", got)
	}
}

func TestBuildUserPromptIncludesICUGuidanceByDefault(t *testing.T) {
	t.Parallel()

	got := buildUserPrompt(Request{
		Source:         "{plan, select, pro{Pro} other{Free}}",
		TargetLanguage: "zh-CN",
	})
	if !strings.Contains(got, "Do not translate ICU keywords, selectors, or placeholder names") {
		t.Fatalf("expected ICU guidance in default user prompt, got %q", got)
	}
}

func TestBuildSystemPromptAppendsRuntimeContextWithDefaultPrompt(t *testing.T) {
	t.Parallel()

	got := buildSystemPrompt(Request{
		TargetLanguage: "fr",
		RuntimeContext: "Entry key: common.hello",
	})
	if !strings.Contains(got, "Target language: fr") {
		t.Fatalf("expected target language in default system prompt, got %q", got)
	}
	if !strings.Contains(got, "Runtime translation context (do not translate or repeat):\nEntry key: common.hello") {
		t.Fatalf("expected runtime context block in system prompt, got %q", got)
	}
}

func TestBuildSystemPromptAppendsRuntimeContextWithCustomSystemPrompt(t *testing.T) {
	t.Parallel()

	got := buildSystemPrompt(Request{
		SystemPrompt:   "custom system",
		RuntimeContext: "Entry key: common.hello",
	})
	if !strings.HasPrefix(got, "custom system\n\nRuntime translation context (do not translate or repeat):") {
		t.Fatalf("expected runtime context appended to custom system prompt, got %q", got)
	}
}
