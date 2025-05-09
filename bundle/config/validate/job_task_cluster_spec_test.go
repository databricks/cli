package validate

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestJobTaskClusterSpec(t *testing.T) {
	expectedSummary := "Missing required cluster or environment settings"

	type testCase struct {
		name         string
		task         jobs.Task
		errorPath    string
		errorDetail  string
		errorSummary string
	}

	testCases := []testCase{
		{
			name: "valid notebook task",
			task: jobs.Task{
				// while a cluster is needed, it will use notebook environment to create one
				NotebookTask: &jobs.NotebookTask{},
			},
		},
		{
			name: "valid notebook task (job_cluster_key)",
			task: jobs.Task{
				JobClusterKey: "cluster1",
				NotebookTask:  &jobs.NotebookTask{},
			},
		},
		{
			name: "valid notebook task (new_cluster)",
			task: jobs.Task{
				NewCluster:   &compute.ClusterSpec{},
				NotebookTask: &jobs.NotebookTask{},
			},
		},
		{
			name: "valid notebook task (existing_cluster_id)",
			task: jobs.Task{
				ExistingClusterId: "cluster1",
				NotebookTask:      &jobs.NotebookTask{},
			},
		},
		{
			name: "valid SQL notebook task",
			task: jobs.Task{
				NotebookTask: &jobs.NotebookTask{
					WarehouseId: "warehouse1",
				},
			},
		},
		{
			name: "valid python wheel task",
			task: jobs.Task{
				JobClusterKey:   "cluster1",
				PythonWheelTask: &jobs.PythonWheelTask{},
			},
		},
		{
			name: "valid python wheel task (environment_key)",
			task: jobs.Task{
				EnvironmentKey:  "environment1",
				PythonWheelTask: &jobs.PythonWheelTask{},
			},
		},
		{
			name: "valid dbt task",
			task: jobs.Task{
				JobClusterKey: "cluster1",
				DbtTask:       &jobs.DbtTask{},
			},
		},
		{
			name: "valid spark jar task",
			task: jobs.Task{
				JobClusterKey: "cluster1",
				SparkJarTask:  &jobs.SparkJarTask{},
			},
		},
		{
			name: "valid spark submit",
			task: jobs.Task{
				NewCluster:      &compute.ClusterSpec{},
				SparkSubmitTask: &jobs.SparkSubmitTask{},
			},
		},
		{
			name: "valid spark python task",
			task: jobs.Task{
				JobClusterKey:   "cluster1",
				SparkPythonTask: &jobs.SparkPythonTask{},
			},
		},
		{
			name: "valid SQL task",
			task: jobs.Task{
				SqlTask: &jobs.SqlTask{},
			},
		},
		{
			name: "valid pipeline task",
			task: jobs.Task{
				PipelineTask: &jobs.PipelineTask{},
			},
		},
		{
			name: "valid run job task",
			task: jobs.Task{
				RunJobTask: &jobs.RunJobTask{},
			},
		},
		{
			name: "valid condition task",
			task: jobs.Task{
				ConditionTask: &jobs.ConditionTask{},
			},
		},
		{
			name: "valid for each task",
			task: jobs.Task{
				ForEachTask: &jobs.ForEachTask{
					Task: jobs.Task{
						JobClusterKey: "cluster1",
						NotebookTask:  &jobs.NotebookTask{},
					},
				},
			},
		},
		{
			name: "invalid python wheel task",
			task: jobs.Task{
				PythonWheelTask: &jobs.PythonWheelTask{},
				TaskKey:         "my_task",
			},
			errorPath: "resources.jobs.job1.tasks[0]",
			errorDetail: `Task "my_task" requires a cluster or an environment to run.
Specify one of the following fields: job_cluster_key, environment_key, existing_cluster_id, new_cluster.`,
			errorSummary: expectedSummary,
		},
		{
			name: "invalid for each task",
			task: jobs.Task{
				ForEachTask: &jobs.ForEachTask{
					Task: jobs.Task{
						PythonWheelTask: &jobs.PythonWheelTask{},
						TaskKey:         "my_task",
					},
				},
			},
			errorPath: "resources.jobs.job1.tasks[0].for_each_task.task",
			errorDetail: `Task "my_task" requires a cluster or an environment to run.
Specify one of the following fields: job_cluster_key, environment_key, existing_cluster_id, new_cluster.`,
			errorSummary: expectedSummary,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			job := &resources.Job{
				JobSettings: jobs.JobSettings{
					Tasks: []jobs.Task{tc.task},
				},
			}

			b := createBundle(map[string]*resources.Job{"job1": job})
			diags := JobTaskClusterSpec().Apply(context.Background(), b)

			if tc.errorPath != "" || tc.errorDetail != "" || tc.errorSummary != "" {
				assert.Len(t, diags, 1)
				assert.Len(t, diags[0].Paths, 1)

				diag := diags[0]

				assert.Equal(t, tc.errorPath, diag.Paths[0].String())
				assert.Equal(t, tc.errorSummary, diag.Summary)
				assert.Equal(t, tc.errorDetail, diag.Detail)
			} else {
				assert.ElementsMatch(t, []string{}, diags)
			}
		})
	}
}

func createBundle(jobs map[string]*resources.Job) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: jobs,
			},
		},
	}
}
