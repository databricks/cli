package fs

import (
	"sort"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:     "ls <dir-name>",
	Short:   "Lists files",
	Long:    `Lists files`,
	Args:    cobra.ExactArgs(1),
	PreRunE: root.MustWorkspaceClient,
	Annotations: map[string]string{
		"template_long": cmdio.Heredoc(`
		{{range .}}{{if .IsDir}}DIRECTORY {{else}}FILE      {{end}}{{.Size}} {{.ModTime|pretty_date}} {{.Name}}
		{{end}}
		`),
		"template": cmdio.Heredoc(`
		{{range .}}{{.Name}}
		{{end}}
		`),
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		path, err := filer.ResolveDbfsPath(args[0])
		if err != nil {
			return err
		}

		f, err := filer.NewDbfsClient(w, path)
		if err != nil {
			return err
		}

		entries, err := f.ReadDir(ctx, "")
		if err != nil {
			return err
		}

		lsOutputs := make([]lsOutput, 0)
		for _, entry := range entries {
			parsedEntry, err := toLsOutput(entry)
			if err != nil {
				return err
			}
			lsOutputs = append(lsOutputs, *parsedEntry)
			sort.Slice(lsOutputs, func(i, j int) bool {
				return lsOutputs[i].Name < lsOutputs[j].Name
			})
		}

		// Use template for long mode if the flag is set
		if longMode {
			return cmdio.RenderWithTemplate(ctx, lsOutputs, "template_long")
		}
		return cmdio.Render(ctx, lsOutputs)
	},
}

var longMode bool

func init() {
	lsCmd.Flags().BoolVarP(&longMode, "long", "l", false, "Displays full information including size, file type  and modification time since Epoch in milliseconds.")
	fsCmd.AddCommand(lsCmd)
}
