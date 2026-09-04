package generate

import (
	"path/filepath"
	"strconv"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func NewGenerateJobCommand() *cobra.Command {
	var configDir string
	var sourceDir string
	var jobId int64
	var force bool
	var bind bool

	cmd := &cobra.Command{
		Use:   "job",
		Short: "Generate bundle configuration for a job",
		Long: `Generate bundle configuration for an existing Databricks job.

This command downloads an existing job's configuration and creates bundle files
that you can use to deploy the job to other environments or manage it as code.

Examples:
  # Import a production job for version control
  databricks bundle generate job --existing-job-id 12345 --key my_etl_job

  # Specify custom directories for organization
  databricks bundle generate job --existing-job-id 67890 \
    --key data_pipeline --config-dir resources --source-dir src

  # Generate and automatically bind to the existing job
  databricks bundle generate job --existing-job-id 12345 --key my_etl_job --bind

What gets generated:
- Job configuration YAML file in the resources directory
- Any associated notebook or Python files in the source directory

After generation, you can deploy this job to other targets using:
  databricks bundle deploy --target staging
  databricks bundle deploy --target prod`,
	}

	cmd.Flags().Int64Var(&jobId, "existing-job-id", 0, `Job ID of the job to generate config for`)
	cmd.MarkFlagRequired("existing-job-id")

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
		job, err := w.Jobs.Get(ctx, jobs.GetJobRequest{JobId: jobId})
		if err != nil {
			return err
		}

		downloader := generate.NewDownloader(w, sourceDir, configDir)

		// Don't download files if the job is using Git source
		// When Git source is used, the job will be using the files from the Git repository
		// but specific tasks might override this behaviour by using `source: WORKSPACE` setting.
		// In this case, we don't want to download the files as well for these specific tasks
		// because it leads to confusion with relative paths between workspace and GIT files.
		// Instead we keep these tasks as is and let the user handle the files manually.
		// The configuration will be deployable as tasks paths for source: WORKSPACE tasks will be absolute workspace paths.
		if job.Settings.GitSource != nil {
			cmdio.LogString(ctx, "Job is using Git source, skipping downloading files")
		} else {
			err = downloader.MarkTasksForDownload(ctx, job.Settings.Tasks)
			if err != nil {
				return err
			}
		}

		v, err := generate.ConvertJobToValue(job)
		if err != nil {
			return err
		}

		jobKey := selectedResourceKey(cmd, job.Settings.Name)
		result := generatedResourceConfig("jobs", jobKey, v)

		err = downloader.FlushToDisk(ctx, force)
		if err != nil {
			return err
		}

		downloader.CleanupOldFiles(ctx)

		// User might continuously run generate command to update their bundle jobs with any changes made in Databricks UI.
		// Due to changing in the generated file names, we need to first rename existing resource file to the new name.
		// Otherwise users can end up with duplicated resources.
		filename, err := migrateGeneratedResourceFilename(configDir, jobKey, "job")
		if err != nil {
			return err
		}

		err = saveGeneratedResourceConfig(result, filename, force, map[string]yaml.Style{
			// Including all JobSettings and nested fields which are map[string]string type
			"spark_conf":  yaml.DoubleQuotedStyle,
			"custom_tags": yaml.DoubleQuotedStyle,
			"tags":        yaml.DoubleQuotedStyle,
		})
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "Job configuration successfully saved to "+filepath.ToSlash(filename))

		warnIfNotIncluded(ctx, b, filename)

		if bind {
			return bindGeneratedResource(cmd, jobKey, strconv.FormatInt(jobId, 10))
		}

		return nil
	}

	return cmd
}
