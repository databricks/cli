package apps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
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
  databricks apps import

  # Import a specific app's source code
  databricks apps import --name my-app

  # Import to a specific directory
  databricks apps import --name my-app --output-dir ./projects

  # Force overwrite existing files
  databricks apps import --name my-app --force`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Prompt for app name if not provided
			if appName == "" {
				selected, err := prompt.PromptForAppSelection(ctx, "Select an app to import")
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
	err := prompt.RunWithSpinnerCtx(ctx, fmt.Sprintf("Fetching app '%s'...", opts.appName), func() error {
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

	cmdio.LogString(ctx, "Source code path: "+app.DefaultSourceCodePath)

	// Step 3: Create output directory
	destDir := opts.appName
	if opts.outputDir != "" {
		destDir = filepath.Join(opts.outputDir, opts.appName)
	}
	if err := ensureOutputDir(destDir, opts.force); err != nil {
		return err
	}

	// Step 4: Download files using the Downloader
	downloader := generate.NewDownloader(w, destDir, destDir)
	sourceCodePath := app.DefaultSourceCodePath

	err = prompt.RunWithSpinnerCtx(ctx, "Downloading files...", func() error {
		if markErr := downloader.MarkDirectoryForDownload(ctx, &sourceCodePath); markErr != nil {
			return fmt.Errorf("failed to list files: %w", markErr)
		}
		return downloader.FlushToDisk(ctx, opts.force)
	})
	if err != nil {
		return fmt.Errorf("failed to download files for app '%s': %w", opts.appName, err)
	}

	// Count downloaded files
	fileCount := countFiles(destDir)

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
	prompt.PrintSuccess(opts.appName, absDestDir, fileCount, true)

	return nil
}

// runNpmInstallInDir runs npm install in the specified directory.
func runNpmInstallInDir(ctx context.Context, dir string) error {
	if _, err := exec.LookPath("npm"); err != nil {
		return errors.New("npm not found: please install Node.js")
	}

	return prompt.RunWithSpinnerCtx(ctx, "Installing dependencies...", func() error {
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

// countFiles counts the number of files (non-directories) in a directory tree.
func countFiles(dir string) int {
	count := 0
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}
