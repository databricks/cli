package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/generate"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/config"
	"github.com/databricks/cli/libs/config/convert"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

func NewGenerateJobCommand() *cobra.Command {
	var outputDir string
	var force bool

	cmd := &cobra.Command{
		Use:     "job JOB_ID",
		Short:   "Generate bundle configuration for a job",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: root.MustConfigureBundle,
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "d", "", `Dir path where the output config and necessary files will be stored`)
	cmd.Flags().BoolVarP(&force, "force", "f", false, `Force overwrite existing files in the output directory`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b := bundle.Get(ctx)
		w := b.WorkspaceClient()

		// If no arguments are specified, prompt the user to select the job to generate.
		if len(args) == 0 && cmdio.IsInteractive(ctx) {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No JOB_ID argument specified. Loading names for Jobs drop-down."
			names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
			close(promptSpinner)

			if err != nil {
				return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "This field is required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}

		if !cmd.Flag("output-dir").Changed {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			input, err := cmdio.Ask(ctx, "Output dir", filepath.Join(wd, "resources"))
			if err != nil {
				return err
			}
			outputDir = input
		}

		var getReq jobs.GetJobRequest
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid JOB_ID: %s", args[0])
		}
		getReq.JobId = id

		job, err := w.Jobs.Get(ctx, getReq)
		if err != nil {
			return err
		}

		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return err
		}

		for _, task := range job.Settings.Tasks {
			err := downloadNotebookAndReplaceTaskPath(ctx, &task, w, outputDir, force)
			if err != nil {
				return err
			}
		}

		v, err := generate.ConvertJobToValue(job)
		if err != nil {
			return err
		}

		jobName := fmt.Sprintf("job_%s", convert.NormaliseString(job.Settings.Name))
		result := map[string]any{
			"resources": map[string]any{
				"jobs": map[string]config.Value{
					jobName: v,
				},
			},
		}

		err = saveConfigToFile(ctx, result, filepath.Join(outputDir, fmt.Sprintf("%s.yml", jobName)), force)
		if err != nil {
			return err
		}

		return nil
	}

	return cmd
}
