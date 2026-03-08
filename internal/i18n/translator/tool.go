package translator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Tool struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

var (
	defaultToolOnce    sync.Once
	defaultTool        *Tool
	defaultToolInitErr error
)

func Translate(ctx context.Context, req Request) (string, error) {
	// Initialization is attempted once. If it fails, all subsequent calls
	// to Translate will return the same error; re-initialization is not possible.
	defaultToolOnce.Do(func() {
		defaultTool, defaultToolInitErr = New()
	})
	if defaultToolInitErr != nil {
		return "", defaultToolInitErr
	}
	return defaultTool.Translate(ctx, req)
}

func New() (*Tool, error) {
	t := &Tool{providers: map[string]Provider{}}
	if err := RegisterBuiltins(t); err != nil {
		return nil, fmt.Errorf("creating translator: %w", err)
	}
	return t, nil
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

	systemPrompt := buildSystemPrompt(req)
	userPrompt := buildUserPrompt(req)
	logPromptCall(req, providerName, systemPrompt, userPrompt)
	req.SystemPrompt = systemPrompt
	req.UserPrompt = userPrompt
	req.RuntimeContext = ""

	start := time.Now()
	translated, err := provider.Translate(ctx, req)
	duration := time.Since(start)
	if err != nil {
		logPromptResult(req, providerName, "", err, duration)
		return "", fmt.Errorf("translate with provider %q: %w", providerName, err)
	}

	translated = strings.TrimSpace(translated)
	logPromptResult(req, providerName, translated, nil, duration)
	return translated, nil
}
