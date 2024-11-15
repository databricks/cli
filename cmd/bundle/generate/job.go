package generate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle/config/generate"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
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

	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	cmd.Flags().StringVarP(&configDir, "config-dir", "d", filepath.Join(wd, "resources"), `Dir path where the output config will be stored`)
	cmd.Flags().StringVarP(&sourceDir, "source-dir", "s", filepath.Join(wd, "src"), `Dir path where the downloaded files will be stored`)
	cmd.Flags().BoolVarP(&force, "force", "f", false, `Force overwrite existing files in the output directory`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := root.MustConfigureBundle(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		w := b.WorkspaceClient()
		job, err := w.Jobs.Get(ctx, jobs.GetJobRequest{JobId: jobId})
		if err != nil {
			return err
		}

		downloader := newDownloader(w, sourceDir, configDir)
		for _, task := range job.Settings.Tasks {
			err := downloader.MarkTaskForDownload(ctx, &task)
			if err != nil {
				return err
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

		oldFilename := filepath.Join(configDir, fmt.Sprintf("%s.yml", jobKey))
		filename := filepath.Join(configDir, fmt.Sprintf("%s.job.yml", jobKey))

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

		cmdio.LogString(ctx, fmt.Sprintf("Job configuration successfully saved to %s", filename))
		return nil
	}

	return cmd
}
