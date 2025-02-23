package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/mock"
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
	b, diags := loadTargetWithDiags(path, env)
	require.NoError(t, diags.Error())
	return b
}

func loadTargetWithDiags(path, env string) (*bundle.Bundle, diag.Diagnostics) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, path)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	diags := bundle.Apply(ctx, b, bundle.Seq(
		phases.LoadNamedTarget(env),
		mutator.RewriteSyncPaths(),
		mutator.SyncDefaultPath(),
		mutator.SyncInferRoot(),
		mutator.MergeJobClusters(),
		mutator.MergeJobParameters(),
		mutator.MergeJobTasks(),
		mutator.MergePipelineClusters(),
		mutator.MergeApps(),
	))
	return b, diags
}

func configureMock(t *testing.T, b *bundle.Bundle) {
	// Configure mock workspace client
	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &config.Config{
		Host: "https://mock.databricks.workspace.com",
	}
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything).Return(&iam.User{
		UserName: "user@domain.com",
	}, nil)
	b.SetWorkpaceClient(m.WorkspaceClient)
}

func initializeTarget(t *testing.T, path, env string) (*bundle.Bundle, diag.Diagnostics) {
	b := load(t, path)
	configureMock(t, b)

	ctx := dbr.MockRuntime(context.Background(), false)
	diags := bundle.Apply(ctx, b, bundle.Seq(
		mutator.SelectTarget(env),
		phases.Initialize(),
	))

	return b, diags
}
