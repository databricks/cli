package generate

import (
	"errors"
	"fmt"
	"io/fs"
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
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewGeneratePipelineCommand() *cobra.Command {
	var configDir string
	var sourceDir string
	var pipelineId string
	var force bool
	var bind bool

	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Generate bundle configuration for a pipeline",
		Long: `Generate bundle configuration for an existing Delta Live Tables pipeline.

This command downloads an existing Lakeflow Declarative Pipeline's configuration and any associated
notebooks, creating bundle files that you can use to deploy the pipeline to other
environments or manage it as code.

Examples:
  # Import a production Lakeflow Declarative Pipeline
  databricks bundle generate pipeline --existing-pipeline-id abc123 --key etl_pipeline

  # Organize files in custom directories
  databricks bundle generate pipeline --existing-pipeline-id def456 \
    --key data_transformation --config-dir resources --source-dir src

  # Generate and automatically bind to the existing pipeline
  databricks bundle generate pipeline --existing-pipeline-id abc123 --key etl_pipeline --bind

What gets generated:
- Pipeline configuration YAML file with settings and libraries
- Pipeline notebooks downloaded to the source directory

After generation, you can deploy to other environments and modify settings
like catalogs, schemas, and compute configurations per target.`,
	}

	cmd.Flags().StringVar(&pipelineId, "existing-pipeline-id", "", `ID of the pipeline to generate config for`)
	cmd.MarkFlagRequired("existing-pipeline-id")

	cmd.Flags().StringVarP(&configDir, "config-dir", "d", "resources", `Dir path where the output config will be stored`)
	cmd.Flags().StringVarP(&sourceDir, "source-dir", "s", "src", `Dir path where the downloaded files will be stored`)
	cmd.Flags().BoolVarP(&force, "force", "f", false, `Force overwrite existing files in the output directory`)
	cmd.Flags().BoolVarP(&bind, "bind", "b", false, `automatically bind the generated resource to the existing resource`)
	cmd.Flags().MarkHidden("bind")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := root.MustConfigureBundle(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		w := b.WorkspaceClient()
		pipeline, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{PipelineId: pipelineId})
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
		for _, lib := range pipeline.Spec.Libraries {
			err := downloader.MarkPipelineLibraryForDownload(ctx, &lib)
			if err != nil {
				return err
			}
		}

		// If the root path is set, we need to download the files from the root path
		remoteRootPath := pipeline.Spec.RootPath
		if pipeline.Spec.RootPath != "" {
			err := downloader.MarkDirectoryForDownload(ctx, &pipeline.Spec.RootPath)
			if err != nil {
				return err
			}
		}

		// Making sure the root path is relative to the config directory.
		rel, err := filepath.Rel(configDir, sourceDir)
		if err != nil {
			return err
		}

		v, err := generate.ConvertPipelineToValue(pipeline.Spec, filepath.ToSlash(rel), remoteRootPath)
		if err != nil {
			return err
		}

		pipelineKey := cmd.Flag("key").Value.String()
		if pipelineKey == "" {
			pipelineKey = textutil.NormalizeString(pipeline.Name)
		}

		result := map[string]dyn.Value{
			"resources": dyn.V(map[string]dyn.Value{
				"pipelines": dyn.V(map[string]dyn.Value{
					pipelineKey: v,
				}),
			}),
		}

		err = downloader.FlushToDisk(ctx, force)
		if err != nil {
			return err
		}

		oldFilename := filepath.Join(configDir, pipelineKey+".yml")
		filename := filepath.Join(configDir, pipelineKey+".pipeline.yml")

		// User might continuously run generate command to update their bundle jobs with any changes made in Databricks UI.
		// Due to changing in the generated file names, we need to first rename existing resource file to the new name.
		// Otherwise users can end up with duplicated resources.
		err = filerRename(ctx, outputFiler, oldFilename, filename)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to rename file %s. DABs uses the resource type as a sub-extension for generated content, please rename it to %s, err: %w", oldFilename, filename, err)
		}

		saver := yamlsaver.NewSaverWithStyle(
			// Including all CreatePipeline and nested fields which are map[string]string type
			map[string]yaml.Style{
				"spark_conf":    yaml.DoubleQuotedStyle,
				"custom_tags":   yaml.DoubleQuotedStyle,
				"configuration": yaml.DoubleQuotedStyle,
			},
		)
		err = saver.SaveAsYAMLToFiler(ctx, outputFiler, result, filename, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "Pipeline configuration successfully saved to "+filename)

		if bind {
			return deployment.BindResource(cmd, pipelineKey, pipelineId, true, false, true)
		}

		return nil
	}

	return cmd
}
