package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	libsenv "github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineTestingEnabled(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"orders":  {Testing: &resources.PipelineTesting{}},
					"reports": {},
				},
			},
		},
	}
	loc := diag.Location{File: "databricks.yml", Line: 9, Column: 9}
	bundletest.SetLocation(b, "resources.pipelines.orders.testing", []diag.Location{loc})

	diags := PipelineTestingEnabled().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, env.ExperimentalPipelineTestingVariable)
	assert.Equal(t, []diag.Location{loc}, diags[0].Locations)
	require.Len(t, diags[0].Paths, 1)
	assert.Equal(t, "resources.pipelines.orders.testing", diags[0].Paths[0].String())

	ctx := libsenv.Set(t.Context(), env.ExperimentalPipelineTestingVariable, "true")
	assert.Empty(t, PipelineTestingEnabled().Apply(ctx, b))
}
