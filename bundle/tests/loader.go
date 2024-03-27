package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/phases"
	"github.com/stretchr/testify/require"
)

func load(t *testing.T, path string) *bundle.Bundle {
	ctx := context.Background()
	b, err := bundle.Load(ctx, path)
	require.NoError(t, err)
	diags := bundle.Apply(ctx, b, phases.Load())
	require.NoError(t, diags.Error())
	return b
}

func loadTarget(t *testing.T, path, env string) *bundle.Bundle {
	ctx := context.Background()
	b, err := bundle.Load(ctx, path)
	require.NoError(t, err)
	diags := bundle.Apply(ctx, b, bundle.Seq(
		phases.LoadNamedTarget(env),
		mutator.RewriteSyncPaths(),
		mutator.MergeJobClusters(),
		mutator.MergeJobTasks(),
		mutator.MergePipelineClusters(),
	))
	require.NoError(t, diags.Error())
	return b
}
