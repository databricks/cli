package fs

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fs",
		Short:   "Filesystem related commands",
		Long:    `Commands to do file system operations on DBFS, UC Volumes and UC Skills.`,
		GroupID: "workspace",
	}

	cmd.AddCommand(
		newCatCommand(),
		newCpCommand(),
		newLsCommand(),
		newMkdirCommand(),
		newRmCommand(),
	)

	return cmd
}
