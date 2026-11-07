package fs

import (
	"context"

	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

// PathResolver selects a filesystem and removes its prefix from a path.
type PathResolver func(context.Context, string) (filer.Filer, string, error)

func New() *cobra.Command {
	return NewWithPathResolver(filerForPath)
}

// NewWithPathResolver returns filesystem commands using the supplied path resolver.
func NewWithPathResolver(resolvePath PathResolver) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fs",
		Short:   "Filesystem related commands",
		Long:    `Commands to do file system operations on DBFS, UC Volumes and UC Skills.`,
		GroupID: "workspace",
	}

	cmd.AddCommand(
		newCatCommand(resolvePath),
		newCpCommand(resolvePath),
		newLsCommand(resolvePath),
		newMkdirCommand(resolvePath),
		newRmCommand(resolvePath),
	)

	return cmd
}
