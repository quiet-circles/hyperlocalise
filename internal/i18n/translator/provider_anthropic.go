package translator

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v2/option"
)

const (
	defaultAnthropicBaseURL    = "https://api.anthropic.com/v1"
	defaultAnthropicBaseURLEnv = "ANTHROPIC_BASE_URL"
	defaultAnthropicAPIKeyEnv  = "ANTHROPIC_API_KEY"
)

type AnthropicProvider struct{}

func NewAnthropicProvider() *AnthropicProvider { return &AnthropicProvider{} }

func (p *AnthropicProvider) Name() string { return ProviderAnthropic }

func (p *AnthropicProvider) Translate(ctx context.Context, req Request) (string, error) {
	baseURL := strings.TrimSpace(os.Getenv(defaultAnthropicBaseURLEnv))
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}

	apiKey := strings.TrimSpace(os.Getenv(defaultAnthropicAPIKeyEnv))
	if apiKey == "" {
		return "", fmt.Errorf("anthropic provider: API key is required (%s)", defaultAnthropicAPIKeyEnv)
	}

	return translateWithOpenAICompatibleClient(
		ctx,
		ProviderAnthropic,
		req,
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)
}
