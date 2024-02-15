package fs

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

func newRmCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm PATH",
		Short:   "Remove files and directories.",
		Long:    `Remove files and directories from DBFS and UC Volumes.`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
	}

	var recursive bool
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively delete a non-empty directory. This is not supported for paths in a UC Volume.")

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

	return cmd
}
