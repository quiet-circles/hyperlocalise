package translator

import (
	"context"
	"fmt"
	"strings"
)

const (
	ProviderOpenAI      = "openai"
	ProviderAzureOpenAI = "azure_openai"
	ProviderAnthropic   = "anthropic"
	ProviderLMStudio    = "lmstudio"
	ProviderGroq        = "groq"
	ProviderMistral     = "mistral"
	ProviderOllama      = "ollama"
	ProviderGemini      = "gemini"
	ProviderBedrock     = "bedrock"
)

type Request struct {
	Source         string
	TargetLanguage string
	Context        string
	ModelProvider  string
	Model          string
	Prompt         string
}

type Provider interface {
	Name() string
	Translate(ctx context.Context, req Request) (string, error)
}

func validateRequest(req Request) error {
	if strings.TrimSpace(req.Source) == "" {
		return fmt.Errorf("translate request: source is required")
	}
	if strings.TrimSpace(req.TargetLanguage) == "" {
		return fmt.Errorf("translate request: target language is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return fmt.Errorf("translate request: model is required")
	}
	return nil
}

func normalizeProvider(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
