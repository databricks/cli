package fs

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

func newRmCommand(filerForPath PathResolver) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm PATH",
		Short:   "Remove files and directories.",
		Long:    `Remove files and directories from DBFS, UC Volumes and UC Skills. Paths in DBFS, UC Volumes and UC Skills need the "dbfs" scheme, for example dbfs:/Volumes/my_catalog/my_schema/my_volume. A path without a scheme is a local path, even if it starts with /Volumes/ or /Skills/.`,
		Args:    root.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
	}

	var recursive bool
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively delete a non-empty directory.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		f, path, err := filerForPath(ctx, args[0])
		if err != nil {
			return err
		}

		if recursive {
			return f.Delete(ctx, path, filer.DeleteRecursively)
		}
		return f.Delete(ctx, path)
	}

	v := newValidArgs()
	v.filerForPathFunc = filerForPath
	cmd.ValidArgsFunction = v.Validate

	return cmd
}
