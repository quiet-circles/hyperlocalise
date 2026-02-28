package translator

import (
	"context"
	"os"
	"strings"

	"github.com/openai/openai-go/v2/option"
)

const (
	defaultLMStudioBaseURL    = "http://127.0.0.1:1234/v1"
	defaultLMStudioBaseURLEnv = "LM_STUDIO_BASE_URL"
	defaultLMStudioAPIKeyEnv  = "LM_STUDIO_API_KEY"
	defaultLMStudioAPIKey     = "lm-studio"
)

type LMStudioProvider struct{}

func NewLMStudioProvider() *LMStudioProvider { return &LMStudioProvider{} }

func (p *LMStudioProvider) Name() string { return ProviderLMStudio }

func (p *LMStudioProvider) Translate(ctx context.Context, req Request) (string, error) {
	baseURL := strings.TrimSpace(os.Getenv(defaultLMStudioBaseURLEnv))
	if baseURL == "" {
		baseURL = defaultLMStudioBaseURL
	}

	apiKey := strings.TrimSpace(os.Getenv(defaultLMStudioAPIKeyEnv))
	if apiKey == "" {
		apiKey = defaultLMStudioAPIKey
	}

	return translateWithOpenAICompatibleClient(
		ctx,
		ProviderLMStudio,
		req,
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)
}
