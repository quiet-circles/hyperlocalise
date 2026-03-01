package translator

import (
	"context"
	"os"
	"strings"

	"github.com/openai/openai-go/v2/option"
)

const (
	defaultOllamaBaseURL    = "http://127.0.0.1:11434/v1"
	defaultOllamaBaseURLEnv = "OLLAMA_BASE_URL"
	defaultOllamaAPIKeyEnv  = "OLLAMA_API_KEY"
	defaultOllamaAPIKey     = "ollama"
)

type OllamaProvider struct{}

func NewOllamaProvider() *OllamaProvider { return &OllamaProvider{} }

func (p *OllamaProvider) Name() string { return ProviderOllama }

func (p *OllamaProvider) Translate(ctx context.Context, req Request) (string, error) {
	baseURL := strings.TrimSpace(os.Getenv(defaultOllamaBaseURLEnv))
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}

	apiKey := strings.TrimSpace(os.Getenv(defaultOllamaAPIKeyEnv))
	if apiKey == "" {
		apiKey = defaultOllamaAPIKey
	}

	return translateWithOpenAICompatibleClient(
		ctx,
		ProviderOllama,
		req,
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)
}
