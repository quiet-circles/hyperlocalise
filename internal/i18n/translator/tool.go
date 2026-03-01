package translator

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type Tool struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

var (
	defaultToolOnce sync.Once
	defaultTool     *Tool
)

func Translate(ctx context.Context, req Request) (string, error) {
	defaultToolOnce.Do(func() {
		defaultTool = New()
	})
	return defaultTool.Translate(ctx, req)
}

func New() *Tool {
	t := &Tool{providers: map[string]Provider{}}
	t.MustRegister(NewOpenAIProvider())
	t.MustRegister(NewLMStudioProvider())
	t.MustRegister(NewGroqProvider())
	return t
}

func (t *Tool) Register(provider Provider) error {
	if provider == nil {
		return fmt.Errorf("register translation provider: provider must not be nil")
	}

	name := normalizeProvider(provider.Name())
	if name == "" {
		return fmt.Errorf("register translation provider: name must not be empty")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.providers[name]; exists {
		return fmt.Errorf("register translation provider %q: already registered", name)
	}

	t.providers[name] = provider
	return nil
}

func (t *Tool) MustRegister(provider Provider) {
	if err := t.Register(provider); err != nil {
		panic(err)
	}
}

func (t *Tool) Translate(ctx context.Context, req Request) (string, error) {
	if err := validateRequest(req); err != nil {
		return "", err
	}

	providerName := normalizeProvider(req.ModelProvider)
	if providerName == "" {
		providerName = ProviderOpenAI
	}

	t.mu.RLock()
	provider, ok := t.providers[providerName]
	t.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("translate: unknown model provider %q", providerName)
	}

	translated, err := provider.Translate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("translate with provider %q: %w", providerName, err)
	}

	return strings.TrimSpace(translated), nil
}
