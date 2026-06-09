package trampoline

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

type functions struct{}

func (f *functions) GetTasks(b *bundle.Bundle) []TaskWithJobKey {
	var tasks []TaskWithJobKey
	for k, job := range b.Config.Resources.Jobs {
		for i := range job.Tasks {
			tasks = append(tasks, TaskWithJobKey{
				JobKey: k,
				Task:   &job.Tasks[i],
			})
		}
	}

	return tasks
}

func (f *functions) GetTemplateData(task *jobs.Task) (map[string]any, error) {
	if task.PythonWheelTask == nil {
		return nil, errors.New("PythonWheelTask cannot be nil")
	}

	data := make(map[string]any)
	data["MyName"] = "Trampoline"
	return data, nil
}

func (f *functions) CleanUp(task *jobs.Task) error {
	task.PythonWheelTask = nil
	return nil
}

func TestGenerateTrampoline(t *testing.T) {
	tmpDir := t.TempDir()

	tasks := []jobs.Task{
		{
			TaskKey: "to_trampoline",
			PythonWheelTask: &jobs.PythonWheelTask{
				PackageName: "test",
				EntryPoint:  "run",
			},
		},
	}

	b := &bundle.Bundle{
		BundleRootPath: filepath.Join(tmpDir, "parent", "my_bundle"),
		SyncRootPath:   filepath.Join(tmpDir, "parent"),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/Workspace/files",
			},
			Bundle: config.Bundle{
				Target: "development",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test": {
						JobSettings: jobs.JobSettings{
							Tasks: tasks,
						},
					},
				},
			},
		},
	}
	ctx := t.Context()

	funcs := functions{}
	trampoline := NewTrampoline("test_trampoline", &funcs, "Hello from {{.MyName}}")
	diags := bundle.Apply(ctx, b, trampoline)
	require.NoError(t, diags.Error())

	dir, err := b.InternalDir(ctx)
	require.NoError(t, err)
	filename := filepath.Join(dir, "notebook_4_test_to_trampoline.py")

	bytes, err := os.ReadFile(filename)
	require.NoError(t, err)

	require.Equal(t, "Hello from Trampoline", string(bytes))

	task := b.Config.Resources.Jobs["test"].Tasks[0]
	require.Equal(t, "/Workspace/files/my_bundle/.databricks/bundle/development/.internal/notebook_4_test_to_trampoline", task.NotebookTask.NotebookPath)
	require.Nil(t, task.PythonWheelTask)
}

func TestGenerateTrampolineWithCollidingKeys(t *testing.T) {
	tmpDir := t.TempDir()

	newJob := func(taskKey string) *resources.Job {
		return &resources.Job{
			JobSettings: jobs.JobSettings{
				Tasks: []jobs.Task{
					{
						TaskKey: taskKey,
						PythonWheelTask: &jobs.PythonWheelTask{
							PackageName: "test",
							EntryPoint:  "run",
						},
					},
				},
			},
		}
	}

	b := &bundle.Bundle{
		BundleRootPath: filepath.Join(tmpDir, "parent", "my_bundle"),
		SyncRootPath:   filepath.Join(tmpDir, "parent"),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/Workspace/files",
			},
			Bundle: config.Bundle{
				Target: "development",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"a_b": newJob("c"),
					"a":   newJob("b_c"),
				},
			},
		},
	}
	ctx := t.Context()

	funcs := functions{}
	trampoline := NewTrampoline("test_trampoline", &funcs, "Hello from {{.MyName}}")
	diags := bundle.Apply(ctx, b, trampoline)
	require.NoError(t, diags.Error())

	dir, err := b.InternalDir(ctx)
	require.NoError(t, err)

	for _, name := range []string{"notebook_3_a_b_c", "notebook_1_a_b_c"} {
		_, err := os.Stat(filepath.Join(dir, name+".py"))
		require.NoError(t, err)
	}

	require.Equal(t,
		"/Workspace/files/my_bundle/.databricks/bundle/development/.internal/notebook_3_a_b_c",
		b.Config.Resources.Jobs["a_b"].Tasks[0].NotebookTask.NotebookPath)
	require.Equal(t,
		"/Workspace/files/my_bundle/.databricks/bundle/development/.internal/notebook_1_a_b_c",
		b.Config.Resources.Jobs["a"].Tasks[0].NotebookTask.NotebookPath)
}
