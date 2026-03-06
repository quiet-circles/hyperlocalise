package evalsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

const llmJudgeName = "llm_judge"

const defaultLLMJudgePrompt = `You are an expert translation evaluator.
Score the candidate translation from 0.0 to 1.0.
Consider accuracy, fluency, terminology, tone, locale fit, and placeholder preservation.
Use the reference only as optional style and tone guidance, not as an exact-match target.
Return strict JSON with this shape only: {"score":0.0,"rationale":"brief explanation"}.`

type judgeTranslateFunc func(ctx context.Context, req translator.Request) (string, error)

type LLMJudgeScorer struct {
	provider  string
	model     string
	prompt    string
	translate judgeTranslateFunc
}

func NewLLMJudgeScorer(provider, model, prompt string, translate judgeTranslateFunc) *LLMJudgeScorer {
	if translate == nil {
		translate = translator.Translate
	}

	return &LLMJudgeScorer{
		provider:  strings.TrimSpace(provider),
		model:     strings.TrimSpace(model),
		prompt:    effectiveLLMJudgePrompt(prompt),
		translate: translate,
	}
}

func (s *LLMJudgeScorer) Name() string { return llmJudgeName }

func (s *LLMJudgeScorer) ScoreJudge(ctx context.Context, in ScoreInput) (JudgeResult, error) {
	if strings.TrimSpace(in.Translated) == "" {
		return JudgeResult{}, fmt.Errorf("judge translation is empty")
	}

	resp, err := s.translate(ctx, translator.Request{
		Source:         in.Case.Source,
		TargetLanguage: in.Case.TargetLocale,
		Context:        in.Case.Context,
		ModelProvider:  s.provider,
		Model:          s.model,
		SystemPrompt:   s.prompt,
		UserPrompt:     buildLLMJudgeUserPrompt(in),
	})
	if err != nil {
		return JudgeResult{}, err
	}

	result, err := parseJudgeResult(resp)
	if err != nil {
		return JudgeResult{}, err
	}
	return result, nil
}

func effectiveLLMJudgePrompt(prompt string) string {
	base := strings.TrimSpace(prompt)
	if base == "" {
		return defaultLLMJudgePrompt
	}
	return base + "\n\nReturn strict JSON only with this shape: {\"score\":0.0,\"rationale\":\"brief explanation\"}."
}

func buildLLMJudgeUserPrompt(in ScoreInput) string {
	var b strings.Builder
	b.WriteString("Evaluate this translation and return only the requested JSON.\n\n")
	b.WriteString("Source text:\n")
	b.WriteString(strings.TrimSpace(in.Case.Source))
	b.WriteString("\n\nTarget locale:\n")
	b.WriteString(strings.TrimSpace(in.Case.TargetLocale))
	b.WriteString("\n\nCandidate translation:\n")
	b.WriteString(strings.TrimSpace(in.Translated))

	if ctx := strings.TrimSpace(in.Case.Context); ctx != "" {
		b.WriteString("\n\nShared context:\n")
		b.WriteString(ctx)
	}
	if ref := strings.TrimSpace(in.Case.Reference); ref != "" {
		b.WriteString("\n\nReference translation (optional style guidance only):\n")
		b.WriteString(ref)
	}

	return b.String()
}

func parseJudgeResult(raw string) (JudgeResult, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if start := strings.Index(cleaned, "{"); start >= 0 {
		var payload map[string]any
		decoder := json.NewDecoder(strings.NewReader(cleaned[start:]))
		if err := decoder.Decode(&payload); err == nil {
			return payloadToJudgeResult(payload)
		}
	}

	score, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return JudgeResult{}, fmt.Errorf("parse judge response: invalid JSON score payload")
	}
	if score < 0 || score > 1 {
		return JudgeResult{}, fmt.Errorf("parse judge response: score %.3f out of range [0,1]", score)
	}
	rounded := round3(score)
	return JudgeResult{Score: &rounded}, nil
}

func parseJudgeScoreValue(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		if v < 0 || v > 1 {
			return 0, fmt.Errorf("parse judge response: score %.3f out of range [0,1]", v)
		}
		return round3(v), nil
	case string:
		score, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, fmt.Errorf("parse judge response: invalid score %q", v)
		}
		if score < 0 || score > 1 {
			return 0, fmt.Errorf("parse judge response: score %.3f out of range [0,1]", score)
		}
		return round3(score), nil
	default:
		return 0, fmt.Errorf("parse judge response: missing score")
	}
}

func payloadToJudgeResult(payload map[string]any) (JudgeResult, error) {
	score, err := parseJudgeScoreValue(payload["score"])
	if err != nil {
		return JudgeResult{}, err
	}
	result := JudgeResult{Score: &score}
	if rationale, ok := payload["rationale"].(string); ok {
		result.Rationale = strings.TrimSpace(rationale)
	}
	return result, nil
}
