package cmd

import (
	"fmt"
	"io"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/runsvc"
	"github.com/spf13/cobra"
)

type runOptions struct {
	configPath string
	dryRun     bool
}

func newRunCmd() *cobra.Command {
	o := runOptions{}

	cmd := &cobra.Command{
		Use:          "run",
		Short:        "generate local translations from source files",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report, err := runsvc.Run(backgroundContext(), runsvc.Input{
				ConfigPath: o.configPath,
				DryRun:     o.dryRun,
			})

			if writeErr := writeRunReport(cmd.OutOrStdout(), report, o.dryRun); writeErr != nil {
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

	return nil
}
