package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/require"
)

func load(t *testing.T, path string) *bundle.Bundle {
	b, err := bundle.Load(path)
	require.NoError(t, err)
	err = bundle.Apply(context.Background(), b, bundle.Seq(mutator.DefaultMutators()...))
	require.NoError(t, err)
	return b
}

func loadEnvironment(t *testing.T, path, env string) *bundle.Bundle {
	b := load(t, path)
	err := bundle.Apply(context.Background(), b, mutator.SelectEnvironment(env))
	require.NoError(t, err)
	return b
}

func loadEnvironmentWithError(t *testing.T, path, env string) error {
	b := load(t, path)
	err := bundle.Apply(context.Background(), b, mutator.SelectEnvironment(env))
	require.Error(t, err)
	return err
}
