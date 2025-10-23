package pipelines

import (
	"fmt"

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

	var startTimeStr string
	var endTimeStr string

	type pipelineHistoryData struct {
		Key     string
		Updates []pipelines.UpdateInfo
	}

	historyGroup := cmdgroup.NewFlagGroup("Filter")
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
		ctx = cmd.Context()

		phases.Initialize(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		// Load the deployment state to get pipeline IDs from resource
		ctx = statemgmt.PullResourcesState(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		bundle.ApplySeqContext(ctx, b,
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

		startTimePtr, err := parseTimeToUnixMillis(startTimeStr)
		if err != nil {
			return err
		}

		endTimePtr, err := parseTimeToUnixMillis(endTimeStr)
		if err != nil {
			return err
		}

		updates, err := fetchPipelineUpdates(ctx, w, startTimePtr, endTimePtr, pipelineId)
		if err != nil {
			return fmt.Errorf("failed to fetch pipeline updates: %w", err)
		}

		data := pipelineHistoryData{
			Key:     key,
			Updates: updates,
		}
		return cmdio.RenderWithTemplate(ctx, data, "", pipelineHistoryTemplate)
	}

	return cmd
}
