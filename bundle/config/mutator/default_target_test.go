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
	bundle := &bundle.Bundle{}
	err := mutator.DefineDefaultTarget().Apply(context.Background(), bundle)
	require.NoError(t, err)
	env, ok := bundle.Config.Targets["default"]
	assert.True(t, ok)
	assert.Equal(t, &config.Target{}, env)
}

func TestDefaultTargetAlreadySpecified(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"development": {},
			},
		},
	}
	err := mutator.DefineDefaultTarget().Apply(context.Background(), bundle)
	require.NoError(t, err)
	_, ok := bundle.Config.Targets["default"]
	assert.False(t, ok)
}
