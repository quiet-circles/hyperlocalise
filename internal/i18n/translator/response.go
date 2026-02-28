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

	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", fmt.Errorf("no text generated")
	}

	return text, nil
}
