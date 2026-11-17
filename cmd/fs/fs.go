package fs

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

// PathResolver selects a filesystem and removes its prefix from a path.
type PathResolver func(context.Context, string) (filer.Filer, string, error)

// Spec declares the fs command.
var Spec = root.CommandSpec{
	Use:     "fs",
	Short:   "Filesystem related commands",
	GroupID: "workspace",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Commands to do file system operations on DBFS, UC Volumes and UC Skills.`
	AddCommands(cmd, filerForPath)
}

// AddCommands adds the filesystem subcommands to cmd, resolving their paths with resolvePath.
func AddCommands(cmd *cobra.Command, resolvePath PathResolver) {
	cmd.AddCommand(
		newCatCommand(resolvePath),
		newCpCommand(resolvePath),
		newLsCommand(resolvePath),
		newMkdirCommand(resolvePath),
		newRmCommand(resolvePath),
	)
}
