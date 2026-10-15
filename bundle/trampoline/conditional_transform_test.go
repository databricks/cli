package trampoline

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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
							Tasks: []jobs.TaskSettings{
								{
									TaskKey: new("key1"),
									PythonWheelTask: &jobs.PythonWheelTask{
										PackageName: new("test_package"),
										EntryPoint:  new("main"),
									},
									Libraries: []jobs.Library{
										{Whl: new("/Workspace/Users/test@test.com/bundle/dist/test.whl")},
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
	diags := bundle.Apply(t.Context(), b, trampoline)
	require.NoError(t, diags.Error())

	task := b.Config.Resources.Jobs["job1"].Tasks[0]
	require.NotNil(t, task.PythonWheelTask)
	require.Equal(t, "test_package", utils.Deref(task.PythonWheelTask.PackageName))
	require.Equal(t, "main", utils.Deref(task.PythonWheelTask.EntryPoint))
	require.Equal(t, "/Workspace/Users/test@test.com/bundle/dist/test.whl", utils.Deref(task.Libraries[0].Whl))

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
							Tasks: []jobs.TaskSettings{
								{
									TaskKey: new("key1"),
									PythonWheelTask: &jobs.PythonWheelTask{
										PackageName: new("test_package"),
										EntryPoint:  new("main"),
									},
									Libraries: []jobs.Library{
										{Whl: new("/Workspace/Users/test@test.com/bundle/dist/test.whl")},
										{Jar: new("/Workspace/Users/test@test.com/bundle/dist/test.jar")},
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
	diags := bundle.Apply(t.Context(), b, trampoline)
	require.NoError(t, diags.Error())

	task := b.Config.Resources.Jobs["job1"].Tasks[0]
	require.Nil(t, task.PythonWheelTask)
	require.NotNil(t, task.NotebookTask)
	require.Equal(t, "/Workspace/files/my_bundle/.databricks/bundle/development/.internal/notebook_4_job1_key1", utils.Deref(task.NotebookTask.NotebookPath))

	require.Len(t, task.Libraries, 1)
	require.Equal(t, "/Workspace/Users/test@test.com/bundle/dist/test.jar", utils.Deref(task.Libraries[0].Jar))
}
