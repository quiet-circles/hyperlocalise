package cmd

import (
	"fmt"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/syncsvc"
	"github.com/spf13/cobra"
)

func newSyncPullCmd() *cobra.Command {
	o := defaultSyncCommonOptions()

	cmd := &cobra.Command{
		Use:          "pull",
		Short:        "pull latest curated translations from remote storage",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := newSyncRuntime(o.configPath)
			if err != nil {
				return fmt.Errorf("initialize sync runtime: %w", err)
			}

			report, err := rt.svc.Pull(backgroundContext(), syncsvc.PullInput{
				Adapter: rt.remote,
				Local:   rt.local,
				Request: storage.PullRequest{
					Locales: o.locales,
				},
				Read: syncsvc.LocalReadRequest{
					Locales: o.locales,
				},
				Options: syncsvc.PullOptions{
					DryRun:                o.dryRun,
					FailOnConflict:        o.failOnConflict,
					ApplyCuratedOverDraft: o.applyCuratedOverDraft,
					Policy:                syncsvc.PolicyConservativeCurationPull,
				},
			})
			if writeErr := writeSyncReport(cmd, report, o.output); writeErr != nil {
				return fmt.Errorf("write sync pull report: %w", writeErr)
			}
			if err != nil {
				return err
			}

			return nil
		},
	}

	addSyncCommonFlags(cmd, &o)
	return cmd
}
