package statemgmt

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

func IsPipelineRunning(ctx context.Context, w *databricks.WorkspaceClient, pipelineId string) (bool, error) {
	resp, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{PipelineId: pipelineId})
	if apierr.IsMissing(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch resp.State {
	case pipelines.PipelineStateIdle, pipelines.PipelineStateFailed, pipelines.PipelineStateDeleted:
		return false, nil
	default:
		return true, nil
	}
}
