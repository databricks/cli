package resources

import (
	"testing"

	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestCollectTaskTypes(t *testing.T) {
	jobs := map[string]*Job{
		"nil": nil,
		"a": {JobSettings: jobs.JobSettings{Tasks: []jobs.Task{
			{TaskKey: "nb", NotebookTask: &jobs.NotebookTask{NotebookPath: "/nb"}},
			{TaskKey: "train", AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp"}},
		}}},
		"b": {JobSettings: jobs.JobSettings{Tasks: []jobs.Task{
			// for_each wrapping a spark_python_task: both the wrapper and the
			// nested task type should be recorded.
			{TaskKey: "fan", ForEachTask: &jobs.ForEachTask{Task: jobs.Task{
				SparkPythonTask: &jobs.SparkPythonTask{PythonFile: "main.py"},
			}}},
		}}},
	}
	assert.Equal(t, []string{
		"has_ai_runtime_task",
		"has_for_each_task",
		"has_notebook_task",
		"has_spark_python_task",
	}, collectTaskTypes(jobs))

	assert.Empty(t, collectTaskTypes(nil))
}

// aiRuntimeJob builds a job with a single ai_runtime_task, plus optionally an extra
// no-op task to make it multi-task.
func aiRuntimeJob(extraTask bool) *Job {
	tasks := []jobs.Task{{
		TaskKey:       "train",
		AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp"},
	}}
	if extraTask {
		tasks = append(tasks, jobs.Task{TaskKey: "prep"})
	}
	return &Job{Tasks: tasks}
}

func TestAiRuntimeTaskMetrics(t *testing.T) {
	scheduledJob := aiRuntimeJob(false)
	scheduledJob.Schedule = &jobs.CronSchedule{QuartzCronExpression: "0 0 * * * ?"}

	forEachJob := &Job{Tasks: []jobs.Task{{
		TaskKey:     "fanout",
		ForEachTask: &jobs.ForEachTask{Task: jobs.Task{AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp"}}},
	}}}

	tests := []struct {
		name                          string
		jobs                          map[string]*Job
		present, scheduled, multitask bool
	}{
		{
			name: "no jobs",
			jobs: nil,
		},
		{
			name: "job without ai_runtime_task",
			jobs: map[string]*Job{"j": {JobSettings: jobs.JobSettings{Tasks: []jobs.Task{{TaskKey: "notebook"}}}}},
		},
		{
			name:    "single task",
			jobs:    map[string]*Job{"j": aiRuntimeJob(false)},
			present: true,
		},
		{
			name:    "multi-task",
			jobs:    map[string]*Job{"j": aiRuntimeJob(true)},
			present: true, multitask: true,
		},
		{
			name:    "scheduled single task",
			jobs:    map[string]*Job{"j": scheduledJob},
			present: true, scheduled: true,
		},
		{
			name:    "for_each nested ai_runtime_task",
			jobs:    map[string]*Job{"j": forEachJob},
			present: true,
		},
		{
			name:    "nil job is skipped",
			jobs:    map[string]*Job{"j": nil, "k": aiRuntimeJob(false)},
			present: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			present, scheduled, multitask := aiRuntimeTaskMetrics(tc.jobs)
			assert.Equal(t, tc.present, present, "present")
			assert.Equal(t, tc.scheduled, scheduled, "scheduled")
			assert.Equal(t, tc.multitask, multitask, "multitask")
		})
	}
}

func TestJobDeployTelemetryBoolValues(t *testing.T) {
	assert.Empty(t, (*Job)(nil).DeployTelemetryBoolValues(nil))

	values := (*Job)(nil).DeployTelemetryBoolValues(map[string]any{
		"nil": (*Job)(nil),
		"j":   aiRuntimeJob(false),
	})
	assert.Equal(t, []protos.BoolMapEntry{
		{Key: "has_ai_runtime_task", Value: true},
		{Key: "ai_runtime_task_scheduled", Value: false},
		{Key: "ai_runtime_task_multitask", Value: false},
	}, values)
}
