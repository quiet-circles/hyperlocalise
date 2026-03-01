package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/evalsvc"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/evalsvc/scoring"
)

func TestRootHelpIncludesEvalCommand(t *testing.T) {
	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run root help: %v", err)
	}

	if !strings.Contains(out.String(), "eval") {
		t.Fatalf("expected help to include eval command, got %q", out.String())
	}
}

func TestEvalRunRequiresEvalSet(t *testing.T) {
	cmd := newRootCmd("")
	cmd.SetArgs([]string{"eval", "run"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--eval-set is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvalRunRejectsInlineAndFilePromptTogether(t *testing.T) {
	cmd := newRootCmd("")
	cmd.SetArgs([]string{"eval", "run", "--eval-set", "set.json", "--prompt", "x", "--prompt-file", "prompt.txt"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvalRunPrintsSummary(t *testing.T) {
	prev := evalRunFunc
	evalRunFunc = func(_ context.Context, _ evalsvc.Input) (evalsvc.Report, error) {
		return evalsvc.Report{Runs: []evalsvc.RunResult{
			{
				ExperimentID: "default|openai|gpt|prompt",
				LatencyMS:    10,
				Quality: scoring.Result{
					WeightedAggregate: 0.9,
				},
			},
			{
				ExperimentID: "default|openai|gpt|prompt",
				LatencyMS:    20,
				Error:        "fail",
				Quality: scoring.Result{
					WeightedAggregate: 0.6,
					HardFails:         []string{scoring.HardFailPlaceholderDrop},
				},
			},
		}}, nil
	}
	t.Cleanup(func() { evalRunFunc = prev })

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"eval", "run", "--eval-set", "set.json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("eval run: %v", err)
	}
	if !strings.Contains(out.String(), "experiment | score | pass_rate") {
		t.Fatalf("expected table header, got %q", out.String())
	}
	if !strings.Contains(out.String(), "50.0%") {
		t.Fatalf("expected pass rate, got %q", out.String())
	}
}

func TestEvalCompareFailsOnRegressionThreshold(t *testing.T) {
	dir := t.TempDir()
	candidatePath := filepath.Join(dir, "candidate.json")
	baselinePath := filepath.Join(dir, "baseline.json")

	candidate := `{"aggregate":{"weightedScore":0.7},"runs":[{"experimentId":"exp","quality":{"weightedAggregate":0.7},"latencyMs":12}]}`
	baseline := `{"aggregate":{"weightedScore":0.9},"runs":[{"experimentId":"exp","quality":{"weightedAggregate":0.9},"latencyMs":10}]}`

	if err := os.WriteFile(candidatePath, []byte(candidate), 0o600); err != nil {
		t.Fatalf("write candidate: %v", err)
	}
	if err := os.WriteFile(baselinePath, []byte(baseline), 0o600); err != nil {
		t.Fatalf("write baseline: %v", err)
	}

	cmd := newRootCmd("")
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"eval", "compare", "--candidate", candidatePath, "--baseline", baselinePath, "--max-regression", "0.1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected regression error")
	}
	if !strings.Contains(err.Error(), "exceeds max regression") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "candidate experiment summary") {
		t.Fatalf("expected compare output, got %q", out.String())
	}
}
