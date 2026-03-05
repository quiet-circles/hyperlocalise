package translator

import "fmt"

// RegisterBuiltins registers all built-in translation providers with the tool.
func RegisterBuiltins(t *Tool) error {
	if t == nil {
		return fmt.Errorf("register built-in translation providers: tool must not be nil")
	}

	providers := []Provider{
		NewOpenAIProvider(),
		NewAzureOpenAIProvider(),
		NewAnthropicProvider(),
		NewLMStudioProvider(),
		NewGroqProvider(),
		NewMistralProvider(),
		NewOllamaProvider(),
		NewGeminiProvider(),
		NewBedrockProvider(),
	}

	for _, provider := range providers {
		if err := t.Register(provider); err != nil {
			return fmt.Errorf("register built-in translation providers: %w", err)
		}
	}

	return nil
}
