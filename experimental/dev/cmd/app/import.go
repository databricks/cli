package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newImportCmd() *cobra.Command {
	var (
		appName   string
		force     bool
		outputDir string
	)

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import app source code from Databricks workspace to local disk",
		Long: `Import app source code from Databricks workspace to local disk.

Downloads the source code of a deployed Databricks app to a local directory
named after the app.

Examples:
  # Interactive mode - select app from picker
  databricks experimental dev app import

  # Import a specific app's source code
  databricks experimental dev app import --name my-app

  # Import to a specific directory
  databricks experimental dev app import --name my-app --output-dir ./projects

  # Force overwrite existing files
  databricks experimental dev app import --name my-app --force`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Prompt for app name if not provided
			if appName == "" {
				selected, err := PromptForAppSelection(ctx, "Select an app to import")
				if err != nil {
					return err
				}
				appName = selected
			}

			return runImport(ctx, importOptions{
				appName:   appName,
				force:     force,
				outputDir: outputDir,
			})
		},
	}

	cmd.Flags().StringVar(&appName, "name", "", "Name of the app to import (prompts if not provided)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the imported app to")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")

	return cmd
}

type importOptions struct {
	appName   string
	force     bool
	outputDir string
}

func runImport(ctx context.Context, opts importOptions) error {
	w := cmdctx.WorkspaceClient(ctx)

	// Step 1: Fetch the app
	var app *apps.App
	err := RunWithSpinnerCtx(ctx, fmt.Sprintf("Fetching app '%s'...", opts.appName), func() error {
		var fetchErr error
		app, fetchErr = w.Apps.Get(ctx, apps.GetAppRequest{Name: opts.appName})
		return fetchErr
	})
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Step 2: Check if the app has a source code path
	if app.DefaultSourceCodePath == "" {
		return errors.New("app has no source code path - it may not have been deployed yet")
	}

	cmdio.LogString(ctx, fmt.Sprintf("Source code path: %s", app.DefaultSourceCodePath))

	// Step 3: Create output directory
	destDir := opts.appName
	if opts.outputDir != "" {
		destDir = filepath.Join(opts.outputDir, opts.appName)
	}
	if err := ensureOutputDir(destDir, opts.force); err != nil {
		return err
	}

	// Step 4: Download files with spinner
	var fileCount int
	err = RunWithSpinnerCtx(ctx, "Downloading files...", func() error {
		var downloadErr error
		fileCount, downloadErr = downloadDirectory(ctx, w, app.DefaultSourceCodePath, destDir, opts.force)
		return downloadErr
	})
	if err != nil {
		return fmt.Errorf("failed to download files for app '%s': %w", opts.appName, err)
	}

	// Get absolute path for display
	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		absDestDir = destDir
	}

	// Step 5: Run npm install if package.json exists
	packageJSONPath := filepath.Join(destDir, "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		if err := runNpmInstallInDir(ctx, destDir); err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("⚠ npm install failed: %v", err))
			cmdio.LogString(ctx, "  You can run 'npm install' manually in the project directory.")
		}
	}

	// Step 6: Detect and configure DABs
	bundlePath := filepath.Join(destDir, "databricks.yml")
	if _, err := os.Stat(bundlePath); err == nil {
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Detected Databricks Asset Bundle configuration.")
		cmdio.LogString(ctx, "Run 'databricks bundle validate' to verify the bundle is configured correctly.")
	}

	// Show success message with next steps
	PrintSuccess(opts.appName, absDestDir, fileCount, true)

	return nil
}

// runNpmInstallInDir runs npm install in the specified directory.
func runNpmInstallInDir(ctx context.Context, dir string) error {
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found: please install Node.js")
	}

	return RunWithSpinnerCtx(ctx, "Installing dependencies...", func() error {
		cmd := exec.CommandContext(ctx, "npm", "install")
		cmd.Dir = dir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	})
}

// ensureOutputDir creates the output directory or checks if it's safe to use.
func ensureOutputDir(dir string, force bool) error {
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", dir)
		}
		if !force {
			return fmt.Errorf("directory %s already exists (use --force to overwrite)", dir)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll(dir, 0o755)
}

// downloadDirectory recursively downloads all files from a workspace path to a local directory.
func downloadDirectory(ctx context.Context, w *databricks.WorkspaceClient, remotePath, localDir string, force bool) (int, error) {
	// List all files recursively
	objects, err := w.Workspace.RecursiveList(ctx, remotePath)
	if err != nil {
		return 0, fmt.Errorf("failed to list workspace files: %w", err)
	}

	// Filter out directories, keep only files
	var files []workspace.ObjectInfo
	for _, obj := range objects {
		if obj.ObjectType != workspace.ObjectTypeDirectory {
			files = append(files, obj)
		}
	}

	if len(files) == 0 {
		return 0, errors.New("no files found in app source code path")
	}

	// Download files in parallel
	errs, errCtx := errgroup.WithContext(ctx)
	errs.SetLimit(10) // Limit concurrent downloads

	for _, file := range files {
		errs.Go(func() error {
			return downloadFile(errCtx, w, file, remotePath, localDir, force)
		})
	}

	if err := errs.Wait(); err != nil {
		return 0, err
	}

	return len(files), nil
}

// downloadFile downloads a single file from the workspace to the local directory.
func downloadFile(ctx context.Context, w *databricks.WorkspaceClient, file workspace.ObjectInfo, remotePath, localDir string, force bool) error {
	// Calculate relative path from the remote root
	relPath := strings.TrimPrefix(file.Path, remotePath)
	relPath = strings.TrimPrefix(relPath, "/")

	// Determine local file path
	localPath := filepath.Join(localDir, relPath)

	// Check if file exists
	if !force {
		if _, err := os.Stat(localPath); err == nil {
			return fmt.Errorf("file %s already exists (use --force to overwrite)", localPath)
		}
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", localPath, err)
	}

	// Download file content
	reader, err := w.Workspace.Download(ctx, file.Path)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", file.Path, err)
	}
	defer reader.Close()

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", localPath, err)
	}
	defer localFile.Close()

	// Copy content
	if _, err := io.Copy(localFile, reader); err != nil {
		return fmt.Errorf("failed to write %s: %w", localPath, err)
	}

	return nil
}
