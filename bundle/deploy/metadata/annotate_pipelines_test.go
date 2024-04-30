package metadata

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotatePipelinesMutator(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				StatePath: "/a/b/c",
			},
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline-1": {
						PipelineSpec: &pipelines.PipelineSpec{
							Name: "My Pipeline One",
						},
					},
					"my-pipeline-2": {
						PipelineSpec: &pipelines.PipelineSpec{
							Name: "My Pipeline Two",
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, AnnotatePipelines())
	require.NoError(t, diags.Error())

	assert.Equal(t,
		&pipelines.PipelineDeployment{
			Kind:             pipelines.DeploymentKindBundle,
			MetadataFilePath: "/a/b/c/metadata.json",
		},
		b.Config.Resources.Pipelines["my-pipeline-1"].PipelineSpec.Deployment)

	assert.Equal(t,
		&pipelines.PipelineDeployment{
			Kind:             pipelines.DeploymentKindBundle,
			MetadataFilePath: "/a/b/c/metadata.json",
		},
		b.Config.Resources.Pipelines["my-pipeline-2"].PipelineSpec.Deployment)
}

func TestAnnotatePipelinesMutatorPipelineWithoutASpec(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				StatePath: "/a/b/c",
			},
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline-1": {},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, AnnotatePipelines())
	require.NoError(t, diags.Error())
}
