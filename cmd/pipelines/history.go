package pipelines

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

func historyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history [flags] PIPELINE_ID",
		Short: "Retrieve past runs for a pipeline",
		Long:  `Retrieve past runs for a pipeline identified by PIPELINE_ID, a unique identifier for a pipeline.`,
	}

	var maxResults int
	var startTimeStr string

	historyGroup := cmdgroup.NewFlagGroup("Filter")
	historyGroup.FlagSet().IntVar(&maxResults, "max-results", 100, "Max number of entries in output.")
	historyGroup.FlagSet().StringVar(&startTimeStr, "start-time", "", "Filter updates after this time (format: 2025-01-15T10:30:00Z)")
	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(historyGroup)

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if len(args) == 0 {
			return errors.New("Provide a PIPELINE_ID.")
		}

		if len(args) > 1 {
			return fmt.Errorf("Expected one PIPELINE_ID, got %d.", len(args))
		}
		w := cmdctx.WorkspaceClient(ctx)

		pipelineId := args[0]

		req := pipelines.ListUpdatesRequest{
			PipelineId: pipelineId,
			MaxResults: maxResults,
		}

		response, err := w.Pipelines.ListUpdates(ctx, req)
		if err != nil {
			return err
		}

		var filteredUpdates []pipelines.UpdateInfo
		if startTimeStr != "" {
			startTime, err := time.Parse("2006-01-02T15:04:05Z", startTimeStr)
			if err != nil {
				return fmt.Errorf("invalid start-time format. Expected format: 2025-01-15T10:30:00Z (YYYY-MM-DDTHH:MM:SSZ), got: %s", startTimeStr)
			}
			startTimeMs := startTime.UnixMilli()

			// Binary search for the split point
			idx := sort.Search(len(response.Updates), func(i int) bool {
				// stop when CreationTime <= cutoff (i.e., no longer after)
				return response.Updates[i].CreationTime <= startTimeMs
			})

			// Check if startTimeMs cutoff is present in the response.Updates
			if idx < len(response.Updates) && response.Updates[idx].CreationTime == startTimeMs {
				filteredUpdates = response.Updates[:idx+1]
			} else {
				filteredUpdates = response.Updates[:idx]
			}
		} else {
			filteredUpdates = response.Updates
		}

		return cmdio.RenderWithTemplate(ctx, filteredUpdates, "Updates summary for pipeline "+pipelineId,
			`{{range .}}Update ID: {{.UpdateId}}
   State: {{.State}}
   Cause: {{.Cause}}
   Creation Time: {{.CreationTime | pretty_date_from_millis}}
   Full Refresh: {{.FullRefresh}}
   Validate Only: {{.ValidateOnly}}
{{end}}`)
	}

	return cmd
}
