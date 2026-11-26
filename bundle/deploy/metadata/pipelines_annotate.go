package metadata

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

func init() {
	annotations.Register("pipelines", AnnotatePipelines)
}

type annotatePipelines struct{}

// AnnotatePipelines sets the bundle deployment metadata of every pipeline.
func AnnotatePipelines() bundle.Mutator {
	return &annotatePipelines{}
}

func (m *annotatePipelines) Name() string {
	return "metadata.AnnotatePipelines"
}

func (m *annotatePipelines) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, pipeline := range b.Config.Resources.Pipelines {
		pipeline.Deployment = &pipelines.PipelineDeployment{
			Kind:             pipelines.DeploymentKindBundle,
			MetadataFilePath: metadataFilePath(b),
		}
	}

	return nil
}
