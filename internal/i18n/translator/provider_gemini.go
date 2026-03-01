package translator

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v2/option"
)

const (
	defaultGeminiBaseURL    = "https://generativelanguage.googleapis.com/v1beta/openai"
	defaultGeminiBaseURLEnv = "GEMINI_BASE_URL"
	defaultGeminiAPIKeyEnv  = "GEMINI_API_KEY"
)

type GeminiProvider struct{}

func NewGeminiProvider() *GeminiProvider { return &GeminiProvider{} }

func (p *GeminiProvider) Name() string { return ProviderGemini }

func (p *GeminiProvider) Translate(ctx context.Context, req Request) (string, error) {
	baseURL := strings.TrimSpace(os.Getenv(defaultGeminiBaseURLEnv))
	if baseURL == "" {
		baseURL = defaultGeminiBaseURL
	}

	apiKey := strings.TrimSpace(os.Getenv(defaultGeminiAPIKeyEnv))
	if apiKey == "" {
		return "", fmt.Errorf("gemini provider: API key is required (%s)", defaultGeminiAPIKeyEnv)
	}

	return translateWithOpenAICompatibleClient(
		ctx,
		ProviderGemini,
		req,
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)
}
