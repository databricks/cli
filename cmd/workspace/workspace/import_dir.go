package workspace

import (
	"context"
	"errors"
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
//
// We deal with 3 different names for files. The need for this
// arises due to workspace API behaviour and limitations
//
// 1. Local name: The name for the file in the local file system
// 2. Remote name: The name of the file as materialized in the workspace
// 3. API payload name: The name to be used for API calls
//
// Example, consider the notebook "foo\\myNotebook.py" on a windows file system.
// The process to upload it would look like
// 1. Read the notebook, referring to it using it's local name "foo\\myNotebook.py"
// 2. API call to import the notebook to the workspace, using it API payload name "foo/myNotebook.py"
// 3. The notebook is materialized in the workspace using it's remote name "foo/myNotebook"
func importFileCallback(ctx context.Context, workspaceFiler filer.Filer, sourceDir, targetDir string) func(string, fs.DirEntry, error) error {
	return func(sourcePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// localName is the name for the file in the local file system
		localName, err := filepath.Rel(sourceDir, sourcePath)

		// nameForApiCall is the name for the file to be used in any API call.
		// This is a file name we provide to the filer.Write and Mkdir methods
		nameForApiCall := filepath.ToSlash(localName)
		if err != nil {
			return err
		}

		// create directory and return early
		if d.IsDir() {
			return workspaceFiler.Mkdir(ctx, nameForApiCall)
		}

		// remoteName is the name of the file as visible in the workspace. We compute
		// the remote name on the client side for logging purposes
		remoteName := filepath.ToSlash(localName)
		isNotebook, _, err := notebook.Detect(sourcePath)
		if err != nil {
			return err
		}
		if isNotebook {
			ext := path.Ext(localName)
			remoteName = strings.TrimSuffix(localName, ext)
		}

		// Open the local file
		f, err := os.Open(sourcePath)
		if err != nil {
			return err
		}

		// Create file in WSFS
		if importOverwrite {
			err = workspaceFiler.Write(ctx, nameForApiCall, f, filer.OverwriteIfExists)
			if err != nil {
				return err
			}
		} else {
			err = workspaceFiler.Write(ctx, nameForApiCall, f)
			if errors.Is(err, fs.ErrExist) {
				// Emit file skipped event with the appropriate template
				fileSkippedEvent := newFileSkippedEvent(localName, path.Join(targetDir, remoteName))
				template := "{{.SourcePath}} -> {{.TargetPath}} (skipped; already exists)\n"
				return cmdio.RenderWithTemplate(ctx, fileSkippedEvent, template)
			}
			if err != nil {
				return err
			}
		}
		fileImportedEvent := newFileImportedEvent(localName, path.Join(targetDir, remoteName))
		return cmdio.RenderWithTemplate(ctx, fileImportedEvent, "{{.SourcePath}} -> {{.TargetPath}}\n")
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
