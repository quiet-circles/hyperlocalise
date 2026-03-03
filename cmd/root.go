package cmd

import (
	"fmt"

	"github.com/quiet-circles/hyperlocalise/internal/envloader"
	"github.com/spf13/cobra"
)

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hyperlocalise",
		Short: "High-performance localization CLI written in Go",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := envloader.LoadProjectFiles(); err != nil {
				return fmt.Errorf("load env files: %w", err)
			}

			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			notifyIfUpdateAvailable(cmd, version)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newVersionCmd(version)) // version subcommand
	cmd.AddCommand(newInitCmd())           // init subcommand
	cmd.AddCommand(newSyncCmd())           // sync subcommands
	cmd.AddCommand(newRunCmd())            // run subcommand
	cmd.AddCommand(newEvalCmd())           // eval subcommands
	cmd.AddCommand(newStatusCmd())         // status subcommand
	cmd.AddCommand(NewManCmd().Cmd)        // hidden manpage subcommand

	return cmd
}

// Execute invokes the command.
func Execute(version string) error {
	if err := newRootCmd(version).Execute(); err != nil {
		return fmt.Errorf("error executing root command: %w", err)
	}

	return nil
}
