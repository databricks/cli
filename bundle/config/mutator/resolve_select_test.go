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

// TestResolveSelect_Normalizes checks that selectors are normalized to their
// qualified "type.name" form in place, without filtering the config. End-to-end
// selection behavior is covered by acceptance/bundle/select.
func TestResolveSelect_Normalizes(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs:      map[string]*resources.Job{"my_job": {}},
				Pipelines: map[string]*resources.Pipeline{"my_pipeline": {}},
			},
		},
	}
	b.Select = []string{"my_job", "pipelines.my_pipeline"}
	diags := bundle.Apply(t.Context(), b, mutator.ResolveSelect())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"jobs.my_job", "pipelines.my_pipeline"}, b.Select)
	// Config is left untouched; the mutator only resolves selectors.
	assert.Len(t, b.Config.Resources.Jobs, 1)
	assert.Len(t, b.Config.Resources.Pipelines, 1)
}

// TestResolveSelect_Ambiguous covers the ambiguous-selector error. This cannot be
// exercised by an acceptance test: UniqueResourceKeys forbids two resources sharing
// a key across types, so an ambiguous selection is unreachable in a loadable bundle.
func TestResolveSelect_Ambiguous(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs:      map[string]*resources.Job{"thing": {}},
				Pipelines: map[string]*resources.Pipeline{"thing": {}},
			},
		},
	}
	b.Select = []string{"thing"}
	diags := bundle.Apply(t.Context(), b, mutator.ResolveSelect())
	require.Error(t, diags.Error())
	assert.ErrorContains(t, diags.Error(), "ambiguous resource: thing")
	assert.ErrorContains(t, diags.Error(), "use a qualified name to disambiguate")
}
