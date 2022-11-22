package mutator_test

import (
	"testing"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultEnvironment(t *testing.T) {
	root := &config.Root{}
	_, err := mutator.DefineDefaultEnvironment().Apply(root)
	require.NoError(t, err)
	env, ok := root.Environments["default"]
	assert.True(t, ok)
	assert.Equal(t, &config.Environment{}, env)
}

func TestDefaultEnvironmentAlreadySpecified(t *testing.T) {
	root := &config.Root{
		Environments: map[string]*config.Environment{
			"development": {},
		},
	}
	_, err := mutator.DefineDefaultEnvironment().Apply(root)
	require.NoError(t, err)
	_, ok := root.Environments["default"]
	assert.False(t, ok)
}
