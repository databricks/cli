package metadata

import (
	"testing"

	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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
						CreatePipelineRequest: pipelines.CreatePipelineRequest{
							Name: new("My Pipeline One"),
						},
					},
					"my-pipeline-2": {
						CreatePipelineRequest: pipelines.CreatePipelineRequest{
							Name: new("My Pipeline Two"),
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, AnnotatePipelines())
	require.NoError(t, diags.Error())

	assert.Equal(t,
		&pipelines.PipelineDeployment{
			Kind:             pipelines.DeploymentKind_Bundle,
			MetadataFilePath: new("/a/b/c/metadata.json"),
		},
		b.Config.Resources.Pipelines["my-pipeline-1"].Deployment)

	assert.Equal(t,
		&pipelines.PipelineDeployment{
			Kind:             pipelines.DeploymentKind_Bundle,
			MetadataFilePath: new("/a/b/c/metadata.json"),
		},
		b.Config.Resources.Pipelines["my-pipeline-2"].Deployment)
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

	diags := bundle.Apply(t.Context(), b, AnnotatePipelines())
	require.NoError(t, diags.Error())
}
