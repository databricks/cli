package output

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
)

type TaskOutput struct {
	TaskKey string
	Output  RunOutput
	EndTime int64
}

type JobOutput struct {
	// output for tasks with a non empty output
	TaskOutputs []TaskOutput `json:"task_outputs"`
}

// Returns tasks output in text form sorted in execution order based on task end time
func (out *JobOutput) String() (string, error) {
	if len(out.TaskOutputs) == 0 {
		return "", nil
	}
	// When only one task, just return that output without any formatting
	if len(out.TaskOutputs) == 1 {
		for _, v := range out.TaskOutputs {
			return v.Output.String()
		}
	}
	result := strings.Builder{}
	result.WriteString("Output:\n")
	slices.SortFunc(out.TaskOutputs, func(a, b TaskOutput) int {
		return cmp.Compare(a.EndTime, b.EndTime)
	})
	for _, v := range out.TaskOutputs {
		if v.Output == nil {
			continue
		}
		taskString, err := v.Output.String()
		if err != nil {
			return "", nil //nolint:nilerr // skip tasks with unparseable output
		}
		result.WriteString("=======\n")
		fmt.Fprintf(&result, "Task %s:\n", v.TaskKey)
		result.WriteString(taskString + "\n")
	}
	return result.String(), nil
}

// GetRun fetches a job run, following next_page_token so that the result
// holds all tasks.
func GetRun(ctx context.Context, w *sdkclients.Clients, runId int64) (*sdkjobs.GetRunResponse, error) {
	req := sdkjobs.GetRunRequest{RunId: new(runId)}
	run, err := w.Jobs.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}
	for utils.Deref(run.NextPageToken) != "" {
		req.PageToken = run.NextPageToken
		next, err := w.Jobs.GetRun(ctx, req)
		if err != nil {
			return nil, err
		}
		run.Tasks = append(run.Tasks, next.Tasks...)
		run.JobClusters = append(run.JobClusters, next.JobClusters...)
		run.JobParameters = append(run.JobParameters, next.JobParameters...)
		run.RepairHistory = append(run.RepairHistory, next.RepairHistory...)
		run.NextPageToken = next.NextPageToken
	}
	return run, nil
}

func GetJobOutput(ctx context.Context, w *sdkclients.Clients, runId int64) (*JobOutput, error) {
	jobRun, err := GetRun(ctx, w, runId)
	if err != nil {
		return nil, err
	}
	result := &JobOutput{
		TaskOutputs: make([]TaskOutput, 0),
	}
	for _, task := range jobRun.Tasks {
		sdkOutput, err := w.Jobs.GetRunOutput(ctx, sdkjobs.GetRunOutputRequest{
			RunId: task.RunId,
		})
		if err != nil {
			return nil, err
		}
		jobRunOutput, err := jobs.GetRunOutputResponseFromSDK(sdkOutput)
		if err != nil {
			return nil, err
		}
		out := toRunOutput(jobRunOutput)
		if out == nil {
			continue
		}
		task := TaskOutput{TaskKey: utils.Deref(task.TaskKey), Output: out, EndTime: utils.Deref(task.EndTime)}
		result.TaskOutputs = append(result.TaskOutputs, task)
	}
	return result, nil
}
