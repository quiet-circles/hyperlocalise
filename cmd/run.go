package cmd

import (
	"fmt"
	"io"
	"runtime"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/runsvc"
	"github.com/quiet-circles/hyperlocalise/internal/progressui"
	"github.com/spf13/cobra"
)

type runOptions struct {
	configPath string
	dryRun     bool
	prune      bool
	pruneLimit int
	pruneForce bool
	workers    int
	progress   string
}

var runFunc = runsvc.Run

func newRunCmd() *cobra.Command {
	o := runOptions{}

	cmd := &cobra.Command{
		Use:          "run",
		Short:        "generate local translations from source files",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workers := o.workers
			if workers == 0 {
				workers = runtime.NumCPU()
			}
			if workers < 1 {
				return fmt.Errorf("invalid --workers value %d: must be >= 1", workers)
			}

			progressMode, err := progressui.ParseMode(o.progress)
			if err != nil {
				return err
			}

			output := cmd.OutOrStdout()
			var renderer *progressui.Renderer
			if progressui.IsEnabled(progressMode, output, nil) {
				renderer = progressui.New(output, progressMode, progressui.Options{Label: "Translating"})
			}
			if renderer != nil {
				defer renderer.Close()
			}

			input := runsvc.Input{
				ConfigPath: o.configPath,
				DryRun:     o.dryRun,
				Prune:      o.prune,
				PruneLimit: o.pruneLimit,
				PruneForce: o.pruneForce,
				Workers:    workers,
			}
			if renderer != nil {
				input.OnEvent = func(event runsvc.Event) {
					applyRunProgressEvent(renderer, event)
				}
			}

			report, err := runFunc(backgroundContext(), input)
			if renderer != nil {
				renderer.Complete()
			}

			if writeErr := writeRunReport(output, report, o.dryRun); writeErr != nil {
				return fmt.Errorf("write run report: %w", writeErr)
			}

			if err != nil {
				return err
			}
			if report.Failed > 0 {
				return fmt.Errorf("run completed with failures: %d", report.Failed)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.configPath, "config", "", "path to i18n config")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", o.dryRun, "preview planned translation work without executing")
	cmd.Flags().BoolVar(&o.prune, "prune", o.prune, "remove target keys that no longer exist in source files")
	cmd.Flags().IntVar(&o.pruneLimit, "prune-max-deletions", 100, "maximum stale keys that can be deleted in one run before requiring an explicit override")
	cmd.Flags().BoolVar(&o.pruneForce, "prune-force", o.pruneForce, "bypass prune deletion safety limit")
	cmd.Flags().IntVar(&o.workers, "workers", o.workers, "number of parallel translation workers (default: number of CPU cores)")
	cmd.Flags().StringVar(&o.progress, "progress", string(progressui.ModeAuto), "progress rendering mode: auto|on|off")

	return cmd
}

func writeRunReport(w io.Writer, report runsvc.Report, dryRun bool) error {
	if _, err := fmt.Fprintf(
		w,
		"planned_total=%d skipped_by_lock=%d executable_total=%d\n",
		report.PlannedTotal,
		report.SkippedByLock,
		report.ExecutableTotal,
	); err != nil {
		return err
	}

	if len(report.Executable) > 0 {
		if _, err := fmt.Fprintln(w, "tasks:"); err != nil {
			return err
		}
		for _, task := range report.Executable {
			if _, err := fmt.Fprintf(
				w,
				"- target=%s key=%s source=%s target_locale=%s profile=%s\n",
				task.TargetPath,
				task.EntryKey,
				task.SourceLocale,
				task.TargetLocale,
				task.ProfileName,
			); err != nil {
				return err
			}
		}
	}

	if len(report.Skipped) > 0 {
		if _, err := fmt.Fprintln(w, "skipped_by_lock:"); err != nil {
			return err
		}
		for _, task := range report.Skipped {
			if _, err := fmt.Fprintf(w, "- target=%s key=%s\n", task.TargetPath, task.EntryKey); err != nil {
				return err
			}
		}
	}

	if dryRun {
		if len(report.PruneCandidates) > 0 {
			if _, err := fmt.Fprintf(w, "prune_candidates=%d\n", len(report.PruneCandidates)); err != nil {
				return err
			}
			for _, candidate := range report.PruneCandidates {
				if _, err := fmt.Fprintf(w, "prune target=%s key=%s\n", candidate.TargetPath, candidate.EntryKey); err != nil {
					return err
				}
			}
		}
		_, err := fmt.Fprintln(w, "dry_run=true")
		return err
	}

	if _, err := fmt.Fprintf(
		w,
		"succeeded=%d failed=%d persisted_to_lock=%d\n",
		report.Succeeded,
		report.Failed,
		report.PersistedToLock,
	); err != nil {
		return err
	}

	for _, failure := range report.Failures {
		if _, err := fmt.Fprintf(w, "failure target=%s key=%s reason=%s\n", failure.TargetPath, failure.EntryKey, failure.Reason); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "prune_applied=%d\n", report.PruneApplied); err != nil {
		return err
	}

	return nil
}
