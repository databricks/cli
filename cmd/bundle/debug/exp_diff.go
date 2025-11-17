package debug

import (
	"encoding/json"
	"strconv"

	"github.com/databricks/cli/bundle/direct/dresources"
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

type FileChange struct {
	Path            string `json:"path"`
	OriginalContent string `json:"originalContent"`
	ModifiedContent string `json:"modifiedContent"`
}

type ChangesSummary struct {
	Jobs      map[string]*ResourceDiff `json:"jobs,omitempty"`
	Pipelines map[string]*ResourceDiff `json:"pipelines,omitempty"`
}

type DiffOutput struct {
	Files   []FileChange    `json:"files"`
	Changes *ChangesSummary `json:"changes"`
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
			Files: []FileChange{},
			Changes: &ChangesSummary{
				Jobs:      make(map[string]*ResourceDiff),
				Pipelines: make(map[string]*ResourceDiff),
			},
		}

		// If no snapshot exists, return empty diff
		if snapshot == nil {
			log.Debugf(ctx, "No previous snapshot found, skipping diff")
			return writeOutput(cmd, output)
		}

		w := b.WorkspaceClient()

		// Create diff writer
		writer := NewDiffWriter(b)
		writer.saveToFile = save

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

			job := dresources.ResourceJob{}
			currentJobComparable := job.RemapState(currentJob)
			previousJobComparable := job.RemapState(previousJob)

			// Compare previous and current state
			changelog, err := diff.Diff(previousJobComparable, currentJobComparable)
			if err != nil {
				log.Warnf(ctx, "Failed to diff job %s: %v", key, err)
				continue
			}

			// Only add to output if there are changes
			if len(changelog) > 0 {
				output.Changes.Jobs[key] = &ResourceDiff{
					Changes: changelog,
				}
				log.Debugf(ctx, "Found %d changes for job %s", len(changelog), key)

				// Generate file change (will save to file if save flag is set)
				fileChange, err := writer.WriteJobDiff(ctx, key, currentJob, changelog)
				if err != nil {
					log.Warnf(ctx, "Failed to process job %s changes: %v", key, err)
				} else if fileChange != nil {
					output.Files = append(output.Files, *fileChange)
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

			pipeline := dresources.ResourcePipeline{}
			currentPipelineComparable := pipeline.RemapState(currentPipeline)
			previousPipelineComparable := pipeline.RemapState(previousPipeline)

			// Compare previous and current state
			changelog, err := diff.Diff(previousPipelineComparable, currentPipelineComparable)
			if err != nil {
				log.Warnf(ctx, "Failed to diff pipeline %s: %v", key, err)
				continue
			}

			// Only add to output if there are changes
			if len(changelog) > 0 {
				output.Changes.Pipelines[key] = &ResourceDiff{
					Changes: changelog,
				}
				log.Debugf(ctx, "Found %d changes for pipeline %s", len(changelog), key)

				// Generate file change (will save to file if save flag is set)
				fileChange, err := writer.WritePipelineDiff(ctx, key, currentPipeline, changelog)
				if err != nil {
					log.Warnf(ctx, "Failed to process pipeline %s changes: %v", key, err)
				} else if fileChange != nil {
					output.Files = append(output.Files, *fileChange)
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
