package mutator

import (
	"context"
	"fmt"
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
	tasks := make([]TaskWithJobKey, 0)
	for k := range b.Config.Resources.Jobs["test"].Tasks {
		tasks = append(tasks, TaskWithJobKey{
			JobKey: "test",
			Task:   &b.Config.Resources.Jobs["test"].Tasks[k],
		})
	}

	return tasks
}

func (f *functions) GetTemplateData(task *jobs.Task) (map[string]any, error) {
	if task.PythonWheelTask == nil {
		return nil, fmt.Errorf("PythonWheelTask cannot be nil")
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
			}},
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: tmpDir,
			Bundle: config.Bundle{
				Target: "development",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test": {
						Paths: resources.Paths{
							ConfigFilePath: tmpDir,
						},
						JobSettings: &jobs.JobSettings{
							Tasks: tasks,
						},
					},
				},
			},
		},
	}
	ctx := context.Background()

	funcs := functions{}
	trampoline := NewTrampoline("test_trampoline", &funcs, "Hello from {{.MyName}}")
	err := bundle.Apply(ctx, b, trampoline)
	require.NoError(t, err)

	dir, err := b.InternalDir()
	require.NoError(t, err)
	filename := filepath.Join(dir, "notebook_test_to_trampoline.py")

	bytes, err := os.ReadFile(filename)
	require.NoError(t, err)

	require.Equal(t, "Hello from Trampoline", string(bytes))

	task := b.Config.Resources.Jobs["test"].Tasks[0]
	require.Equal(t, task.NotebookTask.NotebookPath, ".databricks/bundle/development/.internal/notebook_test_to_trampoline")
	require.Nil(t, task.PythonWheelTask)
}
