package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestValidateBindResourcesNoTarget(t *testing.T) {
	b := &bundle.Bundle{}
	diags := bundle.Apply(t.Context(), b, ValidateBindResources())
	assert.NoError(t, diags.Error())
}

func TestValidateBindResourcesAcceptsDefinedResource(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{"foo": {}},
			},
		},
		Target: &config.Target{
			Bind: config.Bind{
				"jobs": {"foo": {ID: "1"}},
			},
		},
	}
	diags := bundle.Apply(t.Context(), b, ValidateBindResources())
	assert.NoError(t, diags.Error())
}

func TestValidateBindResourcesRejectsUnsupportedType(t *testing.T) {
	b := &bundle.Bundle{
		Target: &config.Target{
			Bind: config.Bind{
				"jobs.permissions": {"foo": {ID: "1"}},
			},
		},
	}
	diags := bundle.Apply(t.Context(), b, ValidateBindResources())
	assert.Error(t, diags.Error())
	assert.Contains(t, diags[0].Summary, `binding "jobs.permissions" is not supported`)
}

func TestValidateBindResourcesRejectsUndefinedResource(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{Target: "dev"},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{"foo": {}},
			},
		},
		Target: &config.Target{
			Bind: config.Bind{
				"jobs": {"bar": {ID: "1"}},
			},
		},
	}
	diags := bundle.Apply(t.Context(), b, ValidateBindResources())
	assert.Error(t, diags.Error())
	assert.Equal(t, `bind block references undefined resource "resources.jobs.bar"; define it in the resources section or remove the bind block`, diags[0].Summary)
	assert.Equal(t, "targets.dev.bind.jobs.bar", diags[0].Paths[0].String())
}
