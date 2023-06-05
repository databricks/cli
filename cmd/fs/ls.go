package fs

import (
	"io/fs"
	"sort"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

type jsonDirEntry struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_directory"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"last_modified"`
}

func toJsonDirEntry(f fs.DirEntry) (*jsonDirEntry, error) {
	info, err := f.Info()
	if err != nil {
		return nil, err
	}

	return &jsonDirEntry{
		Name:    f.Name(),
		IsDir:   f.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:     "ls <dir-name>",
	Short:   "Lists files",
	Long:    `Lists files`,
	Args:    cobra.ExactArgs(1),
	PreRunE: root.MustWorkspaceClient,
	Annotations: map[string]string{
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

		lsOutputs := make([]jsonDirEntry, 0)
		for _, entry := range entries {
			parsedEntry, err := toJsonDirEntry(entry)
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
			return cmdio.RenderWithTemplate(ctx, lsOutputs, cmdio.Heredoc(`
			{{range .}}{{if .IsDir}}DIRECTORY {{else}}FILE      {{end}}{{.Size}} {{.ModTime|pretty_date}} {{.Name}}
			{{end}}
			`))
		}
		return cmdio.Render(ctx, lsOutputs)
	},
}

var longMode bool

func init() {
	lsCmd.Flags().BoolVarP(&longMode, "long", "l", false, "Displays full information including size, file type  and modification time since Epoch in milliseconds.")
	fsCmd.AddCommand(lsCmd)
}
