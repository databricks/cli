package python

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestNoTransformByDefault(t *testing.T) {
	tmpDir := t.TempDir()

	b := &bundle.Bundle{
		Config: config.Root{
			Path: tmpDir,
			Bundle: config.Bundle{
				Target: "development",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey: "key1",
									PythonWheelTask: &jobs.PythonWheelTask{
										PackageName: "test_package",
										EntryPoint:  "main",
									},
									Libraries: []compute.Library{
										{Whl: "/Workspace/Users/test@test.com/bundle/dist/test.whl"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	trampoline := TransformWheelTask()
	err := bundle.Apply(context.Background(), b, trampoline)
	require.NoError(t, err)

	task := b.Config.Resources.Jobs["job1"].Tasks[0]
	require.NotNil(t, task.PythonWheelTask)
	require.Equal(t, "test_package", task.PythonWheelTask.PackageName)
	require.Equal(t, "main", task.PythonWheelTask.EntryPoint)
	require.Equal(t, "/Workspace/Users/test@test.com/bundle/dist/test.whl", task.Libraries[0].Whl)

	require.Nil(t, task.NotebookTask)
}

func TestTransformWithExperimentalSettingSetToTrue(t *testing.T) {
	tmpDir := t.TempDir()

	b := &bundle.Bundle{
		Config: config.Root{
			Path: tmpDir,
			Bundle: config.Bundle{
				Target: "development",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey: "key1",
									PythonWheelTask: &jobs.PythonWheelTask{
										PackageName: "test_package",
										EntryPoint:  "main",
									},
									Libraries: []compute.Library{
										{Whl: "/Workspace/Users/test@test.com/bundle/dist/test.whl"},
									},
								},
							},
						},
					},
				},
			},
			Experimental: &config.Experimental{
				PythonWheelWrapper: true,
			},
		},
	}

	trampoline := TransformWheelTask()
	err := bundle.Apply(context.Background(), b, trampoline)
	require.NoError(t, err)

	task := b.Config.Resources.Jobs["job1"].Tasks[0]
	require.Nil(t, task.PythonWheelTask)
	require.NotNil(t, task.NotebookTask)

	dir, err := b.InternalDir(context.Background())
	require.NoError(t, err)

	internalDirRel, err := filepath.Rel(b.Config.Path, dir)
	require.NoError(t, err)

	require.Equal(t, filepath.Join(internalDirRel, "notebook_job1_key1"), task.NotebookTask.NotebookPath)

	require.Empty(t, task.Libraries)
}
