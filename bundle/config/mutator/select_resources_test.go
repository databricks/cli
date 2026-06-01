package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bundleWithJobsAndPipelines() *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs:      map[string]*resources.Job{"my_job": {}},
				Pipelines: map[string]*resources.Pipeline{"my_pipeline": {}},
			},
		},
	}
}

func TestSelectResources_NoOp(t *testing.T) {
	b := bundleWithJobsAndPipelines()
	diags := bundle.Apply(t.Context(), b, mutator.SelectResources())
	require.NoError(t, diags.Error())
	assert.Len(t, b.Config.Resources.Jobs, 1)
	assert.Len(t, b.Config.Resources.Pipelines, 1)
}

func TestSelectResources_UnqualifiedUnique(t *testing.T) {
	b := bundleWithJobsAndPipelines()
	b.Select = []string{"my_job"}
	diags := bundle.Apply(t.Context(), b, mutator.SelectResources())
	require.NoError(t, diags.Error())
	assert.Len(t, b.Config.Resources.Jobs, 1)
	assert.Empty(t, b.Config.Resources.Pipelines)
}

func TestSelectResources_QualifiedName(t *testing.T) {
	b := bundleWithJobsAndPipelines()
	b.Select = []string{"pipelines.my_pipeline"}
	diags := bundle.Apply(t.Context(), b, mutator.SelectResources())
	require.NoError(t, diags.Error())
	assert.Empty(t, b.Config.Resources.Jobs)
	assert.Len(t, b.Config.Resources.Pipelines, 1)
}

func TestSelectResources_NotFound(t *testing.T) {
	b := bundleWithJobsAndPipelines()
	b.Select = []string{"nonexistent"}
	diags := bundle.Apply(t.Context(), b, mutator.SelectResources())
	require.Error(t, diags.Error())
	assert.ErrorContains(t, diags.Error(), "no such resource: nonexistent")
}

func TestSelectResources_QualifiedNotFound(t *testing.T) {
	b := bundleWithJobsAndPipelines()
	b.Select = []string{"jobs.nonexistent"}
	diags := bundle.Apply(t.Context(), b, mutator.SelectResources())
	require.Error(t, diags.Error())
	assert.ErrorContains(t, diags.Error(), "no such resource: jobs.nonexistent")
}

func TestSelectResources_Ambiguous(t *testing.T) {
	b := bundleWithJobsAndPipelines()
	b.Config.Resources.Pipelines["my_job"] = &resources.Pipeline{}
	b.Select = []string{"my_job"}
	diags := bundle.Apply(t.Context(), b, mutator.SelectResources())
	require.Error(t, diags.Error())
	assert.ErrorContains(t, diags.Error(), "ambiguous resource: my_job")
	assert.ErrorContains(t, diags.Error(), "use a qualified name to disambiguate")
}
