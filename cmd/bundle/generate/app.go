package generate

import (
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

func NewGenerateAppCommand() *cobra.Command {
	var configDir string
	var sourceDir string
	var appName string
	var force bool
	var bind bool

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Generate bundle configuration for a Databricks app",
		Long: `Generate bundle configuration for an existing Databricks app.

This command downloads an existing Databricks app and creates bundle files
that you can use to deploy the app to other environments or manage it as code.

Examples:
  # Import a Streamlit app
  databricks bundle generate app --existing-app-name my-streamlit-app --key analytics_app

  # Import with custom directory structure
  databricks bundle generate app --existing-app-name data-viewer \
    --key data_app --config-dir resources --source-dir src/apps

  # Generate and automatically bind to the existing app
  databricks bundle generate app --existing-app-name my-app --key analytics_app --bind

What gets generated:
- App configuration YAML file with app settings and dependencies
- App source files downloaded to the specified source directory
- Updated bundle configuration to reference the new app resource

After generation, you can deploy the app to different environments and modify
settings like compute resources, environment variables, and access permissions
per target environment.`,
	}

	cmd.Flags().StringVar(&appName, "existing-app-name", "", `App name to generate config for`)
	cmd.MarkFlagRequired("existing-app-name")

	addOutputDirFlag(cmd, &configDir, "config-dir", "d", "resources", `Directory path where the output bundle config will be stored`)
	addOutputDirFlag(cmd, &sourceDir, "source-dir", "s", "src/app", `Directory path where the app files will be stored`)
	addForceFlag(cmd, &force, `Force overwrite existing files in the output directory`)
	addHiddenBindFlag(cmd, &bind, `automatically bind the generated app config to the existing app`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, b, err := configureBundle(cmd)
		if err != nil {
			return err
		}

		w := b.WorkspaceClient(ctx)
		cmdio.LogString(ctx, fmt.Sprintf("Loading app '%s' configuration", appName))
		app, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: appName})
		if err != nil {
			return err
		}

		downloader := generate.NewDownloader(w, sourceDir, configDir)

		sourceCodePath := app.DefaultSourceCodePath
		// If the source code path is not set, we don't need to download anything.
		// This is the case for apps that are not yet deployed.
		if sourceCodePath != "" {
			err = downloader.MarkDirectoryForDownload(ctx, &sourceCodePath)
			if err != nil {
				return err
			}
		}

		// Making sure the source code path is relative to the config directory.
		rel, err := filepath.Rel(configDir, sourceDir)
		if err != nil {
			return err
		}

		v, err := generate.ConvertAppToValue(app, filepath.ToSlash(rel))
		if err != nil {
			return err
		}

		appKey := selectedResourceKey(cmd, app.Name)
		result := generatedResourceConfig("apps", appKey, v)

		err = downloader.FlushToDisk(ctx, force)
		if err != nil {
			return err
		}

		filename := filepath.Join(configDir, appKey+".app.yml")

		err = saveGeneratedResourceConfig(result, filename, force, nil)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "App configuration successfully saved to "+filename)

		warnIfNotIncluded(ctx, b, filename)

		if bind {
			return bindGeneratedResource(cmd, appKey, app.Name)
		}

		return nil
	}

	return cmd
}
