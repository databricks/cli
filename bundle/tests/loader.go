package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/require"
)

func load(t *testing.T, path string) *bundle.Bundle {
	ctx := logdiag.InitContext(context.Background())
	logdiag.SetCollect(ctx, true)
	b, err := bundle.Load(ctx, path)
	require.NoError(t, err)
	phases.Load(ctx, b)
	diags := logdiag.FlushCollected(ctx)
	require.NoError(t, diags.Error())
	return b
}

func loadTarget(t *testing.T, path, env string) *bundle.Bundle {
	b, diags := loadTargetWithDiags(path, env)
	require.NoError(t, diags.Error())
	return b
}

func loadTargetWithDiags(path, env string) (*bundle.Bundle, diag.Diagnostics) {
	ctx := logdiag.InitContext(context.Background())
	logdiag.SetCollect(ctx, true)
	b, err := bundle.Load(ctx, path)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	phases.LoadNamedTarget(ctx, b, env)
	diags := logdiag.FlushCollected(ctx)

	diags = diags.Extend(bundle.ApplySeq(ctx, b,
		mutator.RewriteSyncPaths(),
		mutator.SyncDefaultPath(),
		mutator.SyncInferRoot(),
		resourcemutator.MergeJobClusters(),
		resourcemutator.MergeJobParameters(),
		resourcemutator.MergeJobTasks(),
		resourcemutator.MergePipelineClusters(),
		resourcemutator.MergeApps(),
	))
	return b, diags
}
