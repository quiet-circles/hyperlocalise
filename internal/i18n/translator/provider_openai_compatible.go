package translator

import (
	"context"
	"fmt"
	"reflect"
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

	if usage, ok := usageFromGenerateTextResponse(resp); ok {
		SetUsage(ctx, usage)
	}

	return output, nil
}

func usageFromGenerateTextResponse(resp any) (Usage, bool) {
	rv := reflect.ValueOf(resp)
	if !rv.IsValid() {
		return Usage{}, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return Usage{}, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return Usage{}, false
	}

	usageField := rv.FieldByName("Usage")
	if !usageField.IsValid() {
		return Usage{}, false
	}

	return usageFromValue(usageField)
}

func usageFromValue(value reflect.Value) (Usage, bool) {
	if !value.IsValid() {
		return Usage{}, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return Usage{}, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return Usage{}, false
	}

	prompt, hasPrompt := intField(value, "PromptTokens", "InputTokens")
	completion, hasCompletion := intField(value, "CompletionTokens", "OutputTokens")
	total, hasTotal := intField(value, "TotalTokens")
	if !hasTotal {
		total = prompt + completion
	}
	if !hasPrompt && !hasCompletion && !hasTotal {
		return Usage{}, false
	}

	return Usage{PromptTokens: prompt, CompletionTokens: completion, TotalTokens: total}, true
}

func intField(value reflect.Value, names ...string) (int, bool) {
	for _, name := range names {
		field := value.FieldByName(name)
		if !field.IsValid() {
			continue
		}
		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				continue
			}
			field = field.Elem()
		}
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(field.Int()), true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(field.Uint()), true
		}
	}
	return 0, false
}
