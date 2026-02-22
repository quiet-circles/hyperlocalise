package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// ManCmd manpage command.
type ManCmd struct {
	Cmd *cobra.Command
}

// NewManCmd manpage cmd.
// nolint
func NewManCmd() *ManCmd {
	root := &ManCmd{}

	c := &cobra.Command{
		Use:                   "man",
		Short:                 "Generates command line manpages",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Hidden:                true,
		Args:                  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			header := &doc.GenManHeader{
				Title:   "HYPERLOCALISE",
				Section: "1",
			}

			return doc.GenMan(root.Cmd.Root(), header, os.Stdout)
		},
	}

	root.Cmd = c

	return root
}
