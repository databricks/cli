package config_tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandPipelineGlobPaths(t *testing.T) {
	b, diags := initializeTarget(t, "./pipeline_glob_paths", "default")
	require.NoError(t, diags.Error())
	require.Equal(
		t,
		"/Workspace/Users/user@domain.com/.bundle/pipeline_glob_paths/default/files/dlt/nyc_taxi_loader",
		b.Config.Resources.Pipelines["nyc_taxi_pipeline"].Libraries[0].Notebook.Path,
	)
}

func TestExpandPipelineGlobPathsWithNonExistent(t *testing.T) {
	_, diags := initializeTarget(t, "./pipeline_glob_paths", "error")
	require.ErrorContains(t, diags.Error(), "notebook ./non-existent not found")
}
