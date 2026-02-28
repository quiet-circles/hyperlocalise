package translator

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v2/option"
)

const defaultOpenAIAPIKeyEnv = "OPENAI_API_KEY"

type OpenAIProvider struct{}

func NewOpenAIProvider() *OpenAIProvider { return &OpenAIProvider{} }

func (p *OpenAIProvider) Name() string { return ProviderOpenAI }

func (p *OpenAIProvider) Translate(ctx context.Context, req Request) (string, error) {
	apiKey := strings.TrimSpace(os.Getenv(defaultOpenAIAPIKeyEnv))
	if apiKey == "" {
		return "", fmt.Errorf("openai provider: API key is required (%s)", defaultOpenAIAPIKeyEnv)
	}

	return translateWithOpenAICompatibleClient(ctx, ProviderOpenAI, req, option.WithAPIKey(apiKey))
}
