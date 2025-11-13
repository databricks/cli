package debug

import (
	"encoding/json"
	"strconv"

	"github.com/databricks/cli/bundle/resourcesnapshot"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/r3labs/diff/v3"
	"github.com/spf13/cobra"
)

type ResourceDiff struct {
	Changes diff.Changelog `json:"changes"`
}

type DiffOutput struct {
	Jobs      map[string]*ResourceDiff `json:"jobs,omitempty"`
	Pipelines map[string]*ResourceDiff `json:"pipelines,omitempty"`
}

func NewExpDiffCommand() *cobra.Command {
	var save bool

	cmd := &cobra.Command{
		Use:   "exp-diff",
		Short: "Show differences between current remote state and last deploy snapshot (experimental)",
		Long: `Show differences between the current remote resource state and the state at the time of the last successful deploy.

This command compares the current state of deployed resources (jobs, pipelines) with snapshots
saved during the last successful deploy. It helps identify configuration drift caused by manual
changes or external modifications.

Note: This command is experimental and may change without notice.`,
		Args: root.NoArgs,
	}

	cmd.Flags().BoolVar(&save, "save", false, "Save the diff back to the bundle YAML files")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Load bundle with resource IDs from state
		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			InitIDs: true,
		})
		if err != nil {
			return err
		}

		// Load previous snapshots
		snapshot, err := resourcesnapshot.Load(ctx, b)
		if err != nil {
			return err
		}

		output := &DiffOutput{
			Jobs:      make(map[string]*ResourceDiff),
			Pipelines: make(map[string]*ResourceDiff),
		}

		// If no snapshot exists, return empty diff
		if snapshot == nil {
			log.Debugf(ctx, "No previous snapshot found, skipping diff")
			return writeOutput(cmd, output)
		}

		w := b.WorkspaceClient()

		// Create diff writer if save flag is set
		var writer *DiffWriter
		if save {
			writer = NewDiffWriter(b)
		}

		// Compare jobs
		for key, job := range b.Config.Resources.Jobs {
			if job.ID == "" {
				log.Debugf(ctx, "Skipping job %s: no ID (not deployed)", key)
				continue
			}

			// Check if we have a previous snapshot for this resource
			previousJob, ok := snapshot.Jobs[key]
			if !ok {
				log.Debugf(ctx, "Skipping job %s: no previous snapshot", key)
				continue
			}

			jobID, err := strconv.ParseInt(job.ID, 10, 64)
			if err != nil {
				log.Warnf(ctx, "Skipping job %s: invalid ID %q: %v", key, job.ID, err)
				continue
			}

			// Fetch current remote state
			currentJob, err := w.Jobs.Get(ctx, jobs.GetJobRequest{
				JobId: jobID,
			})
			if err != nil {
				log.Warnf(ctx, "Failed to fetch job %s (ID: %d): %v", key, jobID, err)
				continue
			}

			// Compare previous and current state
			changelog, err := diff.Diff(previousJob, currentJob)
			if err != nil {
				log.Warnf(ctx, "Failed to diff job %s: %v", key, err)
				continue
			}

			// Only add to output if there are changes
			if len(changelog) > 0 {
				output.Jobs[key] = &ResourceDiff{
					Changes: changelog,
				}
				log.Debugf(ctx, "Found %d changes for job %s", len(changelog), key)

				// Save changes back to YAML if save flag is set
				if writer != nil {
					err = writer.WriteJobDiff(ctx, key, currentJob)
					if err != nil {
						log.Warnf(ctx, "Failed to save job %s changes: %v", key, err)
					}
				}
			}
		}

		// Compare pipelines
		for key, pipeline := range b.Config.Resources.Pipelines {
			if pipeline.ID == "" {
				log.Debugf(ctx, "Skipping pipeline %s: no ID (not deployed)", key)
				continue
			}

			// Check if we have a previous snapshot for this resource
			previousPipeline, ok := snapshot.Pipelines[key]
			if !ok {
				log.Debugf(ctx, "Skipping pipeline %s: no previous snapshot", key)
				continue
			}

			// Fetch current remote state
			currentPipeline, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{
				PipelineId: pipeline.ID,
			})
			if err != nil {
				log.Warnf(ctx, "Failed to fetch pipeline %s (ID: %s): %v", key, pipeline.ID, err)
				continue
			}

			// Compare previous and current state
			changelog, err := diff.Diff(previousPipeline, currentPipeline)
			if err != nil {
				log.Warnf(ctx, "Failed to diff pipeline %s: %v", key, err)
				continue
			}

			// Only add to output if there are changes
			if len(changelog) > 0 {
				output.Pipelines[key] = &ResourceDiff{
					Changes: changelog,
				}
				log.Debugf(ctx, "Found %d changes for pipeline %s", len(changelog), key)

				// Save changes back to YAML if save flag is set
				if writer != nil {
					err = writer.WritePipelineDiff(ctx, key, currentPipeline)
					if err != nil {
						log.Warnf(ctx, "Failed to save pipeline %s changes: %v", key, err)
					}
				}
			}
		}

		return writeOutput(cmd, output)
	}

	return cmd
}

func writeOutput(cmd *cobra.Command, output *DiffOutput) error {
	buf, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	_, _ = out.Write(buf)
	_, _ = out.Write([]byte{'\n'})

	return nil
}
