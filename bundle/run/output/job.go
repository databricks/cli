package output

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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
	sort.Slice(out.TaskOutputs, func(i, j int) bool {
		return out.TaskOutputs[i].EndTime < out.TaskOutputs[j].EndTime
	})
	for _, v := range out.TaskOutputs {
		if v.Output == nil {
			continue
		}
		taskString, err := v.Output.String()
		if err != nil {
			return "", nil
		}
		result.WriteString("=======\n")
		result.WriteString(fmt.Sprintf("Task %s:\n", v.TaskKey))
		result.WriteString(taskString + "\n")
	}
	return result.String(), nil
}

func GetJobOutput(ctx context.Context, w *databricks.WorkspaceClient, runId int64) (*JobOutput, error) {
	jobRun, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{
		RunId: runId,
	})
	if err != nil {
		return nil, err
	}
	result := &JobOutput{
		TaskOutputs: make([]TaskOutput, 0),
	}
	for _, task := range jobRun.Tasks {
		jobRunOutput, err := w.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{
			RunId: task.RunId,
		})
		if err != nil {
			return nil, err
		}
		out := toRunOutput(jobRunOutput)
		if out == nil {
			continue
		}
		task := TaskOutput{TaskKey: task.TaskKey, Output: out, EndTime: task.EndTime}
		result.TaskOutputs = append(result.TaskOutputs, task)
	}
	return result, nil
}
