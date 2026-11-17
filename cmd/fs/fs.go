package fs

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// Spec declares the fs command.
var Spec = root.CommandSpec{
	Use:     "fs",
	Short:   "Filesystem related commands",
	GroupID: "workspace",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Commands to do file system operations on DBFS, UC Volumes and UC Skills.`

	cmd.AddCommand(
		newCatCommand(),
		newCpCommand(),
		newLsCommand(),
		newMkdirCommand(),
		newRmCommand(),
	)
}
