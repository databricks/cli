package fs

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newMkdirCommand(filerForPath PathResolver) *cobra.Command {
	cmd := &cobra.Command{
		Use: "mkdir DIR_PATH",
		// Alias `mkdirs` for this command exists for legacy purposes. This command
		// is called databricks fs mkdirs in our legacy CLI: https://github.com/databricks/databricks-cli
		Aliases: []string{"mkdirs"},
		Short:   "Make directories.",
		Long:    `Make directories in DBFS, UC Volumes and UC Skills. Mkdir will create directories along the path to the argument directory. Paths in DBFS, UC Volumes and UC Skills need the "dbfs" scheme, for example dbfs:/Volumes/my_catalog/my_schema/my_volume. A path without a scheme is a local path, even if it starts with /Volumes/ or /Skills/.`,
		Args:    root.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		f, path, err := filerForPath(ctx, args[0])
		if err != nil {
			return err
		}

		return f.Mkdir(ctx, path)
	}

	v := newValidArgs()
	v.filerForPathFunc = filerForPath
	v.onlyDirs = true
	cmd.ValidArgsFunction = v.Validate

	return cmd
}
