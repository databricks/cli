package aircmd

import (
	"context"
	"strconv"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
)

// sweepInfo summarizes a "foreach" run, which fans a single config out into many
// iterations (a hyperparameter sweep). It is shown only in text output.
type sweepInfo struct {
	Total     int
	Succeeded int
	Failed    int
	Active    int
	Completed int
	Tasks     []sweepTask
}

// sweepTask is one iteration of a sweep.
type sweepTask struct {
	TaskKey    string
	RunID      string
	Status     string
	Experiment string
}

// findForEachTask returns the run's foreach task if it has one, or nil. A run is
// a sweep when one of its tasks fans out into iterations.
func findForEachTask(run *jobs.GetRunResponse) *jobs.RunTask {
	for i := range run.Tasks {
		if run.Tasks[i].ForEachTask != nil {
			return &run.Tasks[i]
		}
	}
	return nil
}

// buildSweepInfo gathers the iteration counts and per-iteration rows for a
// sweep. The counts come from the foreach task of the parent run; the
// individual iterations require a lookup of the foreach task run. If a lookup
// fails we still return what we have (logging the failure) so the user sees
// the summary.
//
// sdk-go models neither for_each_task.stats nor iterations on runs/get, so both
// are read with raw requests.
func buildSweepInfo(ctx context.Context, w *sdkclients.Clients, runID int64, task *jobs.RunTask) *sweepInfo {
	info := &sweepInfo{}

	var parent struct {
		Tasks []struct {
			TaskKey     string `json:"task_key"`
			ForEachTask *struct {
				Stats *struct {
					TaskRunStats *struct {
						TotalIterations     int `json:"total_iterations"`
						SucceededIterations int `json:"succeeded_iterations"`
						FailedIterations    int `json:"failed_iterations"`
						ActiveIterations    int `json:"active_iterations"`
						CompletedIterations int `json:"completed_iterations"`
					} `json:"task_run_stats"`
				} `json:"stats"`
			} `json:"for_each_task"`
		} `json:"tasks"`
	}
	if err := getRunRaw(ctx, w, runID, &parent); err != nil {
		log.Debugf(ctx, "air get: could not fetch sweep stats: %v", err)
	}
	for _, t := range parent.Tasks {
		if t.TaskKey != utils.Deref(task.TaskKey) || t.ForEachTask == nil || t.ForEachTask.Stats == nil || t.ForEachTask.Stats.TaskRunStats == nil {
			continue
		}
		stats := t.ForEachTask.Stats.TaskRunStats
		info.Total = stats.TotalIterations
		info.Succeeded = stats.SucceededIterations
		info.Failed = stats.FailedIterations
		info.Active = stats.ActiveIterations
		info.Completed = stats.CompletedIterations
	}

	// The iterations are returned as part of a run lookup on the foreach task.
	var iterated struct {
		Iterations []jobs.RunTask `json:"iterations"`
	}
	if err := getRunRaw(ctx, w, utils.Deref(task.RunId), &iterated); err != nil {
		log.Debugf(ctx, "air get: could not fetch sweep iterations: %v", err)
		return info
	}

	for _, it := range iterated.Iterations {
		row := sweepTask{
			TaskKey: utils.Deref(it.TaskKey),
			RunID:   strconv.FormatInt(utils.Deref(it.RunId), 10),
			Status:  runStatus(it.State),
		}
		if it.GenAiComputeTask != nil && utils.Deref(it.GenAiComputeTask.MlflowExperimentName) != "" {
			row.Experiment = stripExperimentUserPrefix(*it.GenAiComputeTask.MlflowExperimentName)
		}
		info.Tasks = append(info.Tasks, row)
	}
	return info
}
