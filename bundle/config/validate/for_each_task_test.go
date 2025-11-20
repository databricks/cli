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

func TestForEachTask_InvalidRetryFields(t *testing.T) {
	tests := []struct {
		name             string
		task             jobs.Task
		expectedSeverity diag.Severity
		expectedSummary  string
		expectedDetail   string
	}{
		{
			name: "max_retries on parent",
			task: jobs.Task{
				TaskKey:    "parent_task",
				MaxRetries: 3,
			},
			expectedSeverity: diag.Error,
			expectedSummary:  "Invalid max_retries configuration for for_each_task",
			expectedDetail:   "max_retries must be defined on the nested task",
		},
		{
			name: "min_retry_interval_millis on parent",
			task: jobs.Task{
				TaskKey:                "parent_task",
				MinRetryIntervalMillis: 1000,
			},
			expectedSeverity: diag.Warning,
			expectedSummary:  "Invalid min_retry_interval_millis configuration for for_each_task",
			expectedDetail:   "min_retry_interval_millis must be defined on the nested task",
		},
		{
			name: "retry_on_timeout on parent",
			task: jobs.Task{
				TaskKey:        "parent_task",
				RetryOnTimeout: true,
			},
			expectedSeverity: diag.Warning,
			expectedSummary:  "Invalid retry_on_timeout configuration for for_each_task",
			expectedDetail:   "retry_on_timeout must be defined on the nested task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			b := createBundleWithForEachTask(tt.task)

			diags := ForEachTask().Apply(ctx, b)

			require.Len(t, diags, 1)
			assert.Equal(t, tt.expectedSeverity, diags[0].Severity)
			assert.Equal(t, tt.expectedSummary, diags[0].Summary)
			assert.Contains(t, diags[0].Detail, tt.expectedDetail)
		})
	}
}

func TestForEachTask_MultipleRetryFieldsOnParent(t *testing.T) {
	ctx := context.Background()
	b := createBundleWithForEachTask(jobs.Task{
		TaskKey:                "parent_task",
		MaxRetries:             3,
		MinRetryIntervalMillis: 1000,
		RetryOnTimeout:         true,
	})

	diags := ForEachTask().Apply(ctx, b)
	require.Len(t, diags, 3)

	errorCount := 0
	warningCount := 0
	for _, d := range diags {
		if d.Severity == diag.Error {
			errorCount++
		} else if d.Severity == diag.Warning {
			warningCount++
		}
	}
	assert.Equal(t, 1, errorCount)
	assert.Equal(t, 2, warningCount)
}

func TestForEachTask_ValidConfigurationOnChild(t *testing.T) {
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

func TestForEachTask_NoForEachTask(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name: "My Job",
							Tasks: []jobs.Task{
								{
									TaskKey:    "simple_task",
									MaxRetries: 3,
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "test.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.jobs.job1.tasks[0]", []dyn.Location{{File: "job.yml", Line: 1, Column: 1}})

	diags := ForEachTask().Apply(ctx, b)
	assert.Empty(t, diags)
}

func TestForEachTask_RetryOnTimeoutFalse(t *testing.T) {
	ctx := context.Background()
	b := createBundleWithForEachTask(jobs.Task{
		TaskKey:        "parent_task",
		RetryOnTimeout: false,
	})

	diags := ForEachTask().Apply(ctx, b)
	assert.Empty(t, diags)
}
