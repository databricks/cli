package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefineDefaultTarget_AddsDefaultWhenNoneDefined(t *testing.T) {
	u := &ucm.Ucm{}
	diags := ucm.Apply(t.Context(), u, mutator.DefineDefaultTarget())
	require.NoError(t, diags.Error())

	got, ok := u.Config.Targets["default"]
	assert.True(t, ok)
	assert.Equal(t, &config.Target{}, got)
}

func TestDefineDefaultTarget_NoopWhenTargetExists(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Targets: map[string]*config.Target{"dev": {}},
		},
	}
	diags := ucm.Apply(t.Context(), u, mutator.DefineDefaultTarget())
	require.NoError(t, diags.Error())

	_, ok := u.Config.Targets["default"]
	assert.False(t, ok)
}
