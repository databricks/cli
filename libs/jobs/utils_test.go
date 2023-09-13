package jobs_utils

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestCorrectlyFilterTasksByFn(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:         "job1_key1",
									PythonWheelTask: &jobs.PythonWheelTask{},
								},
								{
									TaskKey:      "job1_key2",
									NotebookTask: &jobs.NotebookTask{},
								},
							},
						},
					},
					"job2": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:         "job1_key1",
									PythonWheelTask: &jobs.PythonWheelTask{},
								},
								{
									TaskKey:      "job2_key2",
									NotebookTask: &jobs.NotebookTask{},
								},
							},
						},
					},
				},
			},
		},
	}

	tasks := GetTasksWithJobKeyBy(bundle, func(task *jobs.Task) bool {
		return task.PythonWheelTask != nil
	})

	require.Len(t, tasks, 2)

	require.Equal(t, "job1", tasks[0].JobKey)
	require.Equal(t, "job1_key1", tasks[0].Task.TaskKey)

	require.Equal(t, "job2", tasks[1].JobKey)
	require.Equal(t, "job1_key1", tasks[1].Task.TaskKey)
}
