package translator

import (
	"fmt"
	"strings"

	"go.jetify.com/ai/api"
)

func responseText(resp *api.Response) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("response is nil")
	}

	b := strings.Builder{}
	for _, block := range resp.Content {
		textBlock, ok := block.(*api.TextBlock)
		if !ok {
			continue
		}
		b.WriteString(textBlock.Text)
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
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}

	changed := true
	for changed {
		changed = false
		for _, marker := range trailingMarkers {
			if strings.HasSuffix(trimmed, marker) {
				trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, marker))
				changed = true
			}
		}
	}

	return trimmed
}
