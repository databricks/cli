package pipelines

import (
	"fmt"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

func historyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history [flags] [KEY]",
		Args:  root.MaximumNArgs(1),
		Short: "Retrieve past runs for a pipeline",
		Long:  `Retrieve past runs for a pipeline identified by KEY, the unique name of the pipeline as defined in its YAML file.`,
	}

	var number int
	var startTimeStr string
	var endTimeStr string

	type pipelineHistoryData struct {
		Key     string
		Updates []pipelines.UpdateInfo
	}

	historyGroup := cmdgroup.NewFlagGroup("Filter")
	historyGroup.FlagSet().IntVarP(&number, "number", "n", 100, "Number of entries in output.")
	historyGroup.FlagSet().StringVar(&startTimeStr, "start-time", "", "Filter updates after this time (format: 2025-01-15T10:30:00Z)")
	historyGroup.FlagSet().StringVar(&endTimeStr, "end-time", "", "Filter updates before this time (format: 2025-01-15T10:30:00Z)")
	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(historyGroup)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := utils.ConfigureBundleWithVariables(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		phases.Initialize(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		// Load the deployment state to get pipeline IDs from resource
		bundle.ApplySeqContext(ctx, b,
			statemgmt.StatePull(),
			statemgmt.Load(),
		)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		key, err := resolvePipelineArgument(ctx, b, args)
		if err != nil {
			return err
		}

		pipelineId, err := resolvePipelineIdFromKey(ctx, b, key)
		if err != nil {
			return err
		}

		w := b.WorkspaceClient()

		req := pipelines.ListUpdatesRequest{
			PipelineId: pipelineId,
		}

		// Only set MaxResults if the flag was provided, avoiding setting to the default value.
		if cmd.Flags().Changed("number") {
			req.MaxResults = number
		}

		response, err := w.Pipelines.ListUpdates(ctx, req)
		if err != nil {
			return err
		}

		filteredUpdates := response.Updates
		if startTimeStr != "" {
			startTime, err := time.Parse(time.RFC3339Nano, startTimeStr)
			if err != nil {
				return fmt.Errorf("invalid start-time format. Expected format: 2025-01-15T10:30:00Z (YYYY-MM-DDTHH:MM:SSZ), got: %s", startTimeStr)
			}
			startTimeMs := startTime.UnixMilli()
			filteredUpdates = updatesAfter(filteredUpdates, startTimeMs)
		}

		if endTimeStr != "" {
			endTime, err := time.Parse(time.RFC3339Nano, endTimeStr)
			if err != nil {
				return fmt.Errorf("invalid end-time format. Expected format: 2025-01-15T10:30:00Z (YYYY-MM-DDTHH:MM:SSZ), got: %s", endTimeStr)
			}
			endTimeMs := endTime.UnixMilli()
			filteredUpdates = updatesBefore(filteredUpdates, endTimeMs)
		}

		data := pipelineHistoryData{
			Key:     key,
			Updates: filteredUpdates,
		}
		return cmdio.RenderWithTemplate(ctx, data, "", pipelineHistoryTemplate)
	}

	return cmd
}
