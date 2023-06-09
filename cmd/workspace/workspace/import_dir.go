package workspace

import (
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/notebook"
	"github.com/spf13/cobra"
)

// The callback function imports the file specified at sourcePath. This function is
// meant to be used in conjunction with fs.WalkDir
func importFileCallback(ctx context.Context, workspaceFiler filer.Filer, sourceDir, targetDir string) func(string, fs.DirEntry, error) error {
	return func(sourcePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute file path relative to source directory
		localName, err := filepath.Rel(sourceDir, sourcePath)
		if err != nil {
			return err
		}

		// create directory and return early
		if d.IsDir() {
			return workspaceFiler.Mkdir(ctx, localName)
		}

		// Compute remote name for target
		remoteName := localName
		isNotebook, _, err := notebook.Detect(sourcePath)
		if err != nil {
			return err
		}
		if isNotebook {
			ext := filepath.Ext(localName)
			remoteName = strings.TrimSuffix(localName, ext)
		}

		// Skip file if a file already exists at path in the workspace
		// filer.Stat returns a fs.ErrNotExist if a file does not exist at path.
		// If a file exists, and overwrite is not set, we skip importing the file
		//
		// Why do we need this additional API call?
		// The /workspace-files/import-file API when file already exists returns:
		// 1. 409 Error (Conflict) if object is a file
		// 2. 400 Error (Bad Request) if object is a notebook
		// We make this additional Stat API call to avoid regexing matching the message
		// in the 400 error (in principle avoiding client complexity in order to paint over API gaps)
		if _, err := workspaceFiler.Stat(ctx, remoteName); err == nil && !importOverwrite {
			// Log event that this file/directory has been skipped
			return cmdio.RenderWithTemplate(ctx, newFileSkippedEvent(localName, path.Join(targetDir, remoteName)), "{{.SourcePath}} -> {{.TargetPath}} (skipped; already exists)\n")
		}

		// Open the local file
		f, err := os.Open(sourcePath)
		if err != nil {
			return err
		}

		// Create file in WSFS
		err = workspaceFiler.Write(ctx, localName, f, filer.OverwriteIfExists)
		if err != nil {
			return err
		}

		return cmdio.RenderWithTemplate(ctx, newFileImportedEvent(localName, path.Join(targetDir, remoteName)), "{{.SourcePath}} -> {{.TargetPath}}\n")
	}
}

var importDirCommand = &cobra.Command{
	Use:   "import-dir SOURCE_PATH TARGET_PATH",
	Short: `Import a directory from the local filesystem to a Databricks workspace.`,
	Long: `
Import a directory recursively from the local file system to a Databricks workspace.
Notebooks will have their extensions (one of .scala, .py, .sql, .ipynb, .r) stripped
`,
	PreRunE: root.MustWorkspaceClient,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		sourceDir := args[0]
		targetDir := args[1]

		// Convert sourceDir to an absolute path
		if !filepath.IsAbs(sourceDir) {
			sourceDir, err = filepath.Abs(sourceDir)
			if err != nil {
				return err
			}
		}

		// Initialize a filer rooted at targetDir
		workspaceFiler, err := filer.NewWorkspaceFilesClient(w, targetDir)
		if err != nil {
			return err
		}

		// TODO: print progress events on stderr instead: https://github.com/databricks/cli/issues/448
		err = cmdio.RenderJson(ctx, newImportStartedEvent(sourceDir))
		if err != nil {
			return err
		}

		// Walk local directory tree and import files to the workspace
		err = filepath.WalkDir(sourceDir, importFileCallback(ctx, workspaceFiler, sourceDir, targetDir))
		if err != nil {
			return err
		}
		return cmdio.RenderJson(ctx, newImportCompletedEvent(targetDir))
	},
}

var importOverwrite bool

func init() {
	importDirCommand.Flags().BoolVar(&importOverwrite, "overwrite", false, "overwrite existing workspace files")
	Cmd.AddCommand(importDirCommand)
}
