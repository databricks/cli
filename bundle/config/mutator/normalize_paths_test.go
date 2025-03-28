package mutator

import (
	"context"
	pathlib "path"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestNormalizePaths(t *testing.T) {
	m := NormalizePaths()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: &jobs.JobSettings{
						Tasks: []jobs.Task{
							{
								NotebookTask: &jobs.NotebookTask{
									NotebookPath: "src/notebook.py",
								},
							},
						},
					}},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	b.BundleRootPath = tmpDir
	notebookPathPath := dyn.MustPathFromString("resources.jobs.job1.tasks[0].notebook_task.notebook_path")

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPath(v, notebookPathPath, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
			location := dyn.Location{File: pathlib.Join(tmpDir, "job1.yml")}
			return value.WithLocations([]dyn.Location{location}), nil
		})
	})
	require.NoError(t, err)

	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	err = b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		notebookPath, err := dyn.GetByPath(v, notebookPathPath)

		require.NoError(t, err)
		require.True(t, notebookPath.DerivesDirectory())

		return v, nil
	})
	require.NoError(t, err)
}
