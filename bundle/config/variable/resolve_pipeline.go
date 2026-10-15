package variable

import (
	"context"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/pipelines/v2"
)

type resolvePipeline struct {
	name string
}

func (l resolvePipeline) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	entity, err := findByName(w.Pipelines.ListIter(ctx, pipelines.ListPipelinesRequest{}), l.name, "PipelineStateInfo", func(p *pipelines.PipelineStateInfo) string {
		return utils.Deref(p.Name)
	})
	if err != nil {
		return "", err
	}
	return utils.Deref(entity.PipelineId), nil
}

func (l resolvePipeline) String() string {
	return "pipeline: " + l.name
}
