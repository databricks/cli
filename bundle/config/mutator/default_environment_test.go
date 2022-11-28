package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultEnvironment(t *testing.T) {
	bundle := &bundle.Bundle{}
	_, err := mutator.DefineDefaultEnvironment().Apply(context.Background(), bundle)
	require.NoError(t, err)
	env, ok := bundle.Config.Environments["default"]
	assert.True(t, ok)
	assert.Equal(t, &config.Environment{}, env)
}

func TestDefaultEnvironmentAlreadySpecified(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Environments: map[string]*config.Environment{
				"development": {},
			},
		},
	}
	_, err := mutator.DefineDefaultEnvironment().Apply(context.Background(), bundle)
	require.NoError(t, err)
	_, ok := bundle.Config.Environments["default"]
	assert.False(t, ok)
}
