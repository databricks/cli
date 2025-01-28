package generate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/databricks/cli/bundle/config/generate"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"

	"gopkg.in/yaml.v3"
)

func NewGenerateAppCommand() *cobra.Command {
	var configDir string
	var sourceDir string
	var appName string
	var force bool

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Generate bundle configuration for a Databricks app",
	}

	cmd.Flags().StringVar(&appName, "existing-app-name", "", `App name to generate config for`)
	cmd.MarkFlagRequired("existing-app-name")

	cmd.Flags().StringVarP(&configDir, "config-dir", "d", "resources", `Directory path where the output bundle config will be stored`)
	cmd.Flags().StringVarP(&sourceDir, "source-dir", "s", "src/app", `Directory path where the app files will be stored`)
	cmd.Flags().BoolVarP(&force, "force", "f", false, `Force overwrite existing files in the output directory`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := root.MustConfigureBundle(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		w := b.WorkspaceClient()
		cmdio.LogString(ctx, fmt.Sprintf("Loading app '%s' configuration", appName))
		app, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: appName})
		if err != nil {
			return err
		}

		// Making sure the config directory and source directory are absolute paths.
		if !filepath.IsAbs(configDir) {
			configDir = filepath.Join(b.BundleRootPath, configDir)
		}

		if !filepath.IsAbs(sourceDir) {
			sourceDir = filepath.Join(b.BundleRootPath, sourceDir)
		}

		downloader := newDownloader(w, sourceDir, configDir)

		sourceCodePath := app.DefaultSourceCodePath
		err = downloader.markDirectoryForDownload(ctx, &sourceCodePath)
		if err != nil {
			return err
		}

		appConfig, err := getAppConfig(ctx, app, w)
		if err != nil {
			return fmt.Errorf("failed to get app config: %w", err)
		}

		// Making sure the source code path is relative to the config directory.
		rel, err := filepath.Rel(configDir, sourceDir)
		if err != nil {
			return err
		}

		v, err := generate.ConvertAppToValue(app, filepath.ToSlash(rel), appConfig)
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

		// If there are app.yaml or app.yml files in the source code path, they will be downloaded but we don't want to include them in the bundle.
		// We include this configuration inline, so we need to remove these files.
		for _, configFile := range []string{"app.yml", "app.yaml"} {
			delete(downloader.files, filepath.Join(sourceDir, configFile))
		}

		err = downloader.FlushToDisk(ctx, force)
		if err != nil {
			return err
		}

		filename := filepath.Join(configDir, appKey+".app.yml")

		saver := yamlsaver.NewSaver()
		err = saver.SaveAsYAML(result, filename, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "App configuration successfully saved to "+filename)
		return nil
	}

	return cmd
}

func getAppConfig(ctx context.Context, app *apps.App, w *databricks.WorkspaceClient) (map[string]any, error) {
	sourceCodePath := app.DefaultSourceCodePath

	f, err := filer.NewWorkspaceFilesClient(w, sourceCodePath)
	if err != nil {
		return nil, err
	}

	// The app config is stored in app.yml or app.yaml file in the source code path.
	configFileNames := []string{"app.yml", "app.yaml"}
	for _, configFile := range configFileNames {
		r, err := f.Read(ctx, configFile)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		defer r.Close()

		cmdio.LogString(ctx, "Reading app configuration from "+configFile)
		content, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}

		var appConfig map[string]any
		err = yaml.Unmarshal(content, &appConfig)
		if err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("Failed to parse app configuration:\n%s\nerr: %v", string(content), err))
			return nil, nil
		}

		return appConfig, nil
	}

	return nil, nil
}
