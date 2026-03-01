package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/evalsvc"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/evalsvc/scoring"
	"github.com/spf13/cobra"
)

var evalRunFunc = evalsvc.Run

type evalRunOptions struct {
	evalSetPath string
	profiles    []string
	providers   []string
	models      []string
	promptFile  string
	prompt      string
	outputPath  string
}

type evalCompareOptions struct {
	candidatePath string
	baselinePath  string
	minScore      float64
	maxRegression float64
}

type experimentSummary struct {
	ID                    string
	AverageScore          float64
	PassRate              float64
	PlaceholderViolations int
	AverageLatencyMS      float64
}

func newEvalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "evaluate translation quality across experiment variants",
	}

	cmd.AddCommand(newEvalRunCmd())
	cmd.AddCommand(newEvalCompareCmd())

	return cmd
}

func newEvalRunCmd() *cobra.Command {
	o := evalRunOptions{}
	cmd := &cobra.Command{
		Use:          "run",
		Short:        "execute experiments and write JSON report",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			prompts, err := resolvePrompts(o.prompt, o.promptFile)
			if err != nil {
				return err
			}
			if strings.TrimSpace(o.evalSetPath) == "" {
				return fmt.Errorf("--eval-set is required")
			}

			report, err := evalRunFunc(backgroundContext(), evalsvc.Input{
				EvalSetPath: o.evalSetPath,
				Profiles:    o.profiles,
				Providers:   o.providers,
				Models:      o.models,
				Prompts:     prompts,
				OutputPath:  o.outputPath,
			})
			if err != nil {
				return fmt.Errorf("run eval: %w", err)
			}

			return writeExperimentSummary(cmd.OutOrStdout(), summarizeExperiments(report.Runs), false)
		},
	}

	cmd.Flags().StringVar(&o.evalSetPath, "eval-set", "", "path to eval dataset (json, jsonc, csv)")
	cmd.Flags().StringArrayVar(&o.profiles, "profile", nil, "profile name to evaluate (repeatable)")
	cmd.Flags().StringArrayVar(&o.providers, "provider", nil, "provider override (repeatable)")
	cmd.Flags().StringArrayVar(&o.models, "model", nil, "model override (repeatable)")
	cmd.Flags().StringVar(&o.promptFile, "prompt-file", "", "path to prompt file override")
	cmd.Flags().StringVar(&o.prompt, "prompt", "", "inline prompt override")
	cmd.Flags().StringVar(&o.outputPath, "output", "", "report output JSON path")

	return cmd
}

func newEvalCompareCmd() *cobra.Command {
	o := evalCompareOptions{}
	cmd := &cobra.Command{
		Use:          "compare",
		Short:        "compare candidate report against baseline report",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(o.candidatePath) == "" {
				return fmt.Errorf("--candidate is required")
			}
			if strings.TrimSpace(o.baselinePath) == "" {
				return fmt.Errorf("--baseline is required")
			}

			candidate, err := loadEvalReport(o.candidatePath)
			if err != nil {
				return err
			}
			baseline, err := loadEvalReport(o.baselinePath)
			if err != nil {
				return err
			}

			candidateScore := candidate.Aggregate.WeightedScore
			baselineScore := baseline.Aggregate.WeightedScore
			regression := baselineScore - candidateScore

			if err := writeExperimentSummary(cmd.OutOrStdout(), summarizeExperiments(candidate.Runs), true); err != nil {
				return err
			}

			if _, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"candidate_score=%.3f baseline_score=%.3f regression=%.3f min_score=%.3f max_regression=%.3f\n",
				candidateScore,
				baselineScore,
				regression,
				o.minScore,
				o.maxRegression,
			); err != nil {
				return err
			}

			if o.minScore > 0 && candidateScore < o.minScore {
				return fmt.Errorf("candidate weighted score %.3f below min score %.3f", candidateScore, o.minScore)
			}
			if o.maxRegression > 0 && regression > o.maxRegression {
				return fmt.Errorf("score regression %.3f exceeds max regression %.3f", regression, o.maxRegression)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.candidatePath, "candidate", "", "candidate eval report JSON path")
	cmd.Flags().StringVar(&o.baselinePath, "baseline", "", "baseline eval report JSON path")
	cmd.Flags().Float64Var(&o.minScore, "min-score", 0, "minimum candidate weighted score")
	cmd.Flags().Float64Var(&o.maxRegression, "max-regression", 0, "maximum allowed score regression vs baseline")

	return cmd
}

func resolvePrompts(prompt string, promptFile string) ([]string, error) {
	inline := strings.TrimSpace(prompt)
	file := strings.TrimSpace(promptFile)
	if inline != "" && file != "" {
		return nil, fmt.Errorf("--prompt and --prompt-file are mutually exclusive")
	}
	if inline != "" {
		return []string{inline}, nil
	}
	if file == "" {
		return nil, nil
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read prompt file: %w", err)
	}
	value := strings.TrimSpace(string(content))
	if value == "" {
		return nil, fmt.Errorf("prompt file is empty")
	}

	return []string{value}, nil
}

func loadEvalReport(path string) (evalsvc.Report, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return evalsvc.Report{}, fmt.Errorf("read report %q: %w", path, err)
	}

	var report evalsvc.Report
	if err := json.Unmarshal(content, &report); err != nil {
		return evalsvc.Report{}, fmt.Errorf("decode report %q: %w", path, err)
	}

	return report, nil
}

func summarizeExperiments(runs []evalsvc.RunResult) []experimentSummary {
	type accumulator struct {
		totalRuns             int
		successRuns           int
		totalScore            float64
		totalLatency          float64
		placeholderViolations int
	}

	byExperiment := map[string]*accumulator{}
	for _, run := range runs {
		acc := byExperiment[run.ExperimentID]
		if acc == nil {
			acc = &accumulator{}
			byExperiment[run.ExperimentID] = acc
		}
		acc.totalRuns++
		if run.Error == "" {
			acc.successRuns++
		}
		acc.totalScore += run.Quality.WeightedAggregate
		acc.totalLatency += run.LatencyMS
		for _, hardFail := range run.Quality.HardFails {
			if hardFail == scoring.HardFailPlaceholderDrop {
				acc.placeholderViolations++
			}
		}
	}

	summaries := make([]experimentSummary, 0, len(byExperiment))
	for id, acc := range byExperiment {
		total := float64(acc.totalRuns)
		summaries = append(summaries, experimentSummary{
			ID:                    id,
			AverageScore:          acc.totalScore / total,
			PassRate:              float64(acc.successRuns) / total,
			PlaceholderViolations: acc.placeholderViolations,
			AverageLatencyMS:      acc.totalLatency / total,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ID < summaries[j].ID
	})

	return summaries
}

func writeExperimentSummary(w io.Writer, summaries []experimentSummary, includeHeader bool) error {
	if includeHeader {
		if _, err := fmt.Fprintln(w, "candidate experiment summary:"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "experiment | score | pass_rate | placeholder_violations | latency_ms"); err != nil {
		return err
	}
	for _, summary := range summaries {
		if _, err := fmt.Fprintf(
			w,
			"%s | %.3f | %.1f%% | %d | %.1f\n",
			summary.ID,
			summary.AverageScore,
			summary.PassRate*100,
			summary.PlaceholderViolations,
			summary.AverageLatencyMS,
		); err != nil {
			return err
		}
	}

	return nil
}
