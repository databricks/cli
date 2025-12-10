package validate

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createBundleWithForEachTask(parentTask jobs.Task) *bundle.Bundle {
	if parentTask.ForEachTask == nil {
		parentTask.ForEachTask = &jobs.ForEachTask{
			Inputs: "[1, 2, 3]",
			Task: jobs.Task{
				TaskKey: "child_task",
				NotebookTask: &jobs.NotebookTask{
					NotebookPath: "test.py",
				},
			},
		}
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name:  "My Job",
							Tasks: []jobs.Task{parentTask},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.jobs.job1.tasks[0]", []dyn.Location{{File: "job.yml", Line: 1, Column: 1}})
	return b
}

func TestForEachTask_MaxRetriesError(t *testing.T) {
	ctx := context.Background()
	b := createBundleWithForEachTask(jobs.Task{
		TaskKey:    "parent_task",
		MaxRetries: 3,
	})

	diags := ForEachTask().Apply(ctx, b)

	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Equal(t, "Invalid max_retries configuration for for_each_task", diags[0].Summary)
	assert.Contains(t, diags[0].Detail, `Task "parent_task" has max_retries defined at the parent level`)
	assert.Contains(t, diags[0].Detail, "for_each_task.task.max_retries")
}

func TestForEachTask_MinRetryIntervalWarning(t *testing.T) {
	ctx := context.Background()
	b := createBundleWithForEachTask(jobs.Task{
		TaskKey:                "parent_task",
		MinRetryIntervalMillis: 1000,
	})

	diags := ForEachTask().Apply(ctx, b)

	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Equal(t, "Invalid min_retry_interval_millis configuration for for_each_task", diags[0].Summary)
	assert.Contains(t, diags[0].Detail, `Task "parent_task" has min_retry_interval_millis defined at the parent level`)
	assert.Contains(t, diags[0].Detail, "for_each_task.task.min_retry_interval_millis")
}

func TestForEachTask_RetryOnTimeoutWarning(t *testing.T) {
	ctx := context.Background()
	b := createBundleWithForEachTask(jobs.Task{
		TaskKey:        "parent_task",
		RetryOnTimeout: true,
	})

	diags := ForEachTask().Apply(ctx, b)

	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Equal(t, "Invalid retry_on_timeout configuration for for_each_task", diags[0].Summary)
	assert.Contains(t, diags[0].Detail, `Task "parent_task" has retry_on_timeout defined at the parent level`)
	assert.Contains(t, diags[0].Detail, "for_each_task.task.retry_on_timeout")
}

func TestForEachTask_ValidConfiguration(t *testing.T) {
	ctx := context.Background()
	b := createBundleWithForEachTask(jobs.Task{
		TaskKey: "parent_task",
		ForEachTask: &jobs.ForEachTask{
			Inputs: "[1, 2, 3]",
			Task: jobs.Task{
				TaskKey:    "child_task",
				MaxRetries: 3,
				NotebookTask: &jobs.NotebookTask{
					NotebookPath: "test.py",
				},
			},
		},
	})

	diags := ForEachTask().Apply(ctx, b)
	assert.Empty(t, diags)
}
