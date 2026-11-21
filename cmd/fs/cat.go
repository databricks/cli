package fs

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newCatCommand(filerForPath PathResolver) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cat FILE_PATH",
		Short:   "Show file content.",
		Long:    `Show the contents of a file in DBFS, a UC Volume or a UC Skill. Paths in DBFS, UC Volumes and UC Skills need the "dbfs" scheme, for example dbfs:/Volumes/my_catalog/my_schema/my_volume. A path without a scheme is a local path, even if it starts with /Volumes/ or /Skills/.`,
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
		defer r.Close()
		return cmdio.Render(ctx, r)
	}

	v := newValidArgs()
	v.filerForPathFunc = filerForPath
	cmd.ValidArgsFunction = v.Validate

	return cmd
}
