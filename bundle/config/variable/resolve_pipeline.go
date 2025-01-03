package variable

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

type resolvePipeline struct {
	name string
}

func (l resolvePipeline) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Pipelines.GetByName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return entity.PipelineId, nil
}

func (l resolvePipeline) String() string {
	return "pipeline: " + l.name
}
