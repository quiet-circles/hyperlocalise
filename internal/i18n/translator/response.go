package translator

import (
	"fmt"
	"strings"

	"github.com/openai/openai-go/v2"
)

func responseText(resp *openai.ChatCompletion) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("response is nil")
	}

	b := strings.Builder{}
	if len(resp.Choices) > 0 {
		b.WriteString(resp.Choices[0].Message.Content)
	}

	text := sanitizeGeneratedText(b.String())
	if text == "" {
		return "", fmt.Errorf("no text generated")
	}

	return text, nil
}

func sanitizeGeneratedText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	// Some local model/chat templates append control markers to the assistant text.
	// Strip known trailing markers so they are not written into translation files.
	trailingMarkers := []string{
		"<|END_RESPONSE|>",
		"<|end_response|>",
		"<|eot_id|>",
		"<|end_of_text|>",
		"</s>",
	}

	// Remove marker occurrences even when they are embedded in the text.
	for _, marker := range trailingMarkers {
		trimmed = strings.ReplaceAll(trimmed, marker, "")
	}
	return strings.TrimSpace(trimmed)
}
