package runsvc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/icuparser"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

const (
	translationRetryMaxAttempts = 5
	translationRetryBaseDelay   = 250 * time.Millisecond
	translationRetryMaxDelay    = 5 * time.Second
	maxSourceContextLen         = 800
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
	runtimeContext := buildTranslationRuntimeContext(task.EntryKey, task.SourceContext, task.ContextMemory)
	userPrompt := strings.TrimSpace(task.UserPrompt)

	request := translator.Request{
		Source:         task.SourceText,
		TargetLanguage: task.TargetLocale,
		ModelProvider:  task.Provider,
		Model:          task.Model,
		SystemPrompt:   task.SystemPrompt,
		UserPrompt:     userPrompt,
		RuntimeContext: runtimeContext,
	}

	return s.translateRequestWithRetry(ctx, request)
}

func buildTranslationRuntimeContext(entryKey, sourceContext, sharedMemory string) string {
	parts := make([]string, 0, 3)
	if key := sanitizeScopeIdentifier(entryKey); key != "" {
		parts = append(parts, "Entry key: "+key)
	}
	if sanitizedContext := sanitizePromptContext(sourceContext, maxSourceContextLen); sanitizedContext != "" {
		parts = append(parts, "Source context:\n"+sanitizedContext)
	}
	if memory := strings.TrimSpace(sharedMemory); memory != "" {
		parts = append(parts, "Shared memory:\n"+memory)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func (s *Service) translateRequestWithRetry(ctx context.Context, request translator.Request) (string, error) {
	var lastErr error
	attempt := 0
	for attempt = range translationRetryMaxAttempts {
		translated, err := s.translate(ctx, request)
		if err == nil {
			if err := validateTranslatedInvariant(request.Source, translated); err != nil {
				return "", err
			}
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

func validateTranslatedInvariant(source, translated string) error {
	srcInv, srcErr := icuparser.ParseInvariant(strings.TrimSpace(source))
	if srcErr != nil {
		return nil
	}
	if len(srcInv.Placeholders) == 0 && len(srcInv.ICUBlocks) == 0 {
		return nil
	}

	translatedInv, translatedErr := icuparser.ParseInvariant(strings.TrimSpace(translated))
	if translatedErr != nil {
		return fmt.Errorf("translation invariant violation: invalid ICU/braces structure: %w", translatedErr)
	}
	if !samePlaceholderSet(srcInv.Placeholders, translatedInv.Placeholders) {
		return fmt.Errorf(
			"translation invariant violation: placeholder parity mismatch (expected %v, got %v)",
			srcInv.Placeholders,
			translatedInv.Placeholders,
		)
	}
	if !sameICUBlocks(srcInv.ICUBlocks, translatedInv.ICUBlocks) {
		return fmt.Errorf(
			"translation invariant violation: ICU parity mismatch (expected %s, got %s)",
			formatICUBlocks(srcInv.ICUBlocks),
			formatICUBlocks(translatedInv.ICUBlocks),
		)
	}
	return nil
}

func sameICUBlocks(a, b []icuparser.BlockSignature) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Arg != b[i].Arg || a[i].Type != b[i].Type || !slices.Equal(a[i].Options, b[i].Options) {
			return false
		}
	}
	return true
}

func samePlaceholderSet(a, b []string) bool {
	return slices.Equal(uniqueStrings(a), uniqueStrings(b))
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	var last string
	for i, value := range values {
		if i == 0 || value != last {
			out = append(out, value)
			last = value
		}
	}
	return out
}

func formatICUBlocks(blocks []icuparser.BlockSignature) string {
	if len(blocks) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		parts = append(parts, fmt.Sprintf("%s:%s%v", b.Arg, b.Type, b.Options))
	}
	return "[" + strings.Join(parts, ", ") + "]"
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

func sanitizePromptContext(value string, maxLen int) string {
	clean := strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(clean, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return ""
	}
	joined := strings.Join(out, "\n")
	if maxLen > 0 {
		runes := []rune(joined)
		if len(runes) > maxLen {
			const ellipsis = "…"
			if maxLen <= len([]rune(ellipsis)) {
				joined = ellipsis
			} else {
				joined = strings.TrimSpace(string(runes[:maxLen-len([]rune(ellipsis))])) + ellipsis
			}
		}
	}
	return joined
}
