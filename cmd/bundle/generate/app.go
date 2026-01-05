package generate

import (
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/cmd/bundle/deployment"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/textutil"
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

	cmd.Flags().StringVarP(&configDir, "config-dir", "d", "resources", `Directory path where the output bundle config will be stored`)
	cmd.Flags().StringVarP(&sourceDir, "source-dir", "s", "src/app", `Directory path where the app files will be stored`)
	cmd.Flags().BoolVarP(&force, "force", "f", false, `Force overwrite existing files in the output directory`)
	cmd.Flags().BoolVarP(&bind, "bind", "b", false, `automatically bind the generated app config to the existing app`)
	cmd.Flags().MarkHidden("bind")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := root.MustConfigureBundle(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		w := b.WorkspaceClient()
		cmdio.LogString(ctx, fmt.Sprintf("Loading app '%s' configuration", appName))
		app, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: appName})
		if err != nil {
			return err
		}

		outputFiler, err := filer.NewOutputFiler(ctx, w, b.BundleRootPath)
		if err != nil {
			return err
		}

		// Make sourceDir and configDir relative to the bundle root
		sourceDir, err = makeRelativeToRoot(b.BundleRootPath, sourceDir)
		if err != nil {
			return err
		}
		configDir, err = makeRelativeToRoot(b.BundleRootPath, configDir)
		if err != nil {
			return err
		}

		downloader := generate.NewDownloader(w, sourceDir, configDir, outputFiler)

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

		appKey := cmd.Flag("key").Value.String()
		if appKey == "" {
			appKey = textutil.NormalizeString(app.Name)
		}

		result := map[string]dyn.Value{
			"resources": dyn.V(map[string]dyn.Value{
				"apps": dyn.V(map[string]dyn.Value{
					appKey: v,
				}),
			}),
		}

		err = downloader.FlushToDisk(ctx, force)
		if err != nil {
			return err
		}

		filename := filepath.Join(configDir, appKey+".app.yml")

		saver := yamlsaver.NewSaver()
		err = saver.SaveAsYAMLToFiler(ctx, outputFiler, result, filename, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "App configuration successfully saved to "+filename)

		if bind {
			return deployment.BindResource(cmd, appKey, app.Name, true, false, true)
		}

		return nil
	}

	return cmd
}
