package fs

import (
	"io/ioutil"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

var catCmd = &cobra.Command{
	Use:     "cat <file-path>",
	Short:   "Show file content",
	Long:    `Show the contents of a file. Does not work for directories.`,
	Args:    cobra.ExactArgs(1),
	PreRunE: root.MustWorkspaceClient,
	Annotations: map[string]string{
		"template": cmdio.Heredoc(`{{.Content}}`),
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		path, err := filer.ResolveDbfsPath(args[0])
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

		b, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		content := string(b)
		return cmdio.Render(ctx, toCatOutput(content))
	},
}

func init() {
	fsCmd.AddCommand(catCmd)
}
