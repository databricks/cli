package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"golang.org/x/sync/errgroup"
)

// hydrateConcurrency bounds the parallel runs/get calls when hydrating a batch
// of run ids from the AiTrainingService index.
const hydrateConcurrency = 16

// firstTask returns the run's first task, unwrapping a foreach sweep to the
// iterated task, or nil when the run has no tasks.
func firstTask(r *jobs.Run) *jobs.RunTask {
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
func isAirRun(r *jobs.Run) bool {
	t := firstTask(r)
	if t == nil {
		return false
	}
	return t.AiRuntimeTask != nil ||
		(t.GenAiComputeTask != nil && t.GenAiComputeTask.TrainingScriptPath != "")
}

// isSweep reports whether the run's first task fans out into iterations.
func isSweep(r *jobs.Run) bool {
	return len(r.Tasks) > 0 && r.Tasks[0].ForEachTask != nil
}

// taskRunID returns the run id of the AIR task, used to fetch its MLflow output.
func taskRunID(r *jobs.Run) int64 {
	if len(r.Tasks) == 0 {
		return 0
	}
	return r.Tasks[0].RunId
}

// jobExperiment returns the run's MLflow experiment name (user-folder prefix
// stripped), or "" when there is none.
func jobExperiment(r *jobs.Run) string {
	t := firstTask(r)
	switch {
	case t == nil:
		return ""
	case t.AiRuntimeTask != nil && t.AiRuntimeTask.Experiment != "":
		return stripExperimentUserPrefix(t.AiRuntimeTask.Experiment)
	case t.GenAiComputeTask != nil && t.GenAiComputeTask.MlflowExperimentName != "":
		return stripExperimentUserPrefix(t.GenAiComputeTask.MlflowExperimentName)
	}
	return ""
}

// jobCompute returns the run's accelerator type and count, or ("", 0) when it
// has none.
func jobCompute(r *jobs.Run) (string, int) {
	t := firstTask(r)
	switch {
	case t == nil:
		return "", 0
	case t.AiRuntimeTask != nil && len(t.AiRuntimeTask.Deployments) > 0:
		c := t.AiRuntimeTask.Deployments[0].Compute
		return string(c.AcceleratorType), c.AcceleratorCount
	case t.GenAiComputeTask != nil && t.GenAiComputeTask.Compute != nil:
		c := t.GenAiComputeTask.Compute
		return c.GpuType, c.NumGpus
	}
	return "", 0
}

// jobTiming returns the run's start and end times (epoch ms), preferring the
// first task's window so a run reports its task attempt rather than the wrapper.
func jobTiming(r *jobs.Run) (startMillis, endMillis int64) {
	startMillis, endMillis = r.StartTime, r.EndTime
	if len(r.Tasks) > 0 {
		if t := r.Tasks[0]; t.StartTime > 0 {
			startMillis = t.StartTime
			endMillis = t.EndTime
		}
	}
	return startMillis, endMillis
}

// isTerminal reports whether a run has finished and its details are immutable,
// so its row is safe to cache.
func isTerminal(r *jobs.Run) bool {
	if r.State == nil {
		return false
	}
	switch r.State.LifeCycleState {
	case jobs.RunLifeCycleStateTerminated, jobs.RunLifeCycleStateInternalError, jobs.RunLifeCycleStateSkipped:
		return true
	default:
		return false
	}
}

// baseRunToRun converts a runs/list BaseRun into a Run so the display helpers
// operate on a single type. The two share every field the list path reads.
func baseRunToRun(b jobs.BaseRun) *jobs.Run {
	return &jobs.Run{
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
func fetchJobRun(ctx context.Context, w *databricks.WorkspaceClient, runID int64) (*jobs.Run, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		return nil, fmt.Errorf("failed to get run %d: %w", runID, err)
	}
	return run, nil
}

// hydrateJobRuns fetches the given run ids concurrently via runs/get, preserving
// input order. runs/get enforces per-run view ACLs, so an id the caller can't
// view (403) or that has been purged (404) is dropped; any other error is
// systemic and fails the whole batch.
func hydrateJobRuns(ctx context.Context, w *databricks.WorkspaceClient, ids []int64) ([]*jobs.Run, error) {
	runs := make([]*jobs.Run, len(ids))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(hydrateConcurrency)
	for i, id := range ids {
		g.Go(func() error {
			run, err := fetchJobRun(gctx, w, id)
			if err != nil {
				if apiErr, ok := errors.AsType[*apierr.APIError](err); ok &&
					(apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusNotFound) {
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

	hydrated := make([]*jobs.Run, 0, len(runs))
	for _, run := range runs {
		if run != nil {
			hydrated = append(hydrated, run)
		}
	}
	return hydrated, nil
}
