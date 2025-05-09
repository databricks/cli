package trampoline

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
		BundleRootPath: filepath.Join(tmpDir, "parent", "my_bundle"),
		SyncRootPath:   filepath.Join(tmpDir, "parent"),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "development",
			},
			Workspace: config.Workspace{
				FilePath: "/Workspace/files",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
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
	diags := bundle.Apply(context.Background(), b, trampoline)
	require.NoError(t, diags.Error())

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
		BundleRootPath: filepath.Join(tmpDir, "parent", "my_bundle"),
		SyncRootPath:   filepath.Join(tmpDir, "parent"),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "development",
			},
			Workspace: config.Workspace{
				FilePath: "/Workspace/files",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey: "key1",
									PythonWheelTask: &jobs.PythonWheelTask{
										PackageName: "test_package",
										EntryPoint:  "main",
									},
									Libraries: []compute.Library{
										{Whl: "/Workspace/Users/test@test.com/bundle/dist/test.whl"},
										{Jar: "/Workspace/Users/test@test.com/bundle/dist/test.jar"},
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
	diags := bundle.Apply(context.Background(), b, trampoline)
	require.NoError(t, diags.Error())

	task := b.Config.Resources.Jobs["job1"].Tasks[0]
	require.Nil(t, task.PythonWheelTask)
	require.NotNil(t, task.NotebookTask)
	require.Equal(t, "/Workspace/files/my_bundle/.databricks/bundle/development/.internal/notebook_job1_key1", task.NotebookTask.NotebookPath)

	require.Len(t, task.Libraries, 1)
	require.Equal(t, "/Workspace/Users/test@test.com/bundle/dist/test.jar", task.Libraries[0].Jar)
}
