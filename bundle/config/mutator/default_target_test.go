package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTarget(t *testing.T) {
	b := &bundle.Bundle{}
	diags := bundle.Apply(context.Background(), b, mutator.DefineDefaultTarget())
	require.NoError(t, diags.Error())

	env, ok := b.Config.Targets["default"]
	assert.True(t, ok)
	assert.Equal(t, &config.Target{}, env)
}

func TestDefaultTargetAlreadySpecified(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"development": {},
			},
		},
	}
	diags := bundle.Apply(context.Background(), b, mutator.DefineDefaultTarget())
	require.NoError(t, diags.Error())

	_, ok := b.Config.Targets["default"]
	assert.False(t, ok)
}
