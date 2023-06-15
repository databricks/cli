package fs

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:     "rm PATH",
	Short:   "Remove files and directories from dbfs.",
	Long:    `Remove files and directories from dbfs.`,
	Args:    cobra.ExactArgs(1),
	PreRunE: root.MustWorkspaceClient,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		f, path, err := filerForPath(ctx, args[0])
		if err != nil {
			return err
		}

		if recursive {
			return f.Delete(ctx, path, filer.DeleteRecursively)
		}
		return f.Delete(ctx, path)
	},
}

var recursive bool

func init() {
	rmCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively delete a non-empty directory.")
	fsCmd.AddCommand(rmCmd)
}
