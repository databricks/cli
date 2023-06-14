package fs

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

var catCmd = &cobra.Command{
	Use:     "cat FILE_PATH",
	Short:   "Show file content",
	Long:    `Show the contents of a file.`,
	Args:    cobra.ExactArgs(1),
	PreRunE: root.MustWorkspaceClient,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		path, err := trimDbfsScheme(args[0])
		if err != nil {
			return err
		}

		f, err := filer.NewDbfsClient(w, "/")
		if err != nil {
			return err
		}

		r, err := f.Read(ctx, path)
		if err != nil {
			return err
		}
		return cmdio.RenderReader(ctx, r)
	},
}

func init() {
	fsCmd.AddCommand(catCmd)
}
