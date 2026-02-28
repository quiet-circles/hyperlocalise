package translator

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v2/option"
	"go.jetify.com/ai"
	"go.jetify.com/ai/api"
	jetifyopenai "go.jetify.com/ai/provider/openai"
)

func translateWithOpenAICompatibleClient(ctx context.Context, providerName string, req Request, opts ...option.RequestOption) (string, error) {
	model := jetifyopenai.NewLanguageModel(
		strings.TrimSpace(req.Model),
		jetifyopenai.WithClient(newOpenAIClient(opts...)),
	)

	messages := []api.Message{
		&api.SystemMessage{Content: buildSystemPrompt(req.Prompt)},
		&api.UserMessage{Content: api.ContentFromText(buildUserPrompt(req))},
	}

	resp, err := ai.GenerateText(ctx, messages, ai.WithModel(model))
	if err != nil {
		return "", fmt.Errorf("%s generate text: %w", providerName, err)
	}

	output, err := responseText(resp)
	if err != nil {
		return "", fmt.Errorf("%s response: %w", providerName, err)
	}

	return output, nil
}
