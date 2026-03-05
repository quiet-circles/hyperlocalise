package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/runsvc"
	"github.com/quiet-circles/hyperlocalise/internal/progressui"
	"github.com/spf13/cobra"
)

type runOptions struct {
	configPath                string
	dryRun                    bool
	force                     bool
	prune                     bool
	pruneLimit                int
	pruneForce                bool
	workers                   int
	progress                  string
	bucket                    string
	group                     string
	targetLocales             []string
	outputPath                string
	experimentalContextMemory bool
	contextMemoryScope        string
	contextMemoryMaxChars     int
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
			if cmd.Flags().Changed("target-locale") {
				if len(o.targetLocales) == 0 {
					return fmt.Errorf("invalid --target-locale value: must not be empty")
				}
				for _, locale := range o.targetLocales {
					if strings.TrimSpace(locale) == "" {
						return fmt.Errorf("invalid --target-locale value: must not be empty")
					}
				}
			}
			contextMemoryScope := strings.ToLower(strings.TrimSpace(o.contextMemoryScope))
			if contextMemoryScope == "" {
				contextMemoryScope = runsvc.ContextMemoryScopeFile
			}
			switch contextMemoryScope {
			case runsvc.ContextMemoryScopeFile, runsvc.ContextMemoryScopeBucket, runsvc.ContextMemoryScopeGroup:
			default:
				return fmt.Errorf("invalid --context-memory-scope value %q: must be one of %s|%s|%s", o.contextMemoryScope, runsvc.ContextMemoryScopeFile, runsvc.ContextMemoryScopeBucket, runsvc.ContextMemoryScopeGroup)
			}

			progressMode, err := progressui.ParseMode(o.progress)
			if err != nil {
				return err
			}

			output := cmd.OutOrStdout()
			runCtx, stop := signal.NotifyContext(backgroundContext(), os.Interrupt)
			defer stop()

			var renderer *progressui.Renderer
			if progressui.IsEnabled(progressMode, output, nil) {
				renderer = progressui.New(output, progressMode, progressui.Options{
					Label:       "Translating",
					OnInterrupt: stop,
				})
			}
			if renderer != nil {
				defer renderer.Close()
			}

			input := runsvc.Input{
				ConfigPath:                o.configPath,
				DryRun:                    o.dryRun,
				Force:                     o.force,
				Prune:                     o.prune,
				PruneLimit:                o.pruneLimit,
				PruneForce:                o.pruneForce,
				Workers:                   workers,
				Bucket:                    o.bucket,
				Group:                     o.group,
				TargetLocales:             o.targetLocales,
				ExperimentalContextMemory: o.experimentalContextMemory,
				ContextMemoryScope:        contextMemoryScope,
				ContextMemoryMaxChars:     o.contextMemoryMaxChars,
			}
			if renderer != nil {
				input.OnEvent = func(event runsvc.Event) {
					applyRunProgressEvent(renderer, event)
				}
			}

			report, err := runFunc(runCtx, input)
			if renderer != nil {
				renderer.TokenUsage(report.PromptTokens, report.CompletionTokens, report.TotalTokens)
				renderer.Complete()
			}

			if writeErr := writeRunReport(output, report, o.dryRun); writeErr != nil {
				return fmt.Errorf("write run report: %w", writeErr)
			}
			if writeErr := writeRunReportArtifact(o.outputPath, report); writeErr != nil {
				return fmt.Errorf("write run report artifact: %w", writeErr)
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
	cmd.Flags().BoolVar(&o.force, "force", o.force, "rerun all planned tasks and ignore lockfile skip state")
	cmd.Flags().BoolVar(&o.prune, "prune", o.prune, "remove target keys that no longer exist in source files")
	cmd.Flags().IntVar(&o.pruneLimit, "prune-max-deletions", 100, "maximum stale keys that can be deleted in one run before requiring an explicit override")
	cmd.Flags().BoolVar(&o.pruneForce, "prune-force", o.pruneForce, "bypass prune deletion safety limit")
	cmd.Flags().IntVar(&o.workers, "workers", o.workers, "number of parallel translation workers (default: number of CPU cores)")
	cmd.Flags().StringVar(&o.progress, "progress", string(progressui.ModeAuto), "progress rendering mode: auto|on|off")
	cmd.Flags().StringVar(&o.bucket, "bucket", "", "only run tasks for the given bucket")
	cmd.Flags().StringVar(&o.group, "group", "", "only run tasks for the given group")
	cmd.Flags().StringSliceVar(&o.targetLocales, "target-locale", nil, "only run tasks for the given target locale(s)")
	cmd.Flags().StringVar(&o.outputPath, "output", "", "report output JSON path")
	cmd.Flags().BoolVar(&o.experimentalContextMemory, "experimental-context-memory", o.experimentalContextMemory, "enable experimental two-stage context memory generation before translation")
	cmd.Flags().StringVar(&o.contextMemoryScope, "context-memory-scope", runsvc.ContextMemoryScopeFile, "scope for experimental context memory: file|bucket|group")
	cmd.Flags().IntVar(&o.contextMemoryMaxChars, "context-memory-max-chars", 1200, "maximum context memory characters injected into each translation request")

	return cmd
}

func writeRunReportArtifact(path string, report runsvc.Report) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(trimmed), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(trimmed, payload, 0o644)
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
	if _, err := fmt.Fprintf(w, "prompt_tokens=%d completion_tokens=%d total_tokens=%d\n", report.PromptTokens, report.CompletionTokens, report.TotalTokens); err != nil {
		return err
	}
	if report.ContextMemoryEnabled {
		if _, err := fmt.Fprintf(
			w,
			"context_memory_enabled=%t context_memory_scope=%s context_memory_generated=%d context_memory_fallback_groups=%d\n",
			report.ContextMemoryEnabled,
			report.ContextMemoryScope,
			report.ContextMemoryGenerated,
			report.ContextMemoryFallbackGroups,
		); err != nil {
			return err
		}
	}
	if len(report.LocaleUsage) > 0 {
		locales := make([]string, 0, len(report.LocaleUsage))
		for locale := range report.LocaleUsage {
			locales = append(locales, locale)
		}
		sort.Strings(locales)
		for _, locale := range locales {
			usage := report.LocaleUsage[locale]
			if _, err := fmt.Fprintf(w, "locale_usage locale=%s prompt_tokens=%d completion_tokens=%d total_tokens=%d\n", locale, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens); err != nil {
				return err
			}
		}
	}

	for _, failure := range report.Failures {
		if _, err := fmt.Fprintf(w, "failure target=%s key=%s reason=%s\n", failure.TargetPath, failure.EntryKey, failure.Reason); err != nil {
			return err
		}
	}
	for _, warning := range report.Warnings {
		if _, err := fmt.Fprintf(w, "warning=%s\n", warning); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "prune_applied=%d\n", report.PruneApplied); err != nil {
		return err
	}

	return nil
}
