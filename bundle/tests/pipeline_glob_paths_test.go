package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestExpandPipelineGlobPaths(t *testing.T) {
	b := loadTarget(t, "./pipeline_glob_paths", "default")

	// Configure mock workspace client
	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &config.Config{
		Host: "https://mock.databricks.workspace.com",
	}
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything).Return(&iam.User{
		UserName: "user@domain.com",
	}, nil)
	b.SetWorkpaceClient(m.WorkspaceClient)

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, phases.Initialize())
	require.Empty(t, diags)
	require.Equal(
		t,
		"/Users/user@domain.com/.bundle/pipeline_glob_paths/default/files/dlt/nyc_taxi_loader",
		b.Config.Resources.Pipelines["nyc_taxi_pipeline"].Libraries[0].Notebook.Path,
	)
}

func TestExpandPipelineGlobPathsWithNonExistent(t *testing.T) {
	b := loadTarget(t, "./pipeline_glob_paths", "error")

	// Configure mock workspace client
	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &config.Config{
		Host: "https://mock.databricks.workspace.com",
	}
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything).Return(&iam.User{
		UserName: "user@domain.com",
	}, nil)
	b.SetWorkpaceClient(m.WorkspaceClient)

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, phases.Initialize())
	require.ErrorContains(t, diags.Error(), "notebook ./non-existent not found")
}
