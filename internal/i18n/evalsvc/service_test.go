package evalsvc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/evalset"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/evalsvc/scoring"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

type fakeReferenceScorer struct{}

func (f fakeReferenceScorer) Name() string { return "reference" }
func (f fakeReferenceScorer) ScoreReference(_ context.Context, in ScoreInput) (float64, error) {
	if in.Case.Reference == "" {
		return 0, errors.New("missing reference")
	}
	if strings.EqualFold(strings.TrimSpace(in.Case.Reference), strings.TrimSpace(in.Translated)) {
		return 1, nil
	}
	return 0, nil
}

type fakeJudgeScorer struct{}

func (f fakeJudgeScorer) Name() string { return "judge" }

func scorePtr(v float64) *float64 { return &v }

func (f fakeJudgeScorer) ScoreJudge(_ context.Context, in ScoreInput) (JudgeResult, error) {
	if strings.Contains(in.Translated, "!") {
		return JudgeResult{Score: scorePtr(0.5), Rationale: "punctuation detected"}, nil
	}
	return JudgeResult{Score: scorePtr(0.25), Rationale: "default"}, nil
}

type fakeFailingJudgeScorer struct{}

func (f fakeFailingJudgeScorer) Name() string { return "judge" }
func (f fakeFailingJudgeScorer) ScoreJudge(_ context.Context, _ ScoreInput) (JudgeResult, error) {
	return JudgeResult{}, errors.New("judge failed")
}

type fakeEmptyJudgeScorer struct{}

func (f fakeEmptyJudgeScorer) Name() string { return "judge" }
func (f fakeEmptyJudgeScorer) ScoreJudge(_ context.Context, _ ScoreInput) (JudgeResult, error) {
	return JudgeResult{}, nil
}

type fakeSecondJudgeScorer struct{}

func (f fakeSecondJudgeScorer) Name() string { return "judge_two" }
func (f fakeSecondJudgeScorer) ScoreJudge(_ context.Context, _ ScoreInput) (JudgeResult, error) {
	return JudgeResult{Score: scorePtr(0.75), Rationale: "second judge"}, nil
}

func TestRunIsDeterministicWithSeed(t *testing.T) {
	svc := newTestService()
	input := Input{
		EvalSetPath: "unused.json",
		Profiles:    []string{"default", "fast"},
		Providers:   []string{"openai"},
		Models:      []string{"model-a"},
		Prompts:     []string{"prompt A"},
		Concurrency: 3,
		Seed:        99,
	}

	report1, err := svc.Run(context.Background(), input)
	if err != nil {
		t.Fatalf("run 1: %v", err)
	}
	report2, err := svc.Run(context.Background(), input)
	if err != nil {
		t.Fatalf("run 2: %v", err)
	}

	report1.GeneratedAt = time.Time{}
	report2.GeneratedAt = time.Time{}
	zeroLatency(report1.Runs)
	zeroLatency(report2.Runs)
	zeroCaseLatency(report1.CaseSummaries)
	zeroCaseLatency(report2.CaseSummaries)
	report1.Aggregate.AverageLatencyMS = 0
	report2.Aggregate.AverageLatencyMS = 0
	report1.Aggregate.WeightedScore = 0
	report2.Aggregate.WeightedScore = 0

	if !reflect.DeepEqual(report1, report2) {
		t.Fatalf("expected deterministic report for same seed")
	}
}

func TestRunAccountsForErrors(t *testing.T) {
	svc := newTestService()
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source == "boom" {
			return "", errors.New("provider failed")
		}
		return strings.ToUpper(req.Source), nil
	}

	report, err := svc.Run(context.Background(), Input{
		EvalSetPath: "unused.json",
		Profiles:    []string{"default"},
		Providers:   []string{"openai"},
		Models:      []string{"model-a"},
		Prompts:     []string{"prompt A"},
		Seed:        1,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if report.Aggregate.TotalRuns != 2 {
		t.Fatalf("expected 2 runs, got %d", report.Aggregate.TotalRuns)
	}
	if report.Aggregate.SuccessfulRuns != 1 || report.Aggregate.FailedRuns != 1 {
		t.Fatalf("unexpected success/failure accounting: %+v", report.Aggregate)
	}

	seenErr := false
	for _, run := range report.Runs {
		if run.Error != "" {
			seenErr = true
		}
	}
	if !seenErr {
		t.Fatalf("expected at least one run error")
	}
}

func TestRunAggregatesScorersAndPersistsReport(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "report.json")
	svc := newTestService()
	svc.WithReferenceScorers(fakeReferenceScorer{}).WithJudgeScorers(fakeJudgeScorer{})

	report, err := svc.Run(context.Background(), Input{
		EvalSetPath:  "unused.json",
		Profiles:     []string{"default"},
		Providers:    []string{"openai", "anthropic"},
		Models:       []string{"model-a"},
		Prompts:      []string{"prompt A"},
		OutputPath:   outputPath,
		EvalProvider: "openai",
		EvalModel:    "gpt-4.1-mini",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if report.Aggregate.TotalRuns != 4 {
		t.Fatalf("expected 4 total runs, got %d", report.Aggregate.TotalRuns)
	}
	if report.Aggregate.AverageScoreByName["reference"] != 1 {
		t.Fatalf("unexpected reference aggregate score: %+v", report.Aggregate.AverageScoreByName)
	}
	if report.LLMEvaluation == nil || report.LLMEvaluation.AggregateScore == nil || *report.LLMEvaluation.AggregateScore != 0.25 {
		t.Fatalf("expected llm aggregate score, got %+v", report.LLMEvaluation)
	}
	if report.LLMEvaluation.AverageScoreByName["judge"] != 0.25 {
		t.Fatalf("unexpected llm judge aggregate score: %+v", report.LLMEvaluation)
	}
	if report.LLMEvaluation.Provider != "openai" || report.LLMEvaluation.Model != "gpt-4.1-mini" {
		t.Fatalf("unexpected llm metadata: %+v", report.LLMEvaluation)
	}
	for _, run := range report.Runs {
		if _, ok := run.JudgeResults["judge"]; !ok {
			t.Fatalf("expected judge results on each run: %+v", run)
		}
	}
	if len(report.CaseSummaries) != 2 {
		t.Fatalf("expected 2 case summaries, got %d", len(report.CaseSummaries))
	}

	if len(svc.writes) != 1 || svc.writes[0] != outputPath {
		t.Fatalf("expected report written once to output path, got %+v", svc.writes)
	}
}

func TestRunJudgeScoringDisabledByDefault(t *testing.T) {
	svc := newTestService()
	svc.WithJudgeScorers(fakeJudgeScorer{})

	report, err := svc.Run(context.Background(), Input{
		EvalSetPath: "unused.json",
		Profiles:    []string{"default"},
		Providers:   []string{"openai"},
		Models:      []string{"model-a"},
		Prompts:     []string{"prompt A"},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, run := range report.Runs {
		if _, ok := run.Scores["judge"]; ok {
			t.Fatalf("expected judge scorer to be disabled by default")
		}
		if len(run.JudgeResults) > 0 {
			t.Fatalf("expected judge results to be disabled by default")
		}
	}
}

func TestRunAllowsMissingReferenceInLLMMode(t *testing.T) {
	svc := newTestService()
	svc.loadEvalset = func(_ string) (*evalset.Dataset, error) {
		return &evalset.Dataset{Cases: []evalset.Case{{ID: "a", Source: "hello", TargetLocale: "fr"}}}, nil
	}
	svc.WithJudgeScorers(fakeJudgeScorer{})

	report, err := svc.Run(context.Background(), Input{
		EvalSetPath:  "unused.json",
		Profiles:     []string{"default"},
		Providers:    []string{"openai"},
		Models:       []string{"model-a"},
		Prompts:      []string{"prompt A"},
		EvalProvider: "openai",
		EvalModel:    "judge-model",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.LLMEvaluation == nil || report.LLMEvaluation.AggregateScore == nil {
		t.Fatalf("expected llm evaluation aggregate: %+v", report.LLMEvaluation)
	}
}

func TestRunRecordsJudgeFailuresWithoutFailingReport(t *testing.T) {
	svc := newTestService()
	svc.WithJudgeScorers(fakeFailingJudgeScorer{})

	report, err := svc.Run(context.Background(), Input{
		EvalSetPath:  "unused.json",
		Profiles:     []string{"default"},
		Providers:    []string{"openai"},
		Models:       []string{"model-a"},
		Prompts:      []string{"prompt A"},
		EvalProvider: "openai",
		EvalModel:    "judge-model",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.LLMEvaluation == nil {
		t.Fatalf("expected llm evaluation metadata")
	}
	if report.LLMEvaluation.AggregateScore != nil {
		t.Fatalf("expected no aggregate score on total judge failure: %+v", report.LLMEvaluation)
	}
	if report.LLMEvaluation.FailedJudges != len(report.Runs) {
		t.Fatalf("expected failed judge count to match runs: %+v", report.LLMEvaluation)
	}
	for _, run := range report.Runs {
		if run.JudgeResults["judge"].Error == "" {
			t.Fatalf("expected judge failure recorded on run: %+v", run)
		}
	}
}

func TestRunNormalizesEmptyJudgeResultToFailure(t *testing.T) {
	svc := newTestService()
	svc.WithJudgeScorers(fakeEmptyJudgeScorer{})

	report, err := svc.Run(context.Background(), Input{
		EvalSetPath:  "unused.json",
		Profiles:     []string{"default"},
		Providers:    []string{"openai"},
		Models:       []string{"model-a"},
		Prompts:      []string{"prompt A"},
		EvalProvider: "openai",
		EvalModel:    "judge-model",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.LLMEvaluation == nil {
		t.Fatalf("expected llm evaluation metadata")
	}
	if report.LLMEvaluation.FailedJudges != len(report.Runs) {
		t.Fatalf("expected empty judge results counted as failures: %+v", report.LLMEvaluation)
	}
	for _, run := range report.Runs {
		if got := run.JudgeResults["judge"].Error; got != "judge returned no score" {
			t.Fatalf("expected normalized judge error, got %+v", run)
		}
	}
}

func TestRunTracksSkippedRunsWhenTranslationFailsInLLMMode(t *testing.T) {
	svc := newTestService()
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source == "boom" {
			return "", errors.New("provider failed")
		}
		return strings.ToUpper(req.Source), nil
	}
	svc.WithJudgeScorers(fakeJudgeScorer{})

	report, err := svc.Run(context.Background(), Input{
		EvalSetPath:  "unused.json",
		Profiles:     []string{"default"},
		Providers:    []string{"openai"},
		Models:       []string{"model-a"},
		Prompts:      []string{"prompt A"},
		EvalProvider: "openai",
		EvalModel:    "judge-model",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.LLMEvaluation == nil {
		t.Fatalf("expected llm evaluation metadata")
	}
	if report.LLMEvaluation.SkippedRuns != 1 {
		t.Fatalf("expected 1 skipped run, got %+v", report.LLMEvaluation)
	}
	if report.LLMEvaluation.SuccessfulJudges != 1 || report.LLMEvaluation.FailedJudges != 0 {
		t.Fatalf("unexpected llm counters: %+v", report.LLMEvaluation)
	}
	if report.LLMEvaluation.AggregateScore == nil || *report.LLMEvaluation.AggregateScore != 0.25 {
		t.Fatalf("expected llm aggregate score from successful run, got %+v", report.LLMEvaluation)
	}
}

func TestAggregateLLMEvaluationCountsJudgeCallsAcrossScorers(t *testing.T) {
	svc := newTestService()
	svc.WithJudgeScorers(fakeJudgeScorer{}, fakeSecondJudgeScorer{})

	report, err := svc.Run(context.Background(), Input{
		EvalSetPath:  "unused.json",
		Profiles:     []string{"default"},
		Providers:    []string{"openai"},
		Models:       []string{"model-a"},
		Prompts:      []string{"prompt A"},
		EvalProvider: "openai",
		EvalModel:    "judge-model",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.LLMEvaluation == nil {
		t.Fatalf("expected llm evaluation metadata")
	}
	if report.LLMEvaluation.SuccessfulJudges != len(report.Runs)*2 {
		t.Fatalf("expected judge call count across scorers, got %+v", report.LLMEvaluation)
	}
}

func TestInputValidateRejectsPartialEvalConfig(t *testing.T) {
	err := (Input{EvalSetPath: "set.json", EvalProvider: "openai"}).Validate()
	if err == nil || !strings.Contains(err.Error(), "must be set together") {
		t.Fatalf("expected paired evaluator flag error, got %v", err)
	}
}

func TestInputValidateRejectsEvalPromptWithoutProviderAndModel(t *testing.T) {
	err := (Input{EvalSetPath: "set.json", EvalPrompt: "judge this"}).Validate()
	if err == nil || !strings.Contains(err.Error(), "required when using evaluator prompt overrides") {
		t.Fatalf("expected evaluator prompt validation error, got %v", err)
	}
}

func TestAggregateLLMEvaluationDeterministic(t *testing.T) {
	in := Input{EvalSetPath: "set.json", EvalProvider: "openai", EvalModel: "judge-model"}
	runs := []RunResult{
		{JudgeResults: map[string]JudgeResult{"judge": {Score: scorePtr(0.2)}}},
		{JudgeResults: map[string]JudgeResult{"judge": {Score: scorePtr(0.4)}}},
		{JudgeResults: map[string]JudgeResult{"judge": {Error: "boom"}}},
		{Error: "translation failed"},
	}
	got := aggregateLLMEvaluation(in, runs)
	if got == nil || got.AggregateScore == nil || *got.AggregateScore != 0.3 {
		t.Fatalf("unexpected llm aggregate: %+v", got)
	}
	if got.SuccessfulJudges != 2 || got.FailedJudges != 1 || got.SkippedRuns != 1 {
		t.Fatalf("unexpected llm counters: %+v", got)
	}
}

func TestParseJudgeResult(t *testing.T) {
	got, err := parseJudgeResult("```json\n{\"score\":0.83,\"rationale\":\"good\"}\n```")
	if err != nil {
		t.Fatalf("parse judge result: %v", err)
	}
	if got.Score == nil || *got.Score != 0.83 || got.Rationale != "good" {
		t.Fatalf("unexpected parsed judge result: %+v", got)
	}
}

func TestParseJudgeResultWithTrailingBraces(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantScore float64
		wantErr   bool
	}{
		{
			name:      "trailing text with extra closing braces",
			input:     `{"score":0.8,"rationale":"good"} some text }}`,
			wantScore: 0.8,
			wantErr:   false,
		},
		{
			name:      "markdown code block with trailing braces",
			input:     "```json\n{\"score\":0.75,\"rationale\":\"ok\"}\n```\n\nSome explanation }\n}",
			wantScore: 0.75,
			wantErr:   false,
		},
		{
			name:      "json followed by multiple closing braces",
			input:     `{"score":0.5,"rationale":"average"}}}}`,
			wantScore: 0.5,
			wantErr:   false,
		},
		{
			name:      "nested braces in string value with trailing braces",
			input:     `{"score":0.9,"rationale":"{\"nested\":true} is good"} extra }}}`,
			wantScore: 0.9,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJudgeResult(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseJudgeResult() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.Score == nil || *got.Score != tt.wantScore {
					t.Fatalf("parseJudgeResult() = %+v, want score %.2f", got, tt.wantScore)
				}
			}
		})
	}
}

func TestLLMJudgeScorerUsesOriginalSourceAndCustomPrompt(t *testing.T) {
	var gotReq translator.Request
	scorer := NewLLMJudgeScorer("openai", "gpt-4.1-mini", "judge prompt", func(_ context.Context, req translator.Request) (string, error) {
		gotReq = req
		return `{"score":0.8,"rationale":"ok"}`, nil
	})

	result, err := scorer.ScoreJudge(context.Background(), ScoreInput{
		Case: evalset.Case{
			Source:       "Hello",
			TargetLocale: "fr",
			Context:      "homepage headline",
			Reference:    "Bonjour",
		},
		Translated: "Salut",
	})
	if err != nil {
		t.Fatalf("score judge: %v", err)
	}
	if gotReq.Source != "Hello" {
		t.Fatalf("expected original source in request, got %+v", gotReq)
	}
	if !strings.Contains(gotReq.UserPrompt, "Candidate translation:\nSalut") {
		t.Fatalf("expected candidate translation in user prompt, got %q", gotReq.UserPrompt)
	}
	if !strings.Contains(gotReq.SystemPrompt, "judge prompt") {
		t.Fatalf("expected custom judge system prompt, got %q", gotReq.SystemPrompt)
	}
	if result.Score == nil || *result.Score != 0.8 {
		t.Fatalf("unexpected judge result: %+v", result)
	}
}

type testService struct {
	*Service
	writes []string
}

func newTestService() *testService {
	now := time.Unix(1700000000, 0).UTC()
	dataset := &evalset.Dataset{Cases: []evalset.Case{
		{ID: "a", Source: "hello", TargetLocale: "fr", Reference: "HELLO"},
		{ID: "b", Source: "boom", TargetLocale: "fr", Reference: "BOOM"},
	}}

	t := &testService{}
	t.Service = &Service{
		loadEvalset: func(_ string) (*evalset.Dataset, error) {
			return dataset, nil
		},
		translate: func(_ context.Context, req translator.Request) (string, error) {
			return strings.ToUpper(req.Source), nil
		},
		writeFile: func(path string, _ []byte, _ os.FileMode) error {
			t.writes = append(t.writes, path)
			return nil
		},
		now:    func() time.Time { return now },
		numCPU: func() int { return 2 },
	}

	return t
}

func TestBuildExperimentsUsesCartesianProduct(t *testing.T) {
	experiments, err := buildExperiments(Input{
		Profiles:  []string{"p1", "p2"},
		Providers: []string{"openai", "anthropic"},
		Models:    []string{"m1"},
		Prompts:   []string{"x", "y"},
	})
	if err != nil {
		t.Fatalf("build experiments: %v", err)
	}
	if len(experiments) != 8 {
		t.Fatalf("expected 8 experiments, got %d", len(experiments))
	}
	if experiments[0].id == "" {
		t.Fatalf("expected experiment IDs to be populated")
	}
}

func TestResolveWorkerCount(t *testing.T) {
	if got := resolveWorkerCount(5, func() int { return 1 }); got != 5 {
		t.Fatalf("expected explicit worker count, got %d", got)
	}
	if got := resolveWorkerCount(0, func() int { return 0 }); got != 1 {
		t.Fatalf("expected fallback to 1, got %d", got)
	}
}

func TestExecuteSingleCapturesArtifacts(t *testing.T) {
	var gotReq translator.Request
	svc := &Service{translate: func(_ context.Context, req translator.Request) (string, error) {
		gotReq = req
		return fmt.Sprintf("%s->%s", req.Source, req.TargetLanguage), nil
	}, qualityEvaluator: scoring.NewEvaluator()}

	run := svc.executeSingle(context.Background(), evalset.Case{ID: "case-1", Source: "hello", TargetLocale: "fr"}, experiment{
		id:       "exp-1",
		profile:  "default",
		provider: "openai",
		model:    "m1",
		prompt:   "p1",
	}, nil, nil)

	if run.Translated == "" || run.LatencyMS < 0 {
		t.Fatalf("expected translation artifacts, got %+v", run)
	}
	if run.Profile != "default" || run.Provider != "openai" || run.Model != "m1" || run.Prompt != "p1" {
		t.Fatalf("expected experiment identifiers to be captured, got %+v", run)
	}
	if gotReq.SystemPrompt != "p1" {
		t.Fatalf("expected eval experiment prompt routed to system prompt, got %q", gotReq.SystemPrompt)
	}
	if gotReq.UserPrompt != "" {
		t.Fatalf("expected no custom eval user prompt by default, got %q", gotReq.UserPrompt)
	}
}

func zeroLatency(runs []RunResult) {
	for i := range runs {
		runs[i].LatencyMS = 0
	}
}

func zeroCaseLatency(summaries []CaseSummary) {
	for i := range summaries {
		summaries[i].AverageLatencyMS = 0
	}
}
