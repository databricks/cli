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

func getTasks(b *bundle.Bundle) []*jobs.Task {
	tasks := make([]*jobs.Task, 0)
	for k := range b.Config.Resources.Jobs["test"].Tasks {
		tasks = append(tasks, &b.Config.Resources.Jobs["test"].Tasks[k])
	}

	return tasks
}

func templateData(task *jobs.Task) (map[string]any, error) {
	if task.PythonWheelTask == nil {
		return nil, fmt.Errorf("PythonWheelTask cannot be nil")
	}

	data := make(map[string]any)
	data["MyName"] = "Trampoline"
	return data, nil
}

func cleanUp(task *jobs.Task) {
	task.PythonWheelTask = nil
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

	trampoline := NewTrampoline("test_trampoline", getTasks, templateData, cleanUp, "Hello from {{.MyName}}")
	err := bundle.Apply(ctx, b, trampoline)
	require.NoError(t, err)

	dir, err := b.InternalDir()
	require.NoError(t, err)
	filename := filepath.Join(dir, "notebook_to_trampoline.py")

	bytes, err := os.ReadFile(filename)
	require.NoError(t, err)

	require.Equal(t, "Hello from Trampoline", string(bytes))

	task := b.Config.Resources.Jobs["test"].Tasks[0]
	require.Equal(t, task.NotebookTask.NotebookPath, ".databricks/bundle/development/.internal/notebook_to_trampoline")
	require.Nil(t, task.PythonWheelTask)
}
