package fs

import (
	"fmt"
	"path"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

func parseFileInfo(info filer.FileInfo, parentDir string, isAbsolute bool) map[string]string {
	fullName := info.Name
	if isAbsolute {
		fullName = path.Join(parentDir, info.Name)
	}
	return map[string]string{
		"Name":    fullName,
		"ModTime": info.ModTime.UTC().Format(time.UnixDate),
		"Size":    fmt.Sprint(info.Size),
		"Type":    info.Type,
	}
}

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:         "ls <dir-name>",
	Short:       "Lists files",
	Long:        `Lists files in a DBFS or WSFS directory`,
	Args:        cobra.MaximumNArgs(1),
	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,

	RunE: func(cmd *cobra.Command, args []string) error {
		// Assign template according to whether -l is specified
		template := cmdio.Heredoc(`
		{{range .}}{{.Name}}
		{{end}}
		`)
		if longMode {
			template = cmdio.Heredoc(`
			{{range .}}{{.Type|printf "%-10s"}} {{.Size}}  {{.ModTime}}  {{.Name}}
			{{end}}
			`)
		}

		// Path to list files from. Defaults to`/`
		path := "/"
		if len(args) > 0 {
			path = args[0]
		}

		// Initialize workspace client
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		f, err := filer.NewWorkspaceFilesClient(w, path)
		if err != nil {
			return err
		}

		// Get file info
		filesInfo, err := f.ReadDir(ctx, "")
		if err != nil {
			return err
		}

		// Parse it so it's ready to be rendered
		output := make([]map[string]string, 0)
		for _, info := range filesInfo {
			output = append(output, parseFileInfo(info, path, absolute))
		}
		return cmdio.RenderWithTemplate(ctx, output, template)
	},
}

var longMode bool
var absolute bool

func init() {
	lsCmd.Flags().BoolVarP(&longMode, "long", "l", false, "Displays full information including size, file type  and modification time since Epoch in milliseconds.")
	lsCmd.Flags().BoolVar(&absolute, "absolute", false, "Displays absolute paths.")
	fsCmd.AddCommand(lsCmd)
}
