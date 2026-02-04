package apps

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

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
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

func newImportCommand() *cobra.Command {
	var name string
	var outputDir string
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

If no app name is specified, the command will list all available apps in your
workspace, with apps you own sorted to the top.

Examples:
  # Import an app (creates directory named after the app)
  databricks apps import --name my-streamlit-app

  # Import with custom output directory
  databricks apps import --name my-app --output-dir ~/my-apps/analytics

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

			// Run the import in the output directory
			err = runImport(ctx, w, name, outputDir, forceImport, currentUserEmail, quiet)
			if err != nil {
				return err
			}

			if !quiet {
				cmdio.LogString(ctx, fmt.Sprintf("\n✓ App '%s' has been successfully imported to %s", name, outputDir))
				cmdio.LogString(ctx, "\nYou can now deploy changes with: databricks bundle deploy")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the app to import (if not specified, lists all apps)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to output the bundle to (defaults to app name)")
	cmd.Flags().BoolVar(&forceImport, "force-import", false, "Force re-import of an app that was already imported (only works for apps you own)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress informational messages (only show errors and prompts)")

	return cmd
}

func runImport(ctx context.Context, w *databricks.WorkspaceClient, appName, outputDir string, forceImport bool, currentUserEmail string, quiet bool) error {
	// Step 1: Load the app from workspace
	if !quiet {
		cmdio.LogString(ctx, fmt.Sprintf("Loading app '%s' configuration", appName))
	}
	app, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: appName})
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

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
	// Use the app name for the bundle name
	bundleName := textutil.NormalizeString(app.Name)
	bundleConfig := map[string]dyn.Value{
		"bundle": dyn.NewValue(map[string]dyn.Value{
			"name": dyn.NewValue(bundleName, []dyn.Location{{Line: 1}}),
		}, []dyn.Location{{Line: 1}}),
		"workspace": dyn.NewValue(map[string]dyn.Value{
			"host": dyn.NewValue(w.Config.Host, []dyn.Location{{Line: 2}}),
		}, []dyn.Location{{Line: 10}}),
		"resources": dyn.NewValue(map[string]dyn.Value{
			"apps": dyn.V(map[string]dyn.Value{
				appKey: v,
			}),
		}, []dyn.Location{{Line: 20}}),
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
	return appKey, nil
}

// addBlankLinesBetweenTopLevelKeys adds blank lines between top-level sections in YAML
func addBlankLinesBetweenTopLevelKeys(filename string) error {
	// Read the file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Add blank lines before top-level keys (lines that don't start with space/tab and contain ':')
	var result []string
	for i, line := range lines {
		// Add blank line before top-level keys (except the first line)
		if i > 0 && len(line) > 0 && line[0] != ' ' && line[0] != '\t' && strings.Contains(line, ":") {
			result = append(result, "")
		}
		result = append(result, line)
	}

	// Write back to file
	file, err = os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range result {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}

// inlineAppConfigFile reads app.yml or app.yaml, inlines it into the app value, and returns the filename
func inlineAppConfigFile(appValue *dyn.Value) (string, error) {
	// Check for app.yml first, then app.yaml
	var appConfigFile string
	var appConfigData []byte
	var err error

	for _, filename := range []string{"app.yml", "app.yaml"} {
		if _, statErr := os.Stat(filename); statErr == nil {
			appConfigFile = filename
			appConfigData, err = os.ReadFile(filename)
			if err != nil {
				return "", fmt.Errorf("failed to read %s: %w", filename, err)
			}
			break
		}
	}

	// No app config file found
	if appConfigFile == "" {
		return "", nil
	}

	// Parse the app config
	var appConfig map[string]any
	err = yaml.Unmarshal(appConfigData, &appConfig)
	if err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", appConfigFile, err)
	}

	// Get the current app value as a map
	appMap, ok := appValue.AsMap()
	if !ok {
		return "", errors.New("app value is not a map")
	}

	// Build the new app map with the config section
	newPairs := make([]dyn.Pair, 0, len(appMap.Pairs())+2)

	// Copy existing pairs
	newPairs = append(newPairs, appMap.Pairs()...)

	// Create config section
	configMap := make(map[string]dyn.Value)

	// Add command if present
	if cmd, ok := appConfig["command"]; ok {
		cmdValue, err := convert.FromTyped(cmd, dyn.NilValue)
		if err != nil {
			return "", fmt.Errorf("failed to convert command: %w", err)
		}
		configMap["command"] = cmdValue
	}

	// Add env if present
	if env, ok := appConfig["env"]; ok {
		envValue, err := convert.FromTyped(env, dyn.NilValue)
		if err != nil {
			return "", fmt.Errorf("failed to convert env: %w", err)
		}
		configMap["env"] = envValue
	}

	// Add the config section if we have any items
	if len(configMap) > 0 {
		newPairs = append(newPairs, dyn.Pair{
			Key:   dyn.V("config"),
			Value: dyn.V(configMap),
		})
	}

	// Add resources at top level if present
	if resources, ok := appConfig["resources"]; ok {
		resourcesValue, err := convert.FromTyped(resources, dyn.NilValue)
		if err != nil {
			return "", fmt.Errorf("failed to convert resources: %w", err)
		}
		newPairs = append(newPairs, dyn.Pair{
			Key:   dyn.V("resources"),
			Value: resourcesValue,
		})
	}

	// Create the new app value with the config section
	newMapping := dyn.NewMappingFromPairs(newPairs)
	*appValue = dyn.NewValue(newMapping, appValue.Locations())

	return appConfigFile, nil
}
