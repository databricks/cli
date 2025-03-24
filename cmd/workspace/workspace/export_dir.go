package workspace

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

type exportDirOptions struct {
	sourceDir string
	targetDir string
	overwrite bool
}

// The callback function exports the file specified at relPath. This function is
// meant to be used in conjunction with fs.WalkDir
func (opts exportDirOptions) callback(ctx context.Context, workspaceFiler filer.Filer) func(string, fs.DirEntry, error) error {
	sourceDir := opts.sourceDir
	targetDir := opts.targetDir
	overwrite := opts.overwrite

	return func(relPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		sourcePath := path.Join(sourceDir, relPath)
		targetPath := filepath.Join(targetDir, relPath)

		// create directory and return early
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		// Add extension to local file path if the file is a notebook
		info, err := d.Info()
		if err != nil {
			return err
		}
		objectInfo := info.Sys().(workspace.ObjectInfo)
		targetPath += notebook.GetExtensionByLanguage(&objectInfo)

		// Skip file if a file already exists in path.
		// os.Stat returns a fs.ErrNotExist if a file does not exist at path.
		// If a file exists, and overwrite is not set, we skip exporting the file
		if _, err := os.Stat(targetPath); err == nil && !overwrite {
			// Log event that this file/directory has been skipped
			return cmdio.RenderWithTemplate(ctx, newFileSkippedEvent(relPath, targetPath), "", "{{.SourcePath}} -> {{.TargetPath}} (skipped; already exists)\n")
		}

		// create the file
		f, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer f.Close()

		// Write content to the local file
		r, err := workspaceFiler.Read(ctx, relPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, r)
		if err != nil {
			return err
		}
		return cmdio.RenderWithTemplate(ctx, newFileExportedEvent(sourcePath, targetPath), "", "{{.SourcePath}} -> {{.TargetPath}}\n")
	}
}

func newExportDir() *cobra.Command {
	cmd := &cobra.Command{}

	var opts exportDirOptions

	cmd.Flags().BoolVar(&opts.overwrite, "overwrite", false, "overwrite existing local files")

	cmd.Use = "export-dir SOURCE_PATH TARGET_PATH"
	cmd.Short = `Export a directory from a Databricks workspace to the local file system.`
	cmd.Long = `
	Export a directory recursively from a Databricks workspace to the local file system.
	Notebooks will have one of the following extensions added .scala, .py, .sql, or .r
	based on the language type.
	`

	cmd.Annotations = make(map[string]string)
	cmd.Args = root.ExactArgs(2)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		opts.sourceDir = args[0]
		opts.targetDir = args[1]

		// Initialize a filer and a file system on the source directory
		workspaceFiler, err := filer.NewWorkspaceFilesClient(w, opts.sourceDir)
		if err != nil {
			return err
		}
		workspaceFS := filer.NewFS(ctx, workspaceFiler)

		err = cmdio.RenderWithTemplate(ctx, newExportStartedEvent(opts.sourceDir), "", "Exporting files from {{.SourcePath}}\n")
		if err != nil {
			return err
		}

		err = fs.WalkDir(workspaceFS, ".", opts.callback(ctx, workspaceFiler))
		if err != nil {
			return err
		}
		return cmdio.RenderWithTemplate(ctx, newExportCompletedEvent(opts.targetDir), "", "Export complete\n")
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newExportDir())
	})
}
