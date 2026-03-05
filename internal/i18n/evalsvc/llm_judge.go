package evalsvc

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

const defaultLLMJudgePrompt = "You are a translation quality judge. Return only a decimal score between 0 and 1, where 1 means excellent translation quality and 0 means unusable translation quality."

type llmJudgeScorer struct {
	translate func(context.Context, translator.Request) (string, error)
	provider  string
	model     string
	prompt    string
}

func newLLMJudgeScorer(translate func(context.Context, translator.Request) (string, error), provider string, model string, prompt string) *llmJudgeScorer {
	return &llmJudgeScorer{
		translate: translate,
		provider:  strings.TrimSpace(provider),
		model:     strings.TrimSpace(model),
		prompt:    strings.TrimSpace(prompt),
	}
}

func (s *llmJudgeScorer) Name() string {
	return "judge"
}

func (s *llmJudgeScorer) ScoreJudge(ctx context.Context, in ScoreInput) (float64, error) {
	if s.translate == nil {
		return 0, fmt.Errorf("judge scorer translate function is nil")
	}

	prompt := s.prompt
	if prompt == "" {
		prompt = defaultLLMJudgePrompt
	}

	req := translator.Request{
		Source:         buildLLMJudgeSource(in),
		TargetLanguage: "a numeric quality score in [0,1]",
		Context:        "Evaluate the candidate translation quality and return only one decimal number between 0 and 1.",
		ModelProvider:  s.provider,
		Model:          s.model,
		Prompt:         prompt,
	}

	response, err := s.translate(ctx, req)
	if err != nil {
		return 0, err
	}

	score, parseErr := parseJudgeScore(response)
	if parseErr != nil {
		return 0, fmt.Errorf("parse judge score: %w", parseErr)
	}

	return score, nil
}

func buildLLMJudgeSource(in ScoreInput) string {
	b := strings.Builder{}
	b.WriteString("Evaluate translation quality.\n")
	b.WriteString("Source:\n")
	b.WriteString(in.Case.Source)
	b.WriteString("\n\nReference:\n")
	b.WriteString(in.Case.Reference)
	b.WriteString("\n\nCandidate:\n")
	b.WriteString(in.Translated)
	b.WriteString("\n")
	return b.String()
}

var firstNumberPattern = regexp.MustCompile(`[-+]?\d*\.?\d+`)

func parseJudgeScore(raw string) (float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("empty judge output")
	}

	matched := firstNumberPattern.FindString(trimmed)
	if matched == "" {
		return 0, fmt.Errorf("no numeric score found in %q", trimmed)
	}

	score, err := strconv.ParseFloat(matched, 64)
	if err != nil {
		return 0, fmt.Errorf("parse float %q: %w", matched, err)
	}

	if score < 0 || score > 1 {
		return 0, fmt.Errorf("score %v out of range [0,1]", score)
	}

	return score, nil
}
