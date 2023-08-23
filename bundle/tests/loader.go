package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/require"
)

func loadBundle(t *testing.T, path string, mutators []bundle.Mutator) *bundle.Bundle {
	ctx := context.Background()
	b, err := bundle.Load(ctx, path)
	require.NoError(t, err)
	err = bundle.Apply(ctx, b, bundle.Seq(mutators...))
	require.NoError(t, err)
	return b
}

func load(t *testing.T, path string) *bundle.Bundle {
	return loadBundle(t, path, mutator.DefaultMutators())
}

func loadTarget(t *testing.T, path, env string) *bundle.Bundle {
	return loadBundle(t, path, mutator.DefaultMutatorsForTarget(env))
}
