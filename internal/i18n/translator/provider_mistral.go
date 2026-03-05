package translator

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v2/option"
)

const (
	defaultMistralBaseURL    = "https://api.mistral.ai/v1"
	defaultMistralBaseURLEnv = "MISTRAL_BASE_URL"
	defaultMistralAPIKeyEnv  = "MISTRAL_API_KEY"
)

type MistralProvider struct{}

func NewMistralProvider() *MistralProvider { return &MistralProvider{} }

func (p *MistralProvider) Name() string { return ProviderMistral }

func (p *MistralProvider) Translate(ctx context.Context, req Request) (string, error) {
	baseURL := strings.TrimSpace(os.Getenv(defaultMistralBaseURLEnv))
	if baseURL == "" {
		baseURL = defaultMistralBaseURL
	}

	apiKey := strings.TrimSpace(os.Getenv(defaultMistralAPIKeyEnv))
	if apiKey == "" {
		return "", fmt.Errorf("mistral provider: API key is required (%s)", defaultMistralAPIKeyEnv)
	}

	return translateWithOpenAICompatibleClient(
		ctx,
		ProviderMistral,
		req,
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)
}
