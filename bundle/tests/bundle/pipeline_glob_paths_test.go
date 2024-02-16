package bundle

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestExpandPipelineGlobPathsWithNonExistent(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./pipeline_glob_paths")
	require.NoError(t, err)

	err = bundle.Apply(ctx, b, bundle.Seq(mutator.DefaultMutatorsForTarget("default")...))
	require.NoError(t, err)

	// Configure mock workspace client
	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &config.Config{
		Host: "https://mock.databricks.workspace.com",
	}
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything).Return(&iam.User{
		UserName: "user@domain.com",
	}, nil)
	b.SetWorkpaceClient(m.WorkspaceClient)

	err = bundle.Apply(ctx, b, phases.Initialize())
	require.Error(t, err)
	require.ErrorContains(t, err, "notebook ./non-existent not found")

	require.Equal(
		t,
		b.Config.Resources.Pipelines["nyc_taxi_pipeline"].Libraries[0].Notebook.Path,
		"/Users/user@domain.com/.bundle/pipeline_glob_paths/default/files/dlt/nyc_taxi_loader",
	)
}
