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
	cmd.ValidArgsFunction = v.Validate

	return cmd
}
