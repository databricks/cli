package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func pipelineWithCascade(cascade *bool) *resources.Pipeline {
	return &resources.Pipeline{
		CreatePipeline:   pipelines.CreatePipeline{Name: "my_pipeline"},
		CascadeOnDestroy: cascade,
	}
}

func TestValidateCascadeOnDestroyDirectEngineReturnsNil(t *testing.T) {
	no := false
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"my_pipeline": pipelineWithCascade(&no),
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ValidateCascadeOnDestroy(engine.EngineDirect))
	assert.Empty(t, diags)
}

func TestValidateCascadeOnDestroyTerraformEngineUnsetReturnsNil(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"my_pipeline": pipelineWithCascade(nil),
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ValidateCascadeOnDestroy(engine.EngineTerraform))
	assert.Empty(t, diags)
}

func TestValidateCascadeOnDestroyTerraformEngineSetEmitsError(t *testing.T) {
	no := false
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"my_pipeline": pipelineWithCascade(&no),
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ValidateCascadeOnDestroy(engine.EngineTerraform))
	assert.Len(t, diags, 1)
	assert.Equal(t, "cascade_on_destroy is only supported in direct deployment mode", diags[0].Summary)
	assert.Contains(t, diags[0].Detail, "https://docs.databricks.com/dev-tools/bundles/direct")
}
