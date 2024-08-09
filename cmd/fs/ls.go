package fs

import (
	"io/fs"
	"path"
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

func toJsonDirEntry(f fs.DirEntry, baseDir string, isAbsolute bool) (*jsonDirEntry, error) {
	info, err := f.Info()
	if err != nil {
		return nil, err
	}

	name := f.Name()
	if isAbsolute {
		name = path.Join(baseDir, name)
	}

	return &jsonDirEntry{
		Name:    name,
		IsDir:   f.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

func newLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ls DIR_PATH",
		Short:   "Lists files.",
		Long:    `Lists files in DBFS and UC Volumes.`,
		Args:    root.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
	}

	var long bool
	var absolute bool
	cmd.Flags().BoolVarP(&long, "long", "l", false, "Displays full information including size, file type and modification time since Epoch in milliseconds.")
	cmd.Flags().BoolVar(&absolute, "absolute", false, "Displays absolute paths.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
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
			jsonDirEntry, err := toJsonDirEntry(entry, args[0], absolute)
			if err != nil {
				return err
			}
			jsonDirEntries[i] = *jsonDirEntry
		}
		sort.Slice(jsonDirEntries, func(i, j int) bool {
			return jsonDirEntries[i].Name < jsonDirEntries[j].Name
		})

		// Use template for long mode if the flag is set
		if long {
			return cmdio.RenderWithTemplate(ctx, jsonDirEntries, "", cmdio.Heredoc(`
			{{range .}}{{if .IsDir}}DIRECTORY {{else}}FILE      {{end}}{{.Size}} {{.ModTime|pretty_date}} {{.Name}}
			{{end}}
			`))
		}
		return cmdio.RenderWithTemplate(ctx, jsonDirEntries, "", cmdio.Heredoc(`
		{{range .}}{{.Name}}
		{{end}}
		`))
	}

	v := newValidArgs()
	v.onlyDirs = true
	cmd.ValidArgsFunction = v.Validate

	return cmd
}
