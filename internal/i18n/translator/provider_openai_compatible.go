package translator

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

func translateWithOpenAICompatibleClient(ctx context.Context, providerName string, req Request, opts ...option.RequestOption) (string, error) {
	client := newOpenAIClient(opts...)

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(buildSystemPrompt(req.Prompt)),
			openai.UserMessage(buildUserPrompt(req)),
		},
		Model: openai.ChatModel(strings.TrimSpace(req.Model)),
	})
	if err != nil {
		return "", fmt.Errorf("%s generate text: %w", providerName, err)
	}

	output, err := responseText(resp)
	if err != nil {
		return "", fmt.Errorf("%s response: %w", providerName, err)
	}

	if usage, ok := usageFromGenerateTextResponse(resp); ok {
		SetUsage(ctx, usage)
	}

	return output, nil
}

func usageFromGenerateTextResponse(resp *openai.ChatCompletion) (Usage, bool) {
	if resp == nil {
		return Usage{}, false
	}

	prompt := int(resp.Usage.PromptTokens)
	completion := int(resp.Usage.CompletionTokens)
	total := int(resp.Usage.TotalTokens)
	if total == 0 && (prompt != 0 || completion != 0) {
		total = prompt + completion
	}
	if prompt == 0 && completion == 0 && total == 0 {
		return Usage{}, false
	}

	return Usage{PromptTokens: prompt, CompletionTokens: completion, TotalTokens: total}, true
}
