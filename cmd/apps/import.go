package apps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	bundleutils "github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

func newImportCommand() *cobra.Command {
	var name string
	var outputDir string
	var cleanup bool
	var forceImport bool
	var quiet bool

	cmd := &cobra.Command{
		Use:   "import",
		Short: "(Experimental) Import an existing Databricks app as a bundle",
		Long: `(Experimental) Import an existing Databricks app and convert it to a bundle configuration.

This command creates a new bundle directory with the app configuration, downloads
the app source code, binds the bundle to the existing app, and deploys it using
direct deployment mode. This allows you to manage the app as code going forward.

The command will:
1. Create an empty bundle folder with databricks.yml
2. Download the app and add it to databricks.yml
3. Bind the bundle to the existing app
4. Deploy the bundle in direct mode
5. Start the app
6. Optionally clean up the previous app folder (if --cleanup is set)

If no app name is specified, the command will list all available apps in your
workspace, with apps you own sorted to the top.

Examples:
  # Import an app (creates directory named after the app)
  databricks apps import --name my-streamlit-app

  # Import with custom output directory
  databricks apps import --name my-app --output-dir ~/my-apps/analytics

  # Import and clean up the old app folder
  databricks apps import --name my-app --cleanup

  # Force re-import of your own app (only works for apps you own)
  databricks apps import --name my-app --force-import

  # Silent mode (only show errors and prompts)
  databricks apps import --name my-app -q

  # List available apps (interactive selection)
  databricks apps import`,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			// Get current user to filter apps
			me, err := w.CurrentUser.Me(ctx)
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}
			currentUserEmail := strings.ToLower(me.UserName)

			// If no app name provided, list apps and let user select
			if name == "" {
				// List all apps
				spinner := cmdio.NewSpinner(ctx)
				spinner.Update("Loading available apps...")
				allApps := w.Apps.List(ctx, apps.ListAppsRequest{})

				// Collect all apps
				var appList []apps.App
				for allApps.HasNext(ctx) {
					app, err := allApps.Next(ctx)
					if err != nil {
						spinner.Close()
						return fmt.Errorf("failed to iterate apps: %w", err)
					}
					appList = append(appList, app)
				}
				spinner.Close()

				if len(appList) == 0 {
					return errors.New("no apps found in workspace")
				}

				// Sort apps: owned by current user first
				sort.Slice(appList, func(i, j int) bool {
					iOwned := strings.ToLower(appList[i].Creator) == currentUserEmail
					jOwned := strings.ToLower(appList[j].Creator) == currentUserEmail
					if iOwned != jOwned {
						return iOwned
					}
					return appList[i].Name < appList[j].Name
				})

				// Build selection map
				names := make(map[string]string)
				for _, app := range appList {
					owner := app.Creator
					if owner == "" {
						owner = "unknown"
					}
					// Extract just the username from email if it's an email
					if idx := strings.Index(owner, "@"); idx > 0 {
						owner = owner[:idx]
					}
					label := fmt.Sprintf("%s (owner: %s)", app.Name, owner)
					names[label] = app.Name
				}

				// Prompt user to select
				if !cmdio.IsPromptSupported(ctx) {
					return errors.New("app name must be specified when prompts are not supported")
				}

				selectedLabel, err := cmdio.Select(ctx, names, "Select an app to import")
				if err != nil {
					return err
				}
				name = selectedLabel
			}

			// If output directory is not set, default to the app name
			if outputDir == "" {
				outputDir = name
			}

			// Get absolute path for output directory
			outputDir, err = filepath.Abs(outputDir)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}

			// Check if output directory already exists
			if _, err := os.Stat(outputDir); err == nil {
				return fmt.Errorf("directory '%s' already exists. Please remove it or choose a different output directory", outputDir)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("failed to check if directory exists: %w", err)
			}

			// Create output directory
			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// Save the workspace path for cleanup
			var oldSourceCodePath string

			// Run the import in the output directory
			err = runImport(ctx, w, name, outputDir, &oldSourceCodePath, forceImport, currentUserEmail, quiet)
			if err != nil {
				return err
			}

			// Clean up the previous app folder if requested
			if cleanup && oldSourceCodePath != "" {
				if !quiet {
					cmdio.LogString(ctx, "Cleaning up previous app folder")
				}

				err = w.Workspace.Delete(ctx, workspace.Delete{
					Path:      oldSourceCodePath,
					Recursive: true,
				})
				if err != nil {
					// Log warning but don't fail
					cmdio.LogString(ctx, fmt.Sprintf("Warning: failed to clean up app folder %s: %v", oldSourceCodePath, err))
				} else if !quiet {
					cmdio.LogString(ctx, "Cleaned up app folder: "+oldSourceCodePath)
				}
			}

			if !quiet {
				cmdio.LogString(ctx, fmt.Sprintf("\n✓ App '%s' has been successfully imported to %s", name, outputDir))
				if cleanup && oldSourceCodePath != "" {
					cmdio.LogString(ctx, "✓ Previous app folder has been cleaned up")
				}
				cmdio.LogString(ctx, "\nYou can now deploy changes with: databricks bundle deploy")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the app to import (if not specified, lists all apps)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to output the bundle to (defaults to app name)")
	cmd.Flags().BoolVar(&cleanup, "cleanup", false, "Clean up the previous app folder and all its contents")
	cmd.Flags().BoolVar(&forceImport, "force-import", false, "Force re-import of an app that was already imported (only works for apps you own)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress informational messages (only show errors and prompts)")

	return cmd
}

// runImport orchestrates the app import process: loads the app from workspace,
// generates bundle files, binds to the existing app, deploys, and starts it.
func runImport(ctx context.Context, w *databricks.WorkspaceClient, appName, outputDir string, oldSourceCodePath *string, forceImport bool, currentUserEmail string, quiet bool) error {
	// Step 1: Load the app from workspace
	if !quiet {
		cmdio.LogString(ctx, fmt.Sprintf("Loading app '%s' configuration", appName))
	}
	app, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: appName})
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Save the old source code path for cleanup
	*oldSourceCodePath = app.DefaultSourceCodePath

	// Check if the app's source code path is inside a .bundle folder (indicating it was already imported)
	alreadyImported := app.DefaultSourceCodePath != "" && strings.Contains(app.DefaultSourceCodePath, "/.bundle/")
	if alreadyImported {
		if !forceImport {
			return fmt.Errorf("app '%s' appears to have already been imported (workspace path '%s' is inside a .bundle folder). Use --force-import to import anyway", appName, app.DefaultSourceCodePath)
		}

		// Check if the app is owned by the current user
		appOwner := strings.ToLower(app.Creator)
		if appOwner != currentUserEmail {
			return fmt.Errorf("--force-import can only be used for apps you own. App '%s' is owned by '%s'", appName, app.Creator)
		}

		if !quiet {
			cmdio.LogString(ctx, fmt.Sprintf("Warning: App '%s' appears to have already been imported, but proceeding due to --force-import", appName))
		}
	}

	// Change to output directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(outputDir); err != nil {
		return fmt.Errorf("failed to change to output directory: %w", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	// Step 2: Generate bundle files
	if !quiet {
		cmdio.LogString(ctx, "Creating bundle configuration")
	}

	// Use the bundle generate app command logic
	appKey, err := generateAppBundle(ctx, w, app, quiet)
	if err != nil {
		return fmt.Errorf("failed to generate bundle: %w", err)
	}

	// Set DATABRICKS_BUNDLE_ENGINE to direct mode
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENGINE", "direct")

	// Step 3: Bind the bundle to the existing app (skip if already imported)
	var b *bundle.Bundle
	if !alreadyImported {
		if !quiet {
			cmdio.LogString(ctx, "Binding bundle to existing app")
		}

		// Create a command for binding with required flags
		bindCmd := &cobra.Command{}
		bindCmd.SetContext(ctx)
		bindCmd.Flags().StringSlice("var", []string{}, "set values for variables defined in bundle config")

		// Initialize the bundle
		var err error
		b, err = bundleutils.ProcessBundle(bindCmd, bundleutils.ProcessOptions{
			SkipInitContext: true,
			ReadState:       true,
			InitFunc: func(b *bundle.Bundle) {
				b.Config.Bundle.Deployment.Lock.Force = false
			},
		})
		if err != nil {
			return fmt.Errorf("failed to initialize bundle: %w", err)
		}

		// Find the app resource
		resource, err := b.Config.Resources.FindResourceByConfigKey(appKey)
		if err != nil {
			return fmt.Errorf("failed to find resource: %w", err)
		}

		// Verify the app exists
		exists, err := resource.Exists(ctx, b.WorkspaceClient(), app.Name)
		if err != nil {
			return fmt.Errorf("failed to verify app exists: %w", err)
		}
		if !exists {
			return fmt.Errorf("app '%s' no longer exists in workspace", app.Name)
		}

		// Bind the resource
		tfName := terraform.GroupToTerraformName[resource.ResourceDescription().PluralName]
		phases.Bind(ctx, b, &terraform.BindOptions{
			AutoApprove:  true,
			ResourceType: tfName,
			ResourceKey:  appKey,
			ResourceId:   app.Name,
		})
		if logdiag.HasError(ctx) {
			return errors.New("failed to bind resource")
		}

		if !quiet {
			cmdio.LogString(ctx, fmt.Sprintf("Successfully bound to app '%s'", app.Name))
		}
	} else if !quiet {
		cmdio.LogString(ctx, "Skipping bind step (app already imported)")
	}

	// Step 4: Deploy the bundle
	if !quiet {
		cmdio.LogString(ctx, "Deploying bundle")
	}

	// Create a new command for deployment
	deployCmd := &cobra.Command{}
	deployCmd.SetContext(ctx)
	deployCmd.Flags().StringSlice("var", []string{}, "set values for variables defined in bundle config")

	// Process the bundle (deploy)
	b, err = bundleutils.ProcessBundle(deployCmd, bundleutils.ProcessOptions{
		SkipInitContext: true,
		Deploy:          true,
		FastValidate:    true,
		AlwaysPull:      true,
		InitFunc: func(b *bundle.Bundle) {
			b.AutoApprove = true
		},
	})
	if err != nil {
		return fmt.Errorf("failed to deploy bundle: %w", err)
	}

	if !quiet {
		cmdio.LogString(ctx, "Bundle deployed successfully")
	}

	// Step 5: Run the app (equivalent to "databricks bundle run app")
	if !quiet {
		cmdio.LogString(ctx, "Starting app")
	}

	// Locate the app resource
	ref, err := resources.Lookup(b, appKey, run.IsRunnable)
	if err != nil {
		return fmt.Errorf("failed to find app resource: %w", err)
	}

	// Convert the resource to a runner
	runner, err := run.ToRunner(b, ref)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Run the app with default options
	runOptions := &run.Options{}
	output, err := runner.Run(ctx, runOptions)
	if err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	if output != nil {
		resultString, err := output.String()
		if err != nil {
			return fmt.Errorf("failed to get run output: %w", err)
		}
		if !quiet {
			cmdio.LogString(ctx, resultString)
		}
	}

	if !quiet {
		cmdio.LogString(ctx, "App started successfully")
	}
	return nil
}

// generateAppBundle creates a databricks.yml configuration file and downloads the app source code.
func generateAppBundle(ctx context.Context, w *databricks.WorkspaceClient, app *apps.App, quiet bool) (string, error) {
	// Use constant "app" as the resource key
	appKey := "app"

	// App source code goes to root directory
	sourceDir := "."
	downloader := generate.NewDownloader(w, sourceDir, ".")

	// Download app source code if it exists
	sourceCodePath := app.DefaultSourceCodePath
	if sourceCodePath != "" {
		err := downloader.MarkDirectoryForDownload(ctx, &sourceCodePath)
		if err != nil {
			return "", fmt.Errorf("failed to mark directory for download: %w", err)
		}
	}

	// Convert app to value
	v, err := generate.ConvertAppToValue(app, sourceDir)
	if err != nil {
		return "", fmt.Errorf("failed to convert app to value: %w", err)
	}

	// Check for app.yml or app.yaml and inline its contents
	appConfigFile, err := inlineAppConfigFile(&v)
	if err != nil {
		return "", fmt.Errorf("failed to inline app config: %w", err)
	}

	// Delete the app config file if we inlined it
	if appConfigFile != "" {
		err = os.Remove(appConfigFile)
		if err != nil {
			return "", fmt.Errorf("failed to remove %s: %w", appConfigFile, err)
		}
		if !quiet {
			cmdio.LogString(ctx, "Inlined and removed "+appConfigFile)
		}
	}

	// Create the bundle configuration with explicit line numbers to control ordering
	bundleName := textutil.NormalizeString(app.Name)
	bundleConfig := map[string]dyn.Value{
		"bundle": dyn.NewValue(map[string]dyn.Value{
			"name": dyn.NewValue(bundleName, []dyn.Location{{Line: 1}}),
		}, []dyn.Location{{Line: 1}}),
		"workspace": dyn.NewValue(map[string]dyn.Value{
			"host": dyn.NewValue(w.Config.Host, []dyn.Location{{Line: 2}}),
		}, []dyn.Location{{Line: 2}}),
		"resources": dyn.NewValue(map[string]dyn.Value{
			"apps": dyn.V(map[string]dyn.Value{
				appKey: v,
			}),
		}, []dyn.Location{{Line: 4}}),
	}

	// Download the app source files
	err = downloader.FlushToDisk(ctx, false)
	if err != nil {
		return "", fmt.Errorf("failed to download app source: %w", err)
	}

	// Save databricks.yml
	databricksYml := filepath.Join(".", "databricks.yml")
	saver := yamlsaver.NewSaver()
	err = saver.SaveAsYAML(bundleConfig, databricksYml, false)
	if err != nil {
		return "", fmt.Errorf("failed to save databricks.yml: %w", err)
	}

	// Add blank lines between top-level keys for better readability
	err = addBlankLinesBetweenTopLevelKeys(databricksYml)
	if err != nil {
		return "", fmt.Errorf("failed to format databricks.yml: %w", err)
	}

	if !quiet {
		cmdio.LogString(ctx, "Bundle configuration created at "+databricksYml)
	}

	// Generate .env file from app.yml if it exists
	err = generateEnvFile(ctx, w.Config.Host, w.Config.Profile, app, quiet)
	if err != nil {
		// Log warning but don't fail - .env is optional
		if !quiet {
			cmdio.LogString(ctx, fmt.Sprintf("⚠ Failed to generate .env file: %v", err))
		}
	}

	return appKey, nil
}

// buildResourcesMap creates a map of resource names to their IDs/values from app.Resources.
func buildResourcesMap(app *apps.App) map[string]string {
	resources := make(map[string]string)
	if app.Resources == nil {
		return resources
	}

	for _, resource := range app.Resources {
		if resource.Name == "" {
			continue
		}

		// Extract the resource ID/value based on type
		var value string
		switch {
		case resource.SqlWarehouse != nil:
			value = resource.SqlWarehouse.Id
		case resource.ServingEndpoint != nil:
			value = resource.ServingEndpoint.Name
		case resource.Experiment != nil:
			value = resource.Experiment.ExperimentId
		case resource.Database != nil:
			value = resource.Database.DatabaseName
		case resource.Secret != nil:
			value = resource.Secret.Key
		case resource.GenieSpace != nil:
			value = resource.GenieSpace.SpaceId
		case resource.Job != nil:
			value = resource.Job.Id
		case resource.UcSecurable != nil:
			value = resource.UcSecurable.SecurableFullName
		}

		if value != "" {
			resources[resource.Name] = value
		}
	}

	return resources
}

// generateEnvFile generates a .env file from app.yml and app resources.
func generateEnvFile(ctx context.Context, host, profile string, app *apps.App, quiet bool) error {
	// Check if app.yml or app.yaml exists
	var appYmlPath string
	for _, filename := range []string{"app.yml", "app.yaml"} {
		if _, err := os.Stat(filename); err == nil {
			appYmlPath = filename
			break
		}
	}

	if appYmlPath == "" {
		// No app.yml found, skip .env generation
		return nil
	}

	// Build resources map from app.Resources
	resources := buildResourcesMap(app)

	// Create EnvFileBuilder
	builder, err := NewEnvFileBuilder(host, profile, app.Name, appYmlPath, resources)
	if err != nil {
		return fmt.Errorf("failed to create env builder: %w", err)
	}

	// Write .env file
	err = builder.WriteEnvFile(".")
	if err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	if !quiet {
		cmdio.LogString(ctx, "✓ Generated .env file from app.yml")
	}

	// Write .gitignore if it doesn't exist
	if err := writeGitignoreIfMissing(ctx, "."); err != nil && !quiet {
		cmdio.LogString(ctx, fmt.Sprintf("⚠ Failed to create .gitignore: %v", err))
	}

	return nil
}
