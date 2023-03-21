package run

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"golang.org/x/exp/maps"
)

type JobOutput struct {
	// URL of the job run
	RunPageUrl string

	// output for tasks with a non empty output
	TaskOutputs map[string]RunOutput
}

// TODO: Print the output respecting the execution order (https://github.com/databricks/bricks/issues/259)
func (out *JobOutput) String() (string, error) {
	if len(out.TaskOutputs) == 0 {
		return "", nil
	}
	// When only one task, just return that output without any formatting
	if len(out.TaskOutputs) == 1 {
		for _, v := range out.TaskOutputs {
			return v.String()
		}
	}
	result := strings.Builder{}
	result.WriteString(fmt.Sprintf("Run URL: %s\n", out.RunPageUrl))

	taskKeys := maps.Keys(out.TaskOutputs)
	sort.Strings(taskKeys)
	for _, k := range taskKeys {
		taskString, err := out.TaskOutputs[k].String()
		if err != nil {
			return "", nil
		}
		result.WriteString("=======\n")
		result.WriteString(fmt.Sprintf("Task %s:\n", k))
		result.WriteString(fmt.Sprintf("%s\n", taskString))
	}
	return result.String(), nil
}

func getJobOutput(ctx context.Context, w *databricks.WorkspaceClient, runId int64) (*JobOutput, error) {
	jobRun, err := w.Jobs.GetRun(ctx, jobs.GetRun{
		RunId: runId,
	})
	if err != nil {
		return nil, err
	}
	result := &JobOutput{
		TaskOutputs: make(map[string]RunOutput),
	}
	result.RunPageUrl = jobRun.RunPageUrl

	for _, task := range jobRun.Tasks {
		jobRunOutput, err := w.Jobs.GetRunOutput(ctx, jobs.GetRunOutput{
			RunId: task.RunId,
		})
		if err != nil {
			return nil, err
		}
		result.TaskOutputs[task.TaskKey] = toRunOutput(jobRunOutput)
	}
	return result, nil
}
