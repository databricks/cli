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
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/notebook"
	"github.com/spf13/cobra"
)

type importDirOptions struct {
	sourceDir string
	targetDir string
	overwrite bool
}

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
func (opts importDirOptions) callback(ctx context.Context, workspaceFiler filer.Filer) func(string, fs.DirEntry, error) error {
	sourceDir := opts.sourceDir
	targetDir := opts.targetDir
	overwrite := opts.overwrite

	return func(sourcePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// localName is the name for the file in the local file system
		localName, err := filepath.Rel(sourceDir, sourcePath)
		if err != nil {
			return err
		}

		// nameForApiCall is the name for the file to be used in any API call.
		// This is a file name we provide to the filer.Write and Mkdir methods
		nameForApiCall := filepath.ToSlash(localName)

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
		defer f.Close()

		// Create file in WSFS
		if overwrite {
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
				return cmdio.RenderWithTemplate(ctx, fileSkippedEvent, "", template)
			}
			if err != nil {
				return err
			}
		}
		fileImportedEvent := newFileImportedEvent(localName, path.Join(targetDir, remoteName))
		return cmdio.RenderWithTemplate(ctx, fileImportedEvent, "", "{{.SourcePath}} -> {{.TargetPath}}\n")
	}
}

func newImportDir() *cobra.Command {
	cmd := &cobra.Command{}

	var opts importDirOptions

	cmd.Flags().BoolVar(&opts.overwrite, "overwrite", false, "overwrite existing workspace files")

	cmd.Use = "import-dir SOURCE_PATH TARGET_PATH"
	cmd.Short = `Import a directory from the local filesystem to a Databricks workspace.`
	cmd.Long = `
Import a directory recursively from the local file system to a Databricks workspace.
Notebooks will have their extensions (one of .scala, .py, .sql, .ipynb, .r) stripped
`

	cmd.Annotations = make(map[string]string)
	cmd.Args = root.ExactArgs(2)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		opts.sourceDir = args[0]
		opts.targetDir = args[1]

		// Initialize a filer rooted at targetDir
		workspaceFiler, err := filer.NewWorkspaceFilesClient(w, opts.targetDir)
		if err != nil {
			return err
		}

		err = cmdio.RenderWithTemplate(ctx, newImportStartedEvent(opts.sourceDir), "", "Importing files from {{.SourcePath}}\n")
		if err != nil {
			return err
		}

		// Walk local directory tree and import files to the workspace
		err = filepath.WalkDir(opts.sourceDir, opts.callback(ctx, workspaceFiler))
		if err != nil {
			return err
		}
		return cmdio.RenderWithTemplate(ctx, newImportCompletedEvent(opts.targetDir), "", "Import complete\n")
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newImportDir())
	})
}
