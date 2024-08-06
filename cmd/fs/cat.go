package fs

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newCatCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cat FILE_PATH",
		Short:   "Show file content.",
		Long:    `Show the contents of a file in DBFS or a UC Volume.`,
		Args:    root.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		f, path, err := filerForPath(ctx, args[0])
		if err != nil {
			return err
		}

		r, err := f.Read(ctx, path)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, r)
	}

	v := newValidArgs()
	cmd.ValidArgsFunction = v.Validate

	return cmd
}
