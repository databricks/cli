package pipelines

import (
	"fmt"
	"regexp"

	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

var pipelineIDRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func historyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history [flags] [KEY|PIPELINE_ID]",
		Args:  root.MaximumNArgs(1),
		Short: "Retrieve past runs for a pipeline",
		Long:  `Retrieve past runs for a pipeline identified by KEY, the unique name of the pipeline as defined in its YAML file, or by PIPELINE_ID, the unique identifier of the pipeline.`,
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
		var key, pipelineId string
		var w *databricks.WorkspaceClient
		if len(args) == 1 && pipelineIDRegex.MatchString(args[0]) {
			if err := root.MustWorkspaceClient(cmd, args); err != nil {
				return err
			}
			pipelineId = args[0]
			key = pipelineId
			w = cmdctx.WorkspaceClient(cmd.Context())
		} else {
			b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
				InitIDs: true,
			})
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			key, err = resolvePipelineArgument(ctx, b, args)
			if err != nil {
				return err
			}

			pipelineId, err = resolvePipelineIdFromKey(ctx, b, key)
			if err != nil {
				return err
			}
			w = b.WorkspaceClient(ctx)
		}
		ctx := cmd.Context()

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
