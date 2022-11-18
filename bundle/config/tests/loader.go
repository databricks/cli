package config_tests

import (
	"testing"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/require"
)

func load(t *testing.T, path string) *config.Root {
	root, err := config.Load(path)
	require.NoError(t, err)
	err = mutator.Apply(root, mutator.DefaultMutators())
	require.NoError(t, err)
	return root
}

func loadEnvironment(t *testing.T, path, env string) *config.Root {
	root := load(t, path)
	err := mutator.Apply(root, []mutator.Mutator{mutator.SelectEnvironment(env)})
	require.NoError(t, err)
	return root
}
