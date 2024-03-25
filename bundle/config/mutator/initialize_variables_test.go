package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeVariables(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"foo": nil,
				"bar": {
					Description: "This is a description",
				},
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.InitializeVariables())
	require.NoError(t, diags.Error())
	assert.NotNil(t, b.Config.Variables["foo"])
	assert.NotNil(t, b.Config.Variables["bar"])
	assert.Equal(t, "This is a description", b.Config.Variables["bar"].Description)
}

func TestInitializeVariablesWithoutVariables(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Variables: nil,
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.InitializeVariables())
	require.NoError(t, diags.Error())
	assert.Nil(t, b.Config.Variables)
}
