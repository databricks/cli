package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/apierr"
	"golang.org/x/sync/errgroup"
)

// hydrateConcurrency bounds the parallel runs/get calls when hydrating a batch
// of run ids from the AiTrainingService index.
const hydrateConcurrency = 16

// firstTask returns the run's first task, unwrapping a foreach sweep to the
// iterated task, or nil when the run has no tasks.
func firstTask(r *jobs.GetRunResponse) *jobs.RunTask {
	if len(r.Tasks) == 0 {
		return nil
	}
	t := &r.Tasks[0]
	if t.AiRuntimeTask != nil || t.GenAiComputeTask != nil {
		return t
	}
	if t.ForEachTask != nil {
		// The foreach inner task is a jobs.Task; expose the AI fields via a
		// synthetic RunTask so callers can treat both shapes uniformly.
		return &jobs.RunTask{
			AiRuntimeTask:    t.ForEachTask.Task.AiRuntimeTask,
			GenAiComputeTask: t.ForEachTask.Task.GenAiComputeTask,
		}
	}
	return t
}

// isAirRun reports whether a run is an AI runtime workload: an ai_runtime_task,
// or a legacy gen_ai_compute_task with a training script.
func isAirRun(r *jobs.GetRunResponse) bool {
	t := firstTask(r)
	if t == nil {
		return false
	}
	return t.AiRuntimeTask != nil ||
		(t.GenAiComputeTask != nil && utils.Deref(t.GenAiComputeTask.TrainingScriptPath) != "")
}

// isSweep reports whether the run's first task fans out into iterations.
func isSweep(r *jobs.GetRunResponse) bool {
	return len(r.Tasks) > 0 && r.Tasks[0].ForEachTask != nil
}

// taskRunID returns the run id of the AIR task, used to fetch its MLflow output.
func taskRunID(r *jobs.GetRunResponse) int64 {
	if len(r.Tasks) == 0 {
		return 0
	}
	return utils.Deref(r.Tasks[0].RunId)
}

// jobExperiment returns the run's MLflow experiment name (user-folder prefix
// stripped), or "" when there is none.
func jobExperiment(r *jobs.GetRunResponse) string {
	t := firstTask(r)
	switch {
	case t == nil:
		return ""
	case t.AiRuntimeTask != nil && utils.Deref(t.AiRuntimeTask.Experiment) != "":
		return stripExperimentUserPrefix(utils.Deref(t.AiRuntimeTask.Experiment))
	case t.GenAiComputeTask != nil && utils.Deref(t.GenAiComputeTask.MlflowExperimentName) != "":
		return stripExperimentUserPrefix(utils.Deref(t.GenAiComputeTask.MlflowExperimentName))
	}
	return ""
}

// jobCompute returns the run's accelerator type and count, or ("", 0) when it
// has none.
func jobCompute(r *jobs.GetRunResponse) (string, int) {
	t := firstTask(r)
	switch {
	case t == nil:
		return "", 0
	case t.AiRuntimeTask != nil && len(t.AiRuntimeTask.Deployments) > 0:
		c := t.AiRuntimeTask.Deployments[0].Compute
		return string(c.AcceleratorType), utils.Deref(c.AcceleratorCount)
	case t.GenAiComputeTask != nil && t.GenAiComputeTask.Compute != nil:
		c := t.GenAiComputeTask.Compute
		return utils.Deref(c.GpuType), utils.Deref(c.NumGpus)
	}
	return "", 0
}

// jobTiming returns the run's start and end times (epoch ms), preferring the
// first task's window so a run reports its task attempt rather than the wrapper.
func jobTiming(r *jobs.GetRunResponse) (startMillis, endMillis int64) {
	startMillis, endMillis = utils.Deref(r.StartTime), utils.Deref(r.EndTime)
	if len(r.Tasks) > 0 {
		if t := r.Tasks[0]; utils.Deref(t.StartTime) > 0 {
			startMillis = utils.Deref(t.StartTime)
			endMillis = utils.Deref(t.EndTime)
		}
	}
	return startMillis, endMillis
}

// isTerminal reports whether a run has finished and its details are immutable,
// so its row is safe to cache.
func isTerminal(r *jobs.GetRunResponse) bool {
	if r.State == nil {
		return false
	}
	switch r.State.LifeCycleState {
	case jobs.RunLifeCycleState_RunLifeCycleState_Terminated, jobs.RunLifeCycleState_RunLifeCycleState_InternalError, jobs.RunLifeCycleState_RunLifeCycleState_Skipped:
		return true
	default:
		return false
	}
}

// baseRunToRun converts a runs/list BaseRun into a Run so the display helpers
// operate on a single type. The two share every field the list path reads.
func baseRunToRun(b jobs.BaseRun) *jobs.GetRunResponse {
	return &jobs.GetRunResponse{
		RunId:           b.RunId,
		RunName:         b.RunName,
		CreatorUserName: b.CreatorUserName,
		StartTime:       b.StartTime,
		EndTime:         b.EndTime,
		State:           b.State,
		Tasks:           b.Tasks,
	}
}

// fetchJobRun fetches a single run via runs/get; expand_tasks is implied by the
// typed SDK, which returns the ai_runtime_task in the response.
func fetchJobRun(ctx context.Context, w *sdkclients.Clients, runID int64) (*jobs.GetRunResponse, error) {
	run, err := getRun(ctx, w, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run %d: %w", runID, err)
	}
	return run, nil
}

// hydrateJobRuns fetches the given run ids concurrently via runs/get, preserving
// input order. runs/get enforces per-run view ACLs, so an id the caller can't
// view (403) or that has been purged (404) is dropped; any other error is
// systemic and fails the whole batch.
func hydrateJobRuns(ctx context.Context, w *sdkclients.Clients, ids []int64) ([]*jobs.GetRunResponse, error) {
	runs := make([]*jobs.GetRunResponse, len(ids))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(hydrateConcurrency)
	for i, id := range ids {
		g.Go(func() error {
			run, err := fetchJobRun(gctx, w, id)
			if err != nil {
				if apiErr, ok := errors.AsType[*apierr.APIError](err); ok &&
					(apiErr.HTTPStatusCode() == http.StatusForbidden || apiErr.HTTPStatusCode() == http.StatusNotFound) {
					return nil // not viewable or purged: drop this id
				}
				return fmt.Errorf("failed to get run %d: %w", id, err)
			}
			runs[i] = run
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	hydrated := make([]*jobs.GetRunResponse, 0, len(runs))
	for _, run := range runs {
		if run != nil {
			hydrated = append(hydrated, run)
		}
	}
	return hydrated, nil
}
