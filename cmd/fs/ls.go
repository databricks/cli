package fs

import (
	"io/fs"
	"sort"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
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
	Use:     "ls DIR_PATH",
	Short:   "Lists files",
	Long:    `Lists files`,
	Args:    cobra.ExactArgs(1),
	PreRunE: root.MustWorkspaceClient,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		f, path, err := filerForPath(ctx, args[0])
		if err != nil {
			return err
		}

		entries, err := f.ReadDir(ctx, path)
		if err != nil {
			return err
		}

		jsonDirEntries := make([]jsonDirEntry, len(entries))
		for i, entry := range entries {
			jsonDirEntry, err := toJsonDirEntry(entry)
			if err != nil {
				return err
			}
			jsonDirEntries[i] = *jsonDirEntry
		}
		sort.Slice(jsonDirEntries, func(i, j int) bool {
			return jsonDirEntries[i].Name < jsonDirEntries[j].Name
		})

		// Use template for long mode if the flag is set
		if longMode {
			return cmdio.RenderWithTemplate(ctx, jsonDirEntries, cmdio.Heredoc(`
			{{range .}}{{if .IsDir}}DIRECTORY {{else}}FILE      {{end}}{{.Size}} {{.ModTime|pretty_date}} {{.Name}}
			{{end}}
			`))
		}
		return cmdio.RenderWithTemplate(ctx, jsonDirEntries, cmdio.Heredoc(`
		{{range .}}{{.Name}}
		{{end}}
		`))
	},
}

var longMode bool

func init() {
	lsCmd.Flags().BoolVarP(&longMode, "long", "l", false, "Displays full information including size, file type and modification time since Epoch in milliseconds.")
	fsCmd.AddCommand(lsCmd)
}
