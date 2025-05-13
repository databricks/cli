package metadata

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type annotatePipelines struct{}

func AnnotatePipelines() bundle.Mutator {
	return &annotatePipelines{}
}

func (m *annotatePipelines) Name() string {
	return "metadata.AnnotatePipelines"
}

func (m *annotatePipelines) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, pipeline := range b.Config.Resources.Pipelines {
		pipeline.CreatePipeline.Deployment = &pipelines.PipelineDeployment{
			Kind:             pipelines.DeploymentKindBundle,
			MetadataFilePath: metadataFilePath(b),
		}
	}

	return nil
}
