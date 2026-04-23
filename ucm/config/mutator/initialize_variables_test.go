package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/config/variable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeVariables_FillsNilEntries(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Variables: map[string]*variable.Variable{
				"foo": nil,
				"bar": {Description: "bar doc"},
			},
		},
	}
	diags := ucm.Apply(t.Context(), u, mutator.InitializeVariables())
	require.NoError(t, diags.Error())
	assert.NotNil(t, u.Config.Variables["foo"])
	assert.NotNil(t, u.Config.Variables["bar"])
	assert.Equal(t, "bar doc", u.Config.Variables["bar"].Description)
}

func TestInitializeVariables_NoVariables(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	diags := ucm.Apply(t.Context(), u, mutator.InitializeVariables())
	require.NoError(t, diags.Error())
	assert.Nil(t, u.Config.Variables)
}
