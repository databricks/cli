package generate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewGenerateJobCommand() *cobra.Command {
	var configDir string
	var sourceDir string
	var jobId int64
	var force bool

	cmd := &cobra.Command{
		Use:   "job",
		Short: "Generate bundle configuration for a job",
	}

	cmd.Flags().Int64Var(&jobId, "existing-job-id", 0, `Job ID of the job to generate config for`)
	cmd.MarkFlagRequired("existing-job-id")

	cmd.Flags().StringVarP(&configDir, "config-dir", "d", "resources", `Dir path where the output config will be stored`)
	cmd.Flags().StringVarP(&sourceDir, "source-dir", "s", "src", `Dir path where the downloaded files will be stored`)
	cmd.Flags().BoolVarP(&force, "force", "f", false, `Force overwrite existing files in the output directory`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := root.MustConfigureBundle(cmd)
		if b == nil {
			return root.ErrAlreadyPrinted
		}

		w := b.WorkspaceClient()
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
			for _, task := range job.Settings.Tasks {
				err := downloader.MarkTaskForDownload(ctx, &task)
				if err != nil {
					return err
				}
			}
		}

		v, err := generate.ConvertJobToValue(job)
		if err != nil {
			return err
		}

		jobKey := cmd.Flag("key").Value.String()
		if jobKey == "" {
			jobKey = textutil.NormalizeString(job.Settings.Name)
		}

		result := map[string]dyn.Value{
			"resources": dyn.V(map[string]dyn.Value{
				"jobs": dyn.V(map[string]dyn.Value{
					jobKey: v,
				}),
			}),
		}

		err = downloader.FlushToDisk(ctx, force)
		if err != nil {
			return err
		}

		oldFilename := filepath.Join(configDir, jobKey+".yml")
		filename := filepath.Join(configDir, jobKey+".job.yml")

		// User might continuously run generate command to update their bundle jobs with any changes made in Databricks UI.
		// Due to changing in the generated file names, we need to first rename existing resource file to the new name.
		// Otherwise users can end up with duplicated resources.
		err = os.Rename(oldFilename, filename)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to rename file %s. DABs uses the resource type as a sub-extension for generated content, please rename it to %s, err: %w", oldFilename, filename, err)
		}

		saver := yamlsaver.NewSaverWithStyle(map[string]yaml.Style{
			// Including all JobSettings and nested fields which are map[string]string type
			"spark_conf":  yaml.DoubleQuotedStyle,
			"custom_tags": yaml.DoubleQuotedStyle,
			"tags":        yaml.DoubleQuotedStyle,
		})
		err = saver.SaveAsYAML(result, filename, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "Job configuration successfully saved to "+filename)
		return nil
	}

	return cmd
}
