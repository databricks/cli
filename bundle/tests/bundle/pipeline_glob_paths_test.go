package bundle

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

func TestExpandPipelineGlobPathsWithNonExistent(t *testing.T) {
	ctx := context.Background()
	ctx = root.SetWorkspaceClient(ctx, nil)

	b, err := bundle.Load(ctx, "./pipeline_glob_paths")
	require.NoError(t, err)

	err = bundle.Apply(ctx, b, bundle.Seq(mutator.DefaultMutators()...))
	require.NoError(t, err)
	b.Config.Bundle.Target = "default"

	b.Config.Workspace.CurrentUser = &config.User{User: &iam.User{UserName: "user@domain.com"}}
	b.WorkspaceClient()

	m := phases.Initialize()
	err = bundle.Apply(ctx, b, m)
	require.Error(t, err)
	require.ErrorContains(t, err, "notebook ./non-existent not found")

	require.Equal(
		t,
		b.Config.Resources.Pipelines["nyc_taxi_pipeline"].Libraries[0].Notebook.Path,
		"/Users/user@domain.com/.bundle/pipeline_glob_paths/default/files/dlt/nyc_taxi_loader",
	)
}
