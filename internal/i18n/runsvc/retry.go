package runsvc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

const (
	translationRetryMaxAttempts = 5
	translationRetryBaseDelay   = 250 * time.Millisecond
	translationRetryMaxDelay    = 5 * time.Second
)

var sleepWithContext = func(ctx context.Context, delay time.Duration) error {
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (s *Service) translateWithRetry(ctx context.Context, task Task) (string, error) {
	requestContext := strings.TrimSpace(task.EntryKey)
	if memory := strings.TrimSpace(task.ContextMemory); memory != "" {
		requestContext = requestContext + "\n\nShared memory:\n" + memory
	}

	request := translator.Request{
		Source:         task.SourceText,
		TargetLanguage: task.TargetLocale,
		Context:        requestContext,
		ModelProvider:  task.Provider,
		Model:          task.Model,
		SystemPrompt:   task.SystemPrompt,
		UserPrompt:     task.UserPrompt,
	}

	return s.translateRequestWithRetry(ctx, request)
}

func (s *Service) translateRequestWithRetry(ctx context.Context, request translator.Request) (string, error) {
	var lastErr error
	attempt := 0
	for attempt = range translationRetryMaxAttempts {
		translated, err := s.translate(ctx, request)
		if err == nil {
			return translated, nil
		}
		lastErr = err
		if !isRetryableTranslateError(err) || attempt+1 >= translationRetryMaxAttempts {
			break
		}

		delay := translationRetryDelay(attempt)
		if waitErr := sleepWithContext(ctx, delay); waitErr != nil {
			return "", fmt.Errorf("translation retry wait interrupted: %w", waitErr)
		}
	}

	if lastErr == nil {
		return "", nil
	}
	return "", fmt.Errorf("translation failed after %d attempts: %w", attempt+1, lastErr)
}

func isRetryableTranslateError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") {
		return true
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "timed out") {
		return true
	}
	if strings.Contains(msg, "status code 500") || strings.Contains(msg, "status code 502") || strings.Contains(msg, "status code 503") || strings.Contains(msg, "status code 504") {
		return true
	}
	if strings.Contains(msg, "service unavailable") || strings.Contains(msg, "temporarily unavailable") {
		return true
	}

	return false
}

func translationRetryDelay(attempt int) time.Duration {
	factor := math.Pow(2, float64(attempt))
	delay := time.Duration(float64(translationRetryBaseDelay) * factor)
	if delay > translationRetryMaxDelay {
		return translationRetryMaxDelay
	}
	return delay
}
