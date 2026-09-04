package generate

import (
	"path/filepath"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
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
		Long: `Generate bundle configuration for an existing pipeline.

This command downloads an existing pipeline's configuration and any associated
notebooks, creating bundle files that you can use to deploy the pipeline to other
environments or manage it as code.

Examples:
  # Import a production pipeline
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

	addOutputDirFlag(cmd, &configDir, "config-dir", "d", "resources", `Dir path where the output config will be stored`)
	addOutputDirFlag(cmd, &sourceDir, "source-dir", "s", "src", `Dir path where the downloaded files will be stored`)
	addForceFlag(cmd, &force, `Force overwrite existing files in the output directory`)
	addHiddenBindFlag(cmd, &bind, `automatically bind the generated resource to the existing resource`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, b, err := configureBundle(cmd)
		if err != nil {
			return err
		}

		w := b.WorkspaceClient(ctx)
		pipeline, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{PipelineId: pipelineId})
		if err != nil {
			return err
		}

		downloader := generate.NewDownloader(w, sourceDir, configDir)
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

		pipelineKey := selectedResourceKey(cmd, pipeline.Name)
		result := generatedResourceConfig("pipelines", pipelineKey, v)

		err = downloader.FlushToDisk(ctx, force)
		if err != nil {
			return err
		}

		// User might continuously run generate command to update their bundle jobs with any changes made in Databricks UI.
		// Due to changing in the generated file names, we need to first rename existing resource file to the new name.
		// Otherwise users can end up with duplicated resources.
		filename, err := migrateGeneratedResourceFilename(configDir, pipelineKey, "pipeline")
		if err != nil {
			return err
		}

		err = saveGeneratedResourceConfig(result, filename, force,
			// Including all CreatePipeline and nested fields which are map[string]string type
			map[string]yaml.Style{
				"spark_conf":    yaml.DoubleQuotedStyle,
				"custom_tags":   yaml.DoubleQuotedStyle,
				"configuration": yaml.DoubleQuotedStyle,
			},
		)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "Pipeline configuration successfully saved to "+filepath.ToSlash(filename))

		warnIfNotIncluded(ctx, b, filename)

		if bind {
			return bindGeneratedResource(cmd, pipelineKey, pipelineId)
		}

		return nil
	}

	return cmd
}
