package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

type lookupPipeline struct {
	name string
}

func (l *lookupPipeline) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Pipelines.GetByName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(entity.PipelineId), nil
}

func (l *lookupPipeline) String() string {
	return fmt.Sprintf("pipeline: %s", l.name)
}
