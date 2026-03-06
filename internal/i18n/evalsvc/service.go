package evalsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/evalset"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/evalsvc/scoring"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

// Input controls evaluation execution.
type Input struct {
	EvalSetPath  string
	Profiles     []string
	Providers    []string
	Models       []string
	Prompts      []string
	Concurrency  int
	Seed         int64
	OutputPath   string
	EvalProvider string
	EvalModel    string
	EvalPrompt   string
}

// Validate checks input semantics before execution.
func (in Input) Validate() error {
	if strings.TrimSpace(in.EvalSetPath) == "" {
		return fmt.Errorf("--eval-set is required")
	}

	provider := strings.TrimSpace(in.EvalProvider)
	model := strings.TrimSpace(in.EvalModel)
	prompt := strings.TrimSpace(in.EvalPrompt)

	if (provider == "") != (model == "") {
		return fmt.Errorf("--eval-provider and --eval-model must be set together")
	}
	if prompt != "" && (provider == "" || model == "") {
		return fmt.Errorf("--eval-provider and --eval-model are required when using evaluator prompt overrides")
	}

	return nil
}

// LLMEvaluationEnabled reports whether model-based evaluation is configured.
func (in Input) LLMEvaluationEnabled() bool {
	return strings.TrimSpace(in.EvalProvider) != "" && strings.TrimSpace(in.EvalModel) != ""
}

// Aggregate summarizes evaluation totals.
type Aggregate struct {
	TotalRuns          int                `json:"totalRuns"`
	SuccessfulRuns     int                `json:"successfulRuns"`
	FailedRuns         int                `json:"failedRuns"`
	AverageLatencyMS   float64            `json:"averageLatencyMs"`
	AverageScoreByName map[string]float64 `json:"averageScoreByName,omitempty"`
	WeightedScore      float64            `json:"weightedScore,omitempty"`
	HardFailCounts     map[string]int     `json:"hardFailCounts,omitempty"`
}

// Report is the full result payload for an eval execution.
type Report struct {
	GeneratedAt   time.Time      `json:"generatedAt"`
	Input         Input          `json:"input"`
	Aggregate     Aggregate      `json:"aggregate"`
	LLMEvaluation *LLMEvaluation `json:"llmEvaluation,omitempty"`
	Runs          []RunResult    `json:"runs"`
	CaseSummaries []CaseSummary  `json:"caseSummaries"`
}

// LLMEvaluation summarizes the LLM judge lane.
// SuccessfulJudges and FailedJudges count individual judge calls, not runs.
type LLMEvaluation struct {
	Enabled            bool               `json:"enabled"`
	Provider           string             `json:"provider,omitempty"`
	Model              string             `json:"model,omitempty"`
	Prompt             string             `json:"prompt,omitempty"`
	AggregateScore     *float64           `json:"aggregateScore,omitempty"`
	AverageScoreByName map[string]float64 `json:"averageScoreByName,omitempty"`
	SuccessfulJudges   int                `json:"successfulJudges,omitempty"`
	FailedJudges       int                `json:"failedJudges,omitempty"`
	SkippedRuns        int                `json:"skippedRuns,omitempty"`
}

// JudgeResult stores one judge outcome for a run.
type JudgeResult struct {
	Score     *float64 `json:"score,omitempty"`
	Rationale string   `json:"rationale,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// RunResult captures one case/experiment translation attempt.
type RunResult struct {
	CaseID       string                 `json:"caseId"`
	ExperimentID string                 `json:"experimentId"`
	Profile      string                 `json:"profile"`
	Provider     string                 `json:"provider"`
	Model        string                 `json:"model"`
	Prompt       string                 `json:"prompt"`
	Translated   string                 `json:"translated,omitempty"`
	LatencyMS    float64                `json:"latencyMs"`
	Error        string                 `json:"error,omitempty"`
	Scores       map[string]float64     `json:"scores,omitempty"`
	JudgeResults map[string]JudgeResult `json:"judgeResults,omitempty"`
	Quality      scoring.Result         `json:"quality"`
}

// CaseSummary aggregates all runs for a single case.
type CaseSummary struct {
	CaseID             string             `json:"caseId"`
	RunCount           int                `json:"runCount"`
	SuccessfulRuns     int                `json:"successfulRuns"`
	FailedRuns         int                `json:"failedRuns"`
	AverageLatencyMS   float64            `json:"averageLatencyMs"`
	AverageScoreByName map[string]float64 `json:"averageScoreByName,omitempty"`
	WeightedScore      float64            `json:"weightedScore,omitempty"`
	HardFailCounts     map[string]int     `json:"hardFailCounts,omitempty"`
}

// ScoreInput is passed to scorer implementations.
type ScoreInput struct {
	Case       evalset.Case
	Request    translator.Request
	Translated string
}

// ReferenceScorer computes a score against references.
type ReferenceScorer interface {
	Name() string
	ScoreReference(ctx context.Context, in ScoreInput) (float64, error)
}

// JudgeScorer computes a score via model-as-judge or similar heuristics.
type JudgeScorer interface {
	Name() string
	ScoreJudge(ctx context.Context, in ScoreInput) (JudgeResult, error)
}

type experiment struct {
	id       string
	profile  string
	provider string
	model    string
	prompt   string
}

type Service struct {
	loadEvalset func(path string) (*evalset.Dataset, error)
	translate   func(ctx context.Context, req translator.Request) (string, error)
	writeFile   func(path string, content []byte, perm os.FileMode) error
	now         func() time.Time
	numCPU      func() int

	referenceScorers []ReferenceScorer
	judgeScorers     []JudgeScorer
	qualityEvaluator *scoring.Evaluator
}

func New() *Service {
	return &Service{
		loadEvalset:      evalset.Load,
		translate:        translator.Translate,
		writeFile:        os.WriteFile,
		now:              func() time.Time { return time.Now().UTC() },
		numCPU:           runtime.NumCPU,
		qualityEvaluator: scoring.NewEvaluator(),
	}
}

func Run(ctx context.Context, in Input) (Report, error) {
	return New().Run(ctx, in)
}

func (s *Service) WithReferenceScorers(scorers ...ReferenceScorer) *Service {
	s.referenceScorers = append([]ReferenceScorer(nil), scorers...)
	return s
}

func (s *Service) WithJudgeScorers(scorers ...JudgeScorer) *Service {
	s.judgeScorers = append([]JudgeScorer(nil), scorers...)
	return s
}

func (s *Service) Run(ctx context.Context, in Input) (Report, error) {
	if err := in.Validate(); err != nil {
		return Report{}, err
	}

	dataset, err := s.loadEvalset(in.EvalSetPath)
	if err != nil {
		return Report{}, fmt.Errorf("load evalset: %w", err)
	}

	experiments, err := buildExperiments(in)
	if err != nil {
		return Report{}, err
	}

	cases := append([]evalset.Case(nil), dataset.Cases...)
	if in.Seed != 0 {
		r := rand.New(rand.NewSource(in.Seed))
		r.Shuffle(len(cases), func(i, j int) {
			cases[i], cases[j] = cases[j], cases[i]
		})
	}

	workerCount := resolveWorkerCount(in.Concurrency, s.numCPU)
	referenceScorers := append([]ReferenceScorer(nil), s.referenceScorers...)
	judgeScorers := s.resolveJudgeScorers(in)

	// Initialize qualityEvaluator before spawning concurrent workers.
	if s.qualityEvaluator == nil {
		s.qualityEvaluator = scoring.NewEvaluator()
	}

	runs, err := s.execute(ctx, cases, experiments, referenceScorers, judgeScorers, workerCount)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		GeneratedAt: s.now(),
		Input:       in,
		Runs:        runs,
	}
	report.Aggregate = aggregateRuns(runs)
	report.LLMEvaluation = aggregateLLMEvaluation(in, runs)
	report.CaseSummaries = summarizeCases(runs)

	if in.OutputPath != "" {
		encoded, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			return Report{}, fmt.Errorf("marshal report: %w", marshalErr)
		}
		if writeErr := s.writeFile(in.OutputPath, encoded, 0o644); writeErr != nil {
			return Report{}, fmt.Errorf("write report: %w", writeErr)
		}
	}

	return report, nil
}

func (s *Service) resolveJudgeScorers(in Input) []JudgeScorer {
	if !in.LLMEvaluationEnabled() {
		return nil
	}
	if len(s.judgeScorers) > 0 {
		return append([]JudgeScorer(nil), s.judgeScorers...)
	}

	return []JudgeScorer{NewLLMJudgeScorer(in.EvalProvider, in.EvalModel, in.EvalPrompt, s.translate)}
}

func resolveWorkerCount(requested int, numCPU func() int) int {
	if requested > 0 {
		return requested
	}
	workers := numCPU()
	if workers < 1 {
		return 1
	}
	return workers
}

func buildExperiments(in Input) ([]experiment, error) {
	profiles := normalizedOrDefault(in.Profiles, "default")
	providers := normalizedOrDefault(in.Providers, translator.ProviderOpenAI)
	models := normalizedOrDefault(in.Models, "gpt-4.1-mini")
	prompts := normalizedOrDefault(in.Prompts, "Translate to {{target}}: {{input}}")

	experiments := make([]experiment, 0, len(profiles)*len(providers)*len(models)*len(prompts))
	for _, profile := range profiles {
		for _, provider := range providers {
			for _, model := range models {
				for _, prompt := range prompts {
					experiments = append(experiments, experiment{
						id:       fmt.Sprintf("%s|%s|%s|%s", profile, provider, model, prompt),
						profile:  profile,
						provider: provider,
						model:    model,
						prompt:   prompt,
					})
				}
			}
		}
	}
	if len(experiments) == 0 {
		return nil, fmt.Errorf("build experiments: no experiment variants resolved")
	}

	return experiments, nil
}

func normalizedOrDefault(values []string, fallback string) []string {
	if len(values) == 0 {
		return []string{fallback}
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return []string{fallback}
	}
	return out
}

func (s *Service) execute(ctx context.Context, cases []evalset.Case, experiments []experiment, referenceScorers []ReferenceScorer, judgeScorers []JudgeScorer, workerCount int) ([]RunResult, error) {
	type job struct {
		tc  evalset.Case
		exp experiment
	}

	jobs := make(chan job)
	results := make(chan RunResult)

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				results <- s.executeSingle(ctx, item.tc, item.exp, referenceScorers, judgeScorers)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, tc := range cases {
			for _, exp := range experiments {
				jobs <- job{tc: tc, exp: exp}
			}
		}
	}()

	expected := len(cases) * len(experiments)
	runs := make([]RunResult, 0, expected)
	for range expected {
		runs = append(runs, <-results)
	}

	wg.Wait()
	close(results)

	sort.Slice(runs, func(i, j int) bool {
		if runs[i].CaseID != runs[j].CaseID {
			return runs[i].CaseID < runs[j].CaseID
		}
		return runs[i].ExperimentID < runs[j].ExperimentID
	})

	return runs, nil
}

func (s *Service) executeSingle(ctx context.Context, tc evalset.Case, exp experiment, referenceScorers []ReferenceScorer, judgeScorers []JudgeScorer) RunResult {
	req := translator.Request{
		Source:         tc.Source,
		TargetLanguage: tc.TargetLocale,
		Context:        tc.Context,
		ModelProvider:  exp.provider,
		Model:          exp.model,
		SystemPrompt:   exp.prompt,
	}
	start := time.Now()
	translated, err := s.translate(ctx, req)
	latency := time.Since(start)

	run := RunResult{
		CaseID:       tc.ID,
		ExperimentID: exp.id,
		Profile:      exp.profile,
		Provider:     exp.provider,
		Model:        exp.model,
		Prompt:       exp.prompt,
		Translated:   translated,
		LatencyMS:    float64(latency.Microseconds()) / 1000,
	}

	if err != nil {
		run.Error = err.Error()
		run.Quality = s.qualityEvaluator.Evaluate(tc.Source, "", tc.Reference)
		return run
	}
	run.Quality = s.qualityEvaluator.Evaluate(tc.Source, translated, tc.Reference)

	scoreInput := ScoreInput{Case: tc, Request: req, Translated: translated}
	for _, scorer := range referenceScorers {
		score, scoreErr := scorer.ScoreReference(ctx, scoreInput)
		if scoreErr != nil {
			continue
		}
		if run.Scores == nil {
			run.Scores = map[string]float64{}
		}
		run.Scores[scorer.Name()] = score
	}
	for _, scorer := range judgeScorers {
		judgeResult, scoreErr := scorer.ScoreJudge(ctx, scoreInput)
		if run.JudgeResults == nil {
			run.JudgeResults = map[string]JudgeResult{}
		}
		if scoreErr != nil {
			run.JudgeResults[scorer.Name()] = JudgeResult{Error: scoreErr.Error()}
			continue
		}
		if judgeResult.Score == nil && strings.TrimSpace(judgeResult.Error) == "" {
			judgeResult.Error = "judge returned no score"
		}
		run.JudgeResults[scorer.Name()] = judgeResult
	}

	return run
}

func aggregateLLMEvaluation(in Input, runs []RunResult) *LLMEvaluation {
	if !in.LLMEvaluationEnabled() {
		return nil
	}

	llm := &LLMEvaluation{
		Enabled:  true,
		Provider: strings.TrimSpace(in.EvalProvider),
		Model:    strings.TrimSpace(in.EvalModel),
		Prompt:   effectiveLLMJudgePrompt(strings.TrimSpace(in.EvalPrompt)),
	}

	total := 0.0
	totalCount := 0
	scoreSums := map[string]float64{}
	scoreCounts := map[string]int{}
	for _, run := range runs {
		if strings.TrimSpace(run.Error) != "" {
			llm.SkippedRuns++
			continue
		}
		for name, result := range run.JudgeResults {
			if result.Score != nil {
				score := *result.Score
				total += score
				totalCount++
				llm.SuccessfulJudges++
				scoreSums[name] += score
				scoreCounts[name]++
				continue
			}
			if strings.TrimSpace(result.Error) != "" {
				llm.FailedJudges++
			}
		}
	}

	if totalCount > 0 {
		aggregateScore := round3(total / float64(totalCount))
		llm.AggregateScore = &aggregateScore
		llm.AverageScoreByName = map[string]float64{}
		for name, sum := range scoreSums {
			llm.AverageScoreByName[name] = round3(sum / float64(scoreCounts[name]))
		}
	}

	return llm
}

func aggregateRuns(runs []RunResult) Aggregate {
	agg := Aggregate{TotalRuns: len(runs)}
	if len(runs) == 0 {
		return agg
	}

	totalLatency := 0.0
	totalWeighted := 0.0
	scoreSums := map[string]float64{}
	scoreCounts := map[string]int{}
	hardFailCounts := map[string]int{}
	for _, run := range runs {
		totalLatency += run.LatencyMS
		totalWeighted += run.Quality.WeightedAggregate
		if run.Error != "" {
			agg.FailedRuns++
		} else {
			agg.SuccessfulRuns++
		}
		for _, cat := range run.Quality.HardFails {
			hardFailCounts[cat]++
		}
		for name, score := range run.Scores {
			scoreSums[name] += score
			scoreCounts[name]++
		}
	}

	agg.AverageLatencyMS = round3(totalLatency / float64(len(runs)))
	agg.WeightedScore = round3(totalWeighted / float64(len(runs)))
	if len(scoreSums) > 0 {
		agg.AverageScoreByName = map[string]float64{}
		for name, sum := range scoreSums {
			agg.AverageScoreByName[name] = round3(sum / float64(scoreCounts[name]))
		}
	}
	if len(hardFailCounts) > 0 {
		agg.HardFailCounts = hardFailCounts
	}

	return agg
}

func summarizeCases(runs []RunResult) []CaseSummary {
	byCase := map[string][]RunResult{}
	for _, run := range runs {
		byCase[run.CaseID] = append(byCase[run.CaseID], run)
	}

	caseIDs := make([]string, 0, len(byCase))
	for caseID := range byCase {
		caseIDs = append(caseIDs, caseID)
	}
	sort.Strings(caseIDs)

	summaries := make([]CaseSummary, 0, len(caseIDs))
	for _, caseID := range caseIDs {
		list := byCase[caseID]
		summary := CaseSummary{CaseID: caseID, RunCount: len(list)}

		totalLatency := 0.0
		totalWeighted := 0.0
		scoreSums := map[string]float64{}
		scoreCounts := map[string]int{}
		hardFailCounts := map[string]int{}
		for _, run := range list {
			totalLatency += run.LatencyMS
			totalWeighted += run.Quality.WeightedAggregate
			if run.Error != "" {
				summary.FailedRuns++
			} else {
				summary.SuccessfulRuns++
			}
			for name, score := range run.Scores {
				scoreSums[name] += score
				scoreCounts[name]++
			}
			for _, cat := range run.Quality.HardFails {
				hardFailCounts[cat]++
			}
		}

		summary.AverageLatencyMS = round3(totalLatency / float64(len(list)))
		summary.WeightedScore = round3(totalWeighted / float64(len(list)))
		if len(scoreSums) > 0 {
			summary.AverageScoreByName = map[string]float64{}
			for name, sum := range scoreSums {
				summary.AverageScoreByName[name] = round3(sum / float64(scoreCounts[name]))
			}
		}
		if len(hardFailCounts) > 0 {
			summary.HardFailCounts = hardFailCounts
		}

		summaries = append(summaries, summary)
	}

	return summaries
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
