package pipelines

import (
	"context"
	"fmt"
	"regexp"

	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// LooksLikeUUID reports whether s is a pipeline ID (UUID) rather than a bundle KEY.
func LooksLikeUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

// pipelineDescribeData is the payload rendered by `pipelines describe`: the
// pipeline definition and current state, plus its most recent update (if any).
type pipelineDescribeData struct {
	Key        string                         `json:"key,omitempty"`
	Pipeline   *pipelines.GetPipelineResponse `json:"pipeline"`
	LastUpdate *pipelines.UpdateInfo          `json:"last_update,omitempty"`
}

func describeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe [flags] KEY|PIPELINE_ID",
		Args:  root.MaximumNArgs(1),
		Short: "Show a summary of a pipeline",
		Long: `Show a pipeline's configuration and current state, plus the result of its
most recent update. Identify the pipeline by bundle KEY, or by PIPELINE_ID
(a UUID) to describe any pipeline in the workspace without a bundle.`,
		// Hidden while the command is still limited to basic pipeline info;
		// unhide once richer output (datasets, DAG, lineage) is available.
		Hidden: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// A raw pipeline ID (UUID) is addressed directly, without a bundle, so
		// describe works for pipelines not defined in the current bundle.
		if len(args) == 1 && LooksLikeUUID(args[0]) {
			if err := root.MustWorkspaceClient(cmd, args); err != nil {
				return err
			}
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			return describePipeline(ctx, w, "", args[0])
		}

		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			InitIDs: true,
		})
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		key, err := resolvePipelineArgument(ctx, b, args)
		if err != nil {
			return err
		}

		pipelineId, err := resolvePipelineIdFromKey(ctx, b, key)
		if err != nil {
			return err
		}

		w := b.WorkspaceClient(ctx)
		return describePipeline(ctx, w, key, pipelineId)
	}

	return cmd
}

// describePipeline fetches the pipeline and its most recent update (if any) and
// renders the summary. key is the bundle KEY when addressed that way, or empty
// when addressed by pipeline ID.
func describePipeline(ctx context.Context, w *databricks.WorkspaceClient, key, pipelineId string) error {
	pipeline, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{PipelineId: pipelineId})
	if err != nil {
		return fmt.Errorf("failed to get pipeline %s: %w", pipelineId, err)
	}

	// Enrich with the most recent update, when the pipeline has run before.
	// LatestUpdates is ordered newest-first.
	var lastUpdate *pipelines.UpdateInfo
	if len(pipeline.LatestUpdates) > 0 {
		updateId := pipeline.LatestUpdates[0].UpdateId
		resp, err := w.Pipelines.GetUpdate(ctx, pipelines.GetUpdateRequest{PipelineId: pipelineId, UpdateId: updateId})
		if err != nil {
			return fmt.Errorf("failed to get latest update %s: %w", updateId, err)
		}
		lastUpdate = resp.Update
	}

	data := pipelineDescribeData{
		Key:        key,
		Pipeline:   pipeline,
		LastUpdate: lastUpdate,
	}
	return cmdio.RenderWithTemplate(ctx, data, "", pipelineDescribeTemplate)
}
