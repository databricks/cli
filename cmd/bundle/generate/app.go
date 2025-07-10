package generate

import (
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
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
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := root.MustConfigureBundle(cmd)
		if b == nil {
			return root.ErrAlreadyPrinted
		}

		w := b.WorkspaceClient()
		cmdio.LogString(ctx, fmt.Sprintf("Loading app '%s' configuration", appName))
		app, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: appName})
		if err != nil {
			return err
		}

		downloader := generate.NewDownloader(w, sourceDir, configDir)

		sourceCodePath := app.DefaultSourceCodePath
		err = downloader.MarkDirectoryForDownload(ctx, &sourceCodePath)
		if err != nil {
			return err
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
		err = saver.SaveAsYAML(result, filename, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "App configuration successfully saved to "+filename)
		return nil
	}

	return cmd
}
