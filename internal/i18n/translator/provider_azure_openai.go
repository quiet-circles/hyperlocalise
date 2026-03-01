package translator

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v2/option"
)

const (
	defaultAzureOpenAIBaseURLEnv = "AZURE_OPENAI_BASE_URL"
	defaultAzureOpenAIAPIKeyEnv  = "AZURE_OPENAI_API_KEY"
)

type AzureOpenAIProvider struct{}

func NewAzureOpenAIProvider() *AzureOpenAIProvider { return &AzureOpenAIProvider{} }

func (p *AzureOpenAIProvider) Name() string { return ProviderAzureOpenAI }

func (p *AzureOpenAIProvider) Translate(ctx context.Context, req Request) (string, error) {
	baseURL := strings.TrimSpace(os.Getenv(defaultAzureOpenAIBaseURLEnv))
	if baseURL == "" {
		return "", fmt.Errorf("azure openai provider: base URL is required (%s)", defaultAzureOpenAIBaseURLEnv)
	}

	apiKey := strings.TrimSpace(os.Getenv(defaultAzureOpenAIAPIKeyEnv))
	if apiKey == "" {
		return "", fmt.Errorf("azure openai provider: API key is required (%s)", defaultAzureOpenAIAPIKeyEnv)
	}

	return translateWithOpenAICompatibleClient(
		ctx,
		ProviderAzureOpenAI,
		req,
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)
}
