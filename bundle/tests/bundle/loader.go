package bundle

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/require"
)

func load(t *testing.T, path string) *bundle.Bundle {
	ctx := context.Background()
	b, err := bundle.Load(ctx, path)
	require.NoError(t, err)
	err = bundle.Apply(ctx, b, bundle.Seq(mutator.DefaultMutators()...))
	require.NoError(t, err)
	return b
}

func loadTarget(t *testing.T, path, env string) *bundle.Bundle {
	ctx := context.Background()
	b, err := bundle.Load(ctx, path)
	require.NoError(t, err)
	err = bundle.Apply(ctx, b, bundle.Seq(mutator.DefaultMutatorsForTarget(env)...))
	require.NoError(t, err)
	err = bundle.Apply(ctx, b, bundle.Seq(
		mutator.RewriteSyncPaths(),
		mutator.MergeJobClusters(),
		mutator.MergeJobTasks(),
		mutator.MergePipelineClusters(),
	))
	require.NoError(t, err)
	return b
}
