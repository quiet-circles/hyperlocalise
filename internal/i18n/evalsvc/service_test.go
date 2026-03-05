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
func (f fakeJudgeScorer) ScoreJudge(_ context.Context, in ScoreInput) (float64, error) {
	if strings.Contains(in.Translated, "!") {
		return 0.5, nil
	}
	return 0.25, nil
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
		EvalSetPath:    "unused.json",
		Profiles:       []string{"default"},
		Providers:      []string{"openai", "anthropic"},
		Models:         []string{"model-a"},
		Prompts:        []string{"prompt A"},
		OutputPath:     outputPath,
		EnableLLMJudge: true,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if report.Aggregate.TotalRuns != 4 {
		t.Fatalf("expected 4 total runs, got %d", report.Aggregate.TotalRuns)
	}
	if _, ok := report.Aggregate.AverageScoreByName["reference"]; ok {
		t.Fatalf("expected reference scorer to be disabled in llm mode: %+v", report.Aggregate.AverageScoreByName)
	}
	if report.Aggregate.AverageScoreByName["judge"] != 0.25 {
		t.Fatalf("unexpected judge aggregate score: %+v", report.Aggregate.AverageScoreByName)
	}
	if report.Aggregate.WeightedScore != 0.25 {
		t.Fatalf("expected weighted score from llm judge, got %+v", report.Aggregate.WeightedScore)
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
	svc := &Service{translate: func(_ context.Context, req translator.Request) (string, error) {
		return fmt.Sprintf("%s->%s", req.Source, req.TargetLanguage), nil
	}}

	run := svc.executeSingle(context.Background(), evalset.Case{ID: "case-1", Source: "hello", TargetLocale: "fr"}, experiment{
		id:       "exp-1",
		profile:  "default",
		provider: "openai",
		model:    "m1",
		prompt:   "p1",
	}, nil, false)

	if run.Translated == "" || run.LatencyMS < 0 {
		t.Fatalf("expected translation artifacts, got %+v", run)
	}
	if run.Profile != "default" || run.Provider != "openai" || run.Model != "m1" || run.Prompt != "p1" {
		t.Fatalf("expected experiment identifiers to be captured, got %+v", run)
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

func TestValidationEvalProviderModelPaired(t *testing.T) {
	svc := newTestService()

	// Test: EvalProvider without EvalModel should fail
	_, err := svc.Run(context.Background(), Input{
		EvalSetPath:  "unused.json",
		EvalProvider: "openai",
	})
	if err == nil {
		t.Fatalf("expected error when EvalProvider is set without EvalModel")
	}
	if !strings.Contains(err.Error(), "eval provider and model must be both set or both empty") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test: EvalModel without EvalProvider should fail
	_, err = svc.Run(context.Background(), Input{
		EvalSetPath: "unused.json",
		EvalModel:   "gpt-4",
	})
	if err == nil {
		t.Fatalf("expected error when EvalModel is set without EvalProvider")
	}
	if !strings.Contains(err.Error(), "eval provider and model must be both set or both empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidationEvalPromptRequiresProviderModel(t *testing.T) {
	svc := newTestService()

	// Test: EvalPrompt without provider/model should fail
	_, err := svc.Run(context.Background(), Input{
		EvalSetPath: "unused.json",
		EvalPrompt:  "Judge this translation: {{translation}}",
	})
	if err == nil {
		t.Fatalf("expected error when EvalPrompt is set without provider/model")
	}
	if !strings.Contains(err.Error(), "eval prompt requires both eval provider and eval model") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.Run(context.Background(), Input{
		EvalSetPath:  "unused.json",
		Profiles:     []string{"default"},
		Providers:    []string{"openai"},
		Models:       []string{"model-a"},
		Prompts:      []string{"prompt A"},
		EvalProvider: "openai",
		EvalModel:    "gpt-4",
		EvalPrompt:   "Judge this translation: {{translation}}",
	})
	if err != nil {
		t.Fatalf("expected paired eval provider/model/prompt to pass validation, got %v", err)
	}
}

func TestLLMModeEnabledWithEvalProviderModel(t *testing.T) {
	svc := newTestService()
	svc.WithJudgeScorers(fakeJudgeScorer{})

	// Without eval provider/model, judge scoring should be disabled by default
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

	// Verify judge scores are not present (LLM mode not enabled)
	for _, run := range report.Runs {
		if _, ok := run.Scores["judge"]; ok {
			t.Fatalf("expected judge scorer to be disabled when no eval provider/model")
		}
	}

	// With eval provider/model, LLM mode should be enabled
	report, err = svc.Run(context.Background(), Input{
		EvalSetPath:  "unused.json",
		Profiles:     []string{"default"},
		Providers:    []string{"openai"},
		Models:       []string{"model-a"},
		Prompts:      []string{"prompt A"},
		EvalProvider: "openai",
		EvalModel:    "gpt-4",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Verify judge scores are present (LLM mode enabled)
	foundJudgeScore := false
	for _, run := range report.Runs {
		if _, ok := run.Scores["judge"]; ok {
			foundJudgeScore = true
			break
		}
	}
	if !foundJudgeScore {
		t.Fatalf("expected judge scorer to be enabled when eval provider/model are set")
	}
}

func TestLLModeUsesJudgeScoringOnly(t *testing.T) {
	svc := newTestService()
	svc.WithJudgeScorers(fakeJudgeScorer{})

	// Run in LLM mode
	report, err := svc.Run(context.Background(), Input{
		EvalSetPath:  "unused.json",
		Profiles:     []string{"default"},
		Providers:    []string{"openai"},
		Models:       []string{"model-a"},
		Prompts:      []string{"prompt A"},
		EvalProvider: "openai",
		EvalModel:    "gpt-4",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// In LLM mode, weighted score should come from judge scores
	// The fake judge scorer returns 0.25 for translations without "!" and 0.5 with "!"
	// Our test cases: "hello" -> "HELLO" and "boom" -> "BOOM"
	// Neither contains "!", so judge score should be 0.25
	if report.Aggregate.WeightedScore != 0.25 {
		t.Fatalf("expected weighted score from judge (0.25), got %f", report.Aggregate.WeightedScore)
	}

	// Verify that Quality.WeightedAggregate matches the judge score
	for _, run := range report.Runs {
		if run.Error == "" && run.Quality.WeightedAggregate == 0 {
			t.Fatalf("expected non-zero quality weighted aggregate in LLM mode, got %f", run.Quality.WeightedAggregate)
		}
	}
}

func TestLLMModeUsesBuiltInJudgeFromEvalProviderModel(t *testing.T) {
	svc := newTestService()
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Model == "judge-model" {
			return "0.88", nil
		}
		return strings.ToUpper(req.Source), nil
	}

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

	if report.Aggregate.AverageScoreByName["judge"] != 0.88 {
		t.Fatalf("expected built-in judge score in report, got %+v", report.Aggregate.AverageScoreByName)
	}
	if report.Aggregate.WeightedScore != 0.88 {
		t.Fatalf("expected weighted score from built-in judge, got %v", report.Aggregate.WeightedScore)
	}
}
