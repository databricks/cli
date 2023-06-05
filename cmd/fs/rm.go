package fs

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/databricks-sdk-go/service/files"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:     "rm DIR_PATH",
	Short:   "Remove files and directories from dbfs.",
	Long:    `Remove files and directories from dbfs.`,
	Args:    cobra.ExactArgs(1),
	PreRunE: root.MustWorkspaceClient,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		path, err := resolveDbfsPath(args[0])
		if err != nil {
			return err
		}

		return w.Dbfs.Delete(ctx, files.Delete{
			Path:      path,
			Recursive: recursive,
		})
	},
}

var recursive bool

func init() {
	rmCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Allow deletion of non-empty directories.")
	fsCmd.AddCommand(rmCmd)
}
