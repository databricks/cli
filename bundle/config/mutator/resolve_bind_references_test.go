package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBindReferences(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{Target: "dev"},
			Variables: map[string]*variable.Variable{
				"job_id": {Value: "123"},
			},
		},
		Target: &config.Target{
			Bind: config.Bind{
				"jobs": {
					"foo": {ID: "${var.job_id}"},
					"bar": {ID: "${bundle.target}-456"},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ResolveBindReferences())
	require.NoError(t, diags.Error())
	assert.Equal(t, "123", b.Target.Bind["jobs"]["foo"].ID)
	assert.Equal(t, "dev-456", b.Target.Bind["jobs"]["bar"].ID)
}

func TestResolveBindReferencesIntegerValue(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"job_id": {Value: 123},
			},
		},
		Target: &config.Target{
			Bind: config.Bind{
				"jobs": {"foo": {ID: "${var.job_id}"}},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ResolveBindReferences())
	require.NoError(t, diags.Error())
	assert.Equal(t, "123", b.Target.Bind["jobs"]["foo"].ID)
}

func TestResolveBindReferencesUnknownVariable(t *testing.T) {
	b := &bundle.Bundle{
		Target: &config.Target{
			Bind: config.Bind{
				"jobs": {"foo": {ID: "${var.missing}"}},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ResolveBindReferences())
	assert.Error(t, diags.Error())
}

func TestResolveBindReferencesNoTarget(t *testing.T) {
	b := &bundle.Bundle{}
	diags := bundle.Apply(t.Context(), b, ResolveBindReferences())
	assert.NoError(t, diags.Error())
}
