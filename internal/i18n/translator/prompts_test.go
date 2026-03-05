package translator

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildUserPromptTranslationPayloadShape(t *testing.T) {
	t.Parallel()

	prompt := buildUserPrompt(Request{
		Source:         "Hello world",
		TargetLanguage: "fr",
		Context:        "key=welcome",
	})
	if !strings.Contains(prompt, "TRANSLATION_REQUEST_JSON:") {
		t.Fatalf("expected translation marker, got %q", prompt)
	}
	if strings.Contains(prompt, "TRANSLATION_REPAIR_REQUEST_JSON:") {
		t.Fatalf("did not expect repair marker, got %q", prompt)
	}

	payload := extractPromptJSON(t, prompt, "TRANSLATION_REQUEST_JSON:")
	if got := payload["source_text"]; got != "Hello world" {
		t.Fatalf("unexpected source_text: %q", got)
	}
	if got := payload["target_language"]; got != "fr" {
		t.Fatalf("unexpected target_language: %q", got)
	}
	if got := payload["shared_context_guidance"]; got != "key=welcome" {
		t.Fatalf("unexpected shared_context_guidance: %q", got)
	}
}

func TestBuildUserPromptRepairPayloadShape(t *testing.T) {
	t.Parallel()

	prompt := buildUserPrompt(Request{
		TargetLanguage: "fr",
		RepairSource:   "ORIGINAL",
		RepairDraft:    "DRAFT",
		Context:        "key=welcome",
	})
	if !strings.Contains(prompt, "TRANSLATION_REPAIR_REQUEST_JSON:") {
		t.Fatalf("expected repair marker, got %q", prompt)
	}
	if strings.Contains(prompt, "TRANSLATION_REQUEST_JSON:") {
		t.Fatalf("did not expect translation marker, got %q", prompt)
	}

	payload := extractPromptJSON(t, prompt, "TRANSLATION_REPAIR_REQUEST_JSON:")
	if got := payload["original_source_text"]; got != "ORIGINAL" {
		t.Fatalf("unexpected original_source_text: %q", got)
	}
	if got := payload["translation_draft"]; got != "DRAFT" {
		t.Fatalf("unexpected translation_draft: %q", got)
	}
	if got := payload["target_language"]; got != "fr" {
		t.Fatalf("unexpected target_language: %q", got)
	}
	if got := payload["shared_context_guidance"]; got != "key=welcome" {
		t.Fatalf("unexpected shared_context_guidance: %q", got)
	}
	if _, has := payload["source_text"]; has {
		t.Fatalf("unexpected source_text field in repair payload")
	}
}

func TestValidateRequestRejectsPartialRepairPayload(t *testing.T) {
	t.Parallel()

	err := validateRequest(Request{
		Model:          "gpt-4.1-mini",
		TargetLanguage: "fr",
		RepairSource:   "only-source",
	})
	if err == nil || !strings.Contains(err.Error(), "repair_source and repair_draft must both be provided") {
		t.Fatalf("expected partial repair payload validation error, got %v", err)
	}
}

func extractPromptJSON(t *testing.T, prompt, marker string) map[string]string {
	t.Helper()
	idx := strings.Index(prompt, marker)
	if idx < 0 {
		t.Fatalf("marker %q not found in prompt", marker)
	}
	raw := strings.TrimSpace(prompt[idx+len(marker):])
	payload := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal prompt payload: %v\nraw=%q", err, raw)
	}
	return payload
}
