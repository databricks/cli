package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

func TestValidateProductionPipelines(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: &pipelines.CreatePipeline{
							Development: true,
						},
					},
				},
			},
		},
	}

	diags := validateProductionPipelines(b, false)

	require.EqualError(t, diags.Error(), "target with 'mode: production' cannot include a pipeline with 'development: true'")
}
