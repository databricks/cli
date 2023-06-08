package workspace

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

// The callback function exports the file specified at relPath. This function is
// meant to be used in conjunction with fs.WalkDir
func exportFileCallback(ctx context.Context, workspaceFiler filer.Filer, sourceDir, targetDir string) func(string, fs.DirEntry, error) error {
	return func(relPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		sourcePath := path.Join(sourceDir, relPath)
		targetPath := filepath.Join(targetDir, relPath)

		// create directory and return early
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Add extension to local file path if the file is a notebook
		info, err := d.Info()
		if err != nil {
			return err
		}
		objectInfo := info.Sys().(workspace.ObjectInfo)
		if objectInfo.ObjectType == workspace.ObjectTypeNotebook {
			switch objectInfo.Language {
			case workspace.LanguagePython:
				targetPath += ".py"
			case workspace.LanguageR:
				targetPath += ".r"
			case workspace.LanguageScala:
				targetPath += ".scala"
			case workspace.LanguageSql:
				targetPath += ".sql"
			default:
				// Do not add any extension to the file name
			}
		}

		// Skip file if a file already exists in path.
		// os.Stat returns a fs.ErrNotExist if a file does not exist at path.
		// If a file exists, and overwrite is not set, we skip exporting the file
		if _, err := os.Stat(targetPath); err == nil && !exportOverwrite {
			// Log event that this file/directory has been skipped
			return cmdio.RenderWithTemplate(ctx, newFileSkippedEvent(sourcePath, targetPath), "{{.SourcePath}} -> {{.TargetPath}} (skipped; already exists)\n")
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
		return cmdio.RenderWithTemplate(ctx, newFileExportedEvent(sourcePath, targetPath), "{{.SourcePath}} -> {{.TargetPath}}\n")
	}
}

var exportDirCommand = &cobra.Command{
	Use:   "export-dir SOURCE_PATH TARGET_PATH",
	Short: `Export a directory from a Databricks workspace to the local file system.`,
	Long: `
Export a directory recursively from a Databricks workspace to the local file system.
Notebooks will have one of the following extensions added .scala, .py, .sql, or .r
based on the language type.
`,
	PreRunE: root.MustWorkspaceClient,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		sourceDir := args[0]
		targetDir := args[1]

		// Initialize a filer and a file system on the source directory
		workspaceFiler, err := filer.NewWorkspaceFilesClient(w, sourceDir)
		if err != nil {
			return err
		}
		workspaceFS := filer.NewFS(ctx, workspaceFiler)

		// TODO: print progress events on stderr instead: https://github.com/databricks/cli/issues/448
		err = cmdio.RenderWithTemplate(ctx, newExportStartedEvent(sourceDir), "Export started. Download files from  {{.SourcePath}}\n")
		if err != nil {
			return err
		}

		err = fs.WalkDir(workspaceFS, ".", exportFileCallback(ctx, workspaceFiler, sourceDir, targetDir))
		if err != nil {
			return err
		}
		return cmdio.RenderWithTemplate(ctx, newExportCompletedEvent(targetDir), "Export complete. Files can be found at {{.TargetPath}}\n")
	},
}

var exportOverwrite bool

func init() {
	exportDirCommand.Flags().BoolVar(&exportOverwrite, "overwrite", false, "overwrite existing local files")
	Cmd.AddCommand(exportDirCommand)
}
