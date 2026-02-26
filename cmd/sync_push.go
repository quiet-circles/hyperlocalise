package cmd

import (
	"fmt"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/syncsvc"
	"github.com/spf13/cobra"
)

func newSyncPushCmd() *cobra.Command {
	o := defaultSyncCommonOptions()

	cmd := &cobra.Command{
		Use:          "push",
		Short:        "push local translation changes to remote storage",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := newSyncRuntime(o.configPath)
			if err != nil {
				return fmt.Errorf("initialize sync runtime: %w", err)
			}

			report, err := rt.svc.Push(backgroundContext(), syncsvc.PushInput{
				Adapter: rt.remote,
				Local:   rt.local,
				Read: syncsvc.LocalReadRequest{
					Locales: o.locales,
				},
				Options: syncsvc.PushOptions{
					DryRun:         o.dryRun,
					FailOnConflict: o.failOnConflict,
				},
			})
			if writeErr := writeSyncReport(cmd, report, o.output); writeErr != nil {
				return fmt.Errorf("write sync push report: %w", writeErr)
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
