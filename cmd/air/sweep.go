package aircmd

import (
	"context"
	"strconv"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// sweepInfo summarizes a "foreach" run, which fans a single config out into many
// iterations (a hyperparameter sweep).
type sweepInfo struct {
	Total     int         `json:"total"`
	Succeeded int         `json:"succeeded"`
	Failed    int         `json:"failed"`
	Active    int         `json:"active"`
	Completed int         `json:"completed"`
	Tasks     []sweepTask `json:"tasks"`
}

// sweepTask is one iteration of a sweep.
type sweepTask struct {
	TaskKey string `json:"task_key"`
	RunID   string `json:"run_id"`
	Status  string `json:"status"`
	// ExperimentName is null when the iteration names no experiment; Experiment
	// is the same value for the text table, which shows it blank.
	ExperimentName *string `json:"experiment_name"`
	Experiment     string  `json:"-"`
}

// findForEachTask returns the run's foreach task if it has one, or nil. A run is
// a sweep when one of its tasks fans out into iterations.
func findForEachTask(run *jobs.Run) *jobs.RunTask {
	for i := range run.Tasks {
		if run.Tasks[i].ForEachTask != nil {
			return &run.Tasks[i]
		}
	}
	return nil
}

// buildSweepInfo gathers the iteration counts and per-iteration rows for a
// sweep. The counts come from the task we already have; the individual
// iterations require a second lookup. If that lookup fails we still return the
// counts (logging the failure) so the user sees the summary.
func buildSweepInfo(ctx context.Context, w *databricks.WorkspaceClient, task *jobs.RunTask) *sweepInfo {
	info := &sweepInfo{Tasks: []sweepTask{}}
	if task.ForEachTask.Stats != nil && task.ForEachTask.Stats.TaskRunStats != nil {
		stats := task.ForEachTask.Stats.TaskRunStats
		info.Total = stats.TotalIterations
		info.Succeeded = stats.SucceededIterations
		info.Failed = stats.FailedIterations
		info.Active = stats.ActiveIterations
		info.Completed = stats.CompletedIterations
	}

	// The iterations are returned as part of a run lookup on the foreach task.
	iterated, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: task.RunId})
	if err != nil {
		log.Debugf(ctx, "air get: could not fetch sweep iterations: %v", err)
		return info
	}

	for _, it := range iterated.Iterations {
		row := sweepTask{
			TaskKey: it.TaskKey,
			RunID:   strconv.FormatInt(it.RunId, 10),
			Status:  runStatus(it.State),
		}
		if it.GenAiComputeTask != nil && it.GenAiComputeTask.MlflowExperimentName != "" {
			exp := stripExperimentUserPrefix(it.GenAiComputeTask.MlflowExperimentName)
			row.Experiment = exp
			row.ExperimentName = &exp
		}
		info.Tasks = append(info.Tasks, row)
	}
	return info
}
