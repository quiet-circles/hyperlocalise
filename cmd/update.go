package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

const installScriptURL = "https://raw.githubusercontent.com/quiet-circles/hyperlocalise/main/install.sh"

var selfUpdateRunner = runSelfUpdate

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "update [version]",
		Short:        "Update hyperlocalise using the bootstrap installer",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := ""
			if len(args) == 1 {
				version = args[0]
			}

			if err := selfUpdateRunner(cmd.Context(), version, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
				return fmt.Errorf("self update: %w", err)
			}

			return nil
		},
	}
}

func runSelfUpdate(ctx context.Context, version string, stdout io.Writer, stderr io.Writer) error {
	selfUpdateCommand := fmt.Sprintf("curl -fsSL %s | bash", installScriptURL)

	command := exec.CommandContext(ctx, "bash", "-c", selfUpdateCommand)
	command.Stdout = stdout
	command.Stderr = stderr

	if version != "" {
		command.Env = append(os.Environ(), "VERSION="+version)
	}

	if err := command.Run(); err != nil {
		return fmt.Errorf("run installer command: %w", err)
	}

	return nil
}
