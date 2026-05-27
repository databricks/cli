package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateBindResourcesNoTarget(t *testing.T) {
	b := &bundle.Bundle{}
	diags := bundle.Apply(t.Context(), b, ValidateBindResources())
	assert.NoError(t, diags.Error())
}

func TestValidateBindResourcesAcceptsTopLevel(t *testing.T) {
	b := &bundle.Bundle{
		Target: &config.Target{
			Bind: config.Bind{
				"jobs": {"foo": {ID: "1"}},
			},
		},
	}
	diags := bundle.Apply(t.Context(), b, ValidateBindResources())
	assert.NoError(t, diags.Error())
}

func TestValidateBindResourcesRejectsChildResources(t *testing.T) {
	b := &bundle.Bundle{
		Target: &config.Target{
			Bind: config.Bind{
				"jobs.permissions": {"foo": {ID: "1"}},
			},
		},
	}
	diags := bundle.Apply(t.Context(), b, ValidateBindResources())
	assert.Error(t, diags.Error())
	assert.Contains(t, diags[0].Summary, "binding jobs.permissions is not allowed")
}
