package run

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type JobOutput struct {
	RunPageUrl string
	Tasks      map[string]RunOutput
}

// TODO: Print the output respecting the execution order (https://github.com/databricks/bricks/issues/259)
func (out *JobOutput) String() (string, error) {
	if len(out.Tasks) == 0 {
		return "", nil
	}
	// When only one task, just return that output without any formatting
	if len(out.Tasks) == 1 {
		for _, v := range out.Tasks {
			return v.String()
		}
	}
	result := "=======\n"
	result += fmt.Sprintf("Run url: %s\n", out.RunPageUrl)
	for k, v := range out.Tasks {
		taskString, err := v.String()
		if err != nil {
			return "", nil
		}
		result += "=======\n"
		result += fmt.Sprintf("Task %s:\n", k)
		result += fmt.Sprintf("%s\n", taskString)
	}
	return result, nil
}

func (r *jobRunner) GetJobOutput(ctx context.Context, runId int64) (*JobOutput, error) {
	w := r.bundle.WorkspaceClient()
	jobRun, err := w.Jobs.GetRun(ctx, jobs.GetRun{
		RunId: runId,
	})
	if err != nil {
		return nil, err
	}
	result := &JobOutput{
		Tasks: make(map[string]RunOutput),
	}
	result.RunPageUrl = jobRun.RunPageUrl

	for _, task := range jobRun.Tasks {
		jobRunOutput, err := w.Jobs.GetRunOutput(ctx, jobs.GetRunOutput{
			RunId: task.RunId,
		})
		if err != nil {
			return nil, err
		}
		taskRunOutput, err := toRunOutput(*jobRunOutput)
		if err != nil {
			return nil, err
		}
		result.Tasks[task.TaskKey] = taskRunOutput
	}
	return result, nil
}
