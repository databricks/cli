package fs

import (
	"path"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

type FileInfo filer.FileInfo

func (i *FileInfo) ExpandPath(root string) {
	i.Name = path.Join(root, i.Name)
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
			{{range .}}{{.Type|printf "%-10s"}} {{.Size}}  {{.ModTime|unix_date}}  {{.Name}}
			{{end}}
			`)
		}

		// Path to list files from. Defaults to`/`
		rootPath := "/"
		if len(args) > 0 {
			rootPath = args[0]
		}

		// Initialize workspace client
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		f, err := filer.NewWorkspaceFilesClient(w, rootPath)
		if err != nil {
			return err
		}

		// Get file info
		filesInfo, err := f.ReadDir(ctx, "")
		if err != nil {
			return err
		}

		// compute output with expanded paths if necessary
		output := make([]FileInfo, len(filesInfo))
		for i, v := range filesInfo {
			output[i] = FileInfo(v)
			if absolute {
				output[i].ExpandPath(rootPath)
			}
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
