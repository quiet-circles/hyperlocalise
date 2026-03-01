package translator

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v2/option"
)

const (
	defaultGroqBaseURL    = "https://api.groq.com/openai/v1"
	defaultGroqBaseURLEnv = "GROQ_BASE_URL"
	defaultGroqAPIKeyEnv  = "GROQ_API_KEY"
)

type GroqProvider struct{}

func NewGroqProvider() *GroqProvider { return &GroqProvider{} }

func (p *GroqProvider) Name() string { return ProviderGroq }

func (p *GroqProvider) Translate(ctx context.Context, req Request) (string, error) {
	baseURL := strings.TrimSpace(os.Getenv(defaultGroqBaseURLEnv))
	if baseURL == "" {
		baseURL = defaultGroqBaseURL
	}

	apiKey := strings.TrimSpace(os.Getenv(defaultGroqAPIKeyEnv))
	if apiKey == "" {
		return "", fmt.Errorf("groq provider: API key is required (%s)", defaultGroqAPIKeyEnv)
	}

	return translateWithOpenAICompatibleClient(
		ctx,
		ProviderGroq,
		req,
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)
}
