package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"golang.org/x/sync/errgroup"
)

// hydrateConcurrency bounds the parallel runs/get calls when hydrating a batch
// of run ids from the AiTrainingService index.
const hydrateConcurrency = 16

// isAirTask reports whether a task run is an AI runtime workload: an
// ai_runtime_task, or a legacy gen_ai_compute_task with a training script. A
// foreach task counts when the task it iterates is one.
func isAirTask(t *jobs.RunTask) bool {
	if t.ForEachTask != nil {
		inner := t.ForEachTask.Task
		return inner.AiRuntimeTask != nil ||
			(inner.GenAiComputeTask != nil && inner.GenAiComputeTask.TrainingScriptPath != "")
	}
	return t.AiRuntimeTask != nil ||
		(t.GenAiComputeTask != nil && t.GenAiComputeTask.TrainingScriptPath != "")
}

// firstAirTaskIndex returns the index in r.Tasks of the run's first AI runtime
// task in task order, or -1 when it has none. A job may put other tasks before
// it (the official multi-task DAB runs a notebook prep step first), so the
// first task is not inspected directly.
func firstAirTaskIndex(r *jobs.Run) int {
	for i := range r.Tasks {
		if isAirTask(&r.Tasks[i]) {
			return i
		}
	}
	return -1
}

// airTaskKeys returns the task keys of the run's AI runtime tasks in task
// order, each once: a retried task appears in r.Tasks once per attempt.
func airTaskKeys(r *jobs.Run) []string {
	var keys []string
	for i := range r.Tasks {
		t := &r.Tasks[i]
		if isAirTask(t) && !slices.Contains(keys, t.TaskKey) {
			keys = append(keys, t.TaskKey)
		}
	}
	return keys
}

// taskRun returns a copy of r narrowed to the task runs (every attempt, in
// order) of the AI runtime task with the given key, so the display helpers,
// which read a run's tasks, report that task alone. Its state is that task's
// (see taskState). It returns nil when key is not one of the run's AI runtime
// tasks.
func taskRun(r *jobs.Run, key string) *jobs.Run {
	if !slices.Contains(airTaskKeys(r), key) {
		return nil
	}
	narrowed := *r
	narrowed.Tasks = nil
	for i := range r.Tasks {
		if r.Tasks[i].TaskKey == key {
			narrowed.Tasks = append(narrowed.Tasks, r.Tasks[i])
		}
	}
	latest := &narrowed.Tasks[len(narrowed.Tasks)-1]
	if state := taskState(r, latest); state != r.State {
		narrowed.State = state
		narrowed.Status = latest.Status
	}
	return &narrowed
}

// taskState returns the state to report for the task whose latest attempt is
// latest: that attempt's state, so one task's result doesn't stand for another
// (a successful train next to a failed eval, or an AI runtime task skipped
// because its prep task failed). It keeps the run's state when the task has
// none, and while the run is still active after a failed attempt, since the
// task may be waiting to retry and runs/get doesn't reliably report how many
// retries are left.
func taskState(r *jobs.Run, latest *jobs.RunTask) *jobs.RunState {
	s := latest.State
	if s == nil {
		return r.State
	}
	failed := isTerminalState(s) && s.ResultState != jobs.RunResultStateSuccess
	if failed && !isTerminal(r) {
		return r.State
	}
	return s
}

// airTask returns the task run of the run's first AI runtime task as the Jobs
// API reports it (for a sweep, the foreach wrapper, which carries the state and
// timing), or nil when the run has none.
func airTask(r *jobs.Run) *jobs.RunTask {
	i := firstAirTaskIndex(r)
	if i < 0 {
		return nil
	}
	return &r.Tasks[i]
}

// leadTask is airTask with a fallback to the first task for a run without an AI
// runtime task (such as a legacy gen_ai_compute_task without a training
// script), whose fields `air get` still shows; nil when the run has no tasks.
func leadTask(r *jobs.Run) *jobs.RunTask {
	if t := airTask(r); t != nil {
		return t
	}
	if len(r.Tasks) == 0 {
		return nil
	}
	return &r.Tasks[0]
}

// firstTask returns the run's first AI runtime task, unwrapping a foreach sweep
// to the iterated task. A run without one falls back to its first task, so a
// legacy run's fields are still read; nil when the run has no tasks.
func firstTask(r *jobs.Run) *jobs.RunTask {
	t := leadTask(r)
	if t == nil {
		return nil
	}
	if t.ForEachTask != nil {
		// The foreach inner task is a jobs.Task; expose the AI fields via a
		// synthetic RunTask so callers can treat both shapes uniformly.
		return &jobs.RunTask{
			TaskKey:          t.TaskKey,
			AiRuntimeTask:    t.ForEachTask.Task.AiRuntimeTask,
			GenAiComputeTask: t.ForEachTask.Task.GenAiComputeTask,
		}
	}
	return t
}

// isAirRun reports whether a run is an AI runtime workload: it has an
// ai_runtime_task, or a legacy gen_ai_compute_task with a training script.
func isAirRun(r *jobs.Run) bool {
	return firstAirTaskIndex(r) >= 0
}

// isSweep reports whether the run's AI runtime task fans out into iterations.
func isSweep(r *jobs.Run) bool {
	t := leadTask(r)
	return t != nil && t.ForEachTask != nil
}

// retryTasks returns the run's task runs with only the fields that pick its
// MLflow run (see mlflowIDsFromRetries), so the list cache stays small.
func retryTasks(r *jobs.Run) []jobs.RunTask {
	tasks := make([]jobs.RunTask, len(r.Tasks))
	for i, t := range r.Tasks {
		tasks[i] = jobs.RunTask{RunId: t.RunId, TaskKey: t.TaskKey, AttemptNumber: t.AttemptNumber}
	}
	return tasks
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
// AI runtime task's window so a run reports its task attempt rather than the
// wrapper.
func jobTiming(r *jobs.Run) (startMillis, endMillis int64) {
	startMillis, endMillis = r.StartTime, r.EndTime
	if t := leadTask(r); t != nil && t.StartTime > 0 {
		startMillis = t.StartTime
		endMillis = t.EndTime
	}
	return startMillis, endMillis
}

// isTerminal reports whether a run has finished and its details are immutable,
// so its row is safe to cache.
func isTerminal(r *jobs.Run) bool {
	return r.State != nil && isTerminalState(r.State)
}

// isTerminalState reports whether a run or task state is final.
func isTerminalState(s *jobs.RunState) bool {
	switch s.LifeCycleState {
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
// view (403) or that has been purged (404, or the 400 "Run <id> does not exist."
// that the SDK maps to ErrResourceDoesNotExist) is dropped; any other error is
// systemic and fails the whole batch.
func hydrateJobRuns(ctx context.Context, w *databricks.WorkspaceClient, ids []int64) ([]*jobs.Run, error) {
	runs := make([]*jobs.Run, len(ids))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(hydrateConcurrency)
	for i, id := range ids {
		g.Go(func() error {
			run, err := fetchJobRun(gctx, w, id)
			if err != nil {
				if errors.Is(err, apierr.ErrResourceDoesNotExist) {
					return nil // purged: drop this id
				}
				if apiErr, ok := errors.AsType[*apierr.APIError](err); ok &&
					(apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusNotFound) {
					return nil // not viewable or purged: drop this id
				}
				return err
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
