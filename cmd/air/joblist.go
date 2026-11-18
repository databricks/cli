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

// taskSpec returns the AI task definitions a task run runs: its own, or for a
// foreach sweep those of the task it iterates.
func taskSpec(t *jobs.RunTask) (*jobs.AiRuntimeTask, *jobs.GenAiComputeTask) {
	if t.ForEachTask != nil {
		return t.ForEachTask.Task.AiRuntimeTask, t.ForEachTask.Task.GenAiComputeTask
	}
	return t.AiRuntimeTask, t.GenAiComputeTask
}

// isAirTask reports whether a task run is an AI runtime workload: an
// ai_runtime_task, or a legacy gen_ai_compute_task with a training script,
// directly or as the task a foreach sweep iterates.
func isAirTask(t *jobs.RunTask) bool {
	ai, genAI := taskSpec(t)
	return ai != nil || (genAI != nil && genAI.TrainingScriptPath != "")
}

// airTaskKeys returns the task keys of the run's AI runtime tasks in task
// order, each once: a retried task appears in r.Tasks once per attempt. It is
// the one scan for AI runtime tasks; everything after it reads a run narrowed
// to one of them (see taskRun), whose task is Tasks[0].
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

// isAirRun reports whether a run is an AI runtime workload: it has at least
// one AI runtime task.
func isAirRun(r *jobs.Run) bool {
	return len(airTaskKeys(r)) > 0
}

// taskRun returns a copy of r narrowed to the task runs (every attempt, in
// order) of the task with the given key, one of airTaskKeys(r), so the display
// helpers, which read a run's tasks, report that task alone. When the run has
// other tasks too, the copy also takes that task's state (see taskState) and
// drops the run's own start and end times, so a task that hasn't started (or
// was skipped) reports no timing rather than the whole run's. A run whose only
// task is this one keeps its run-level state and timing.
func taskRun(r *jobs.Run, key string) *jobs.Run {
	narrowed := *r
	narrowed.Tasks = nil
	others := false
	for i := range r.Tasks {
		if r.Tasks[i].TaskKey == key {
			narrowed.Tasks = append(narrowed.Tasks, r.Tasks[i])
		} else {
			others = true
		}
	}
	if others {
		latest := &narrowed.Tasks[len(narrowed.Tasks)-1]
		if state := taskState(r, latest); state != r.State {
			narrowed.State = state
			narrowed.Status = latest.Status
		}
		narrowed.StartTime, narrowed.EndTime = 0, 0
	}
	return &narrowed
}

// defaultTaskRun returns r narrowed to its first AI runtime task in task order,
// or r as is when it has none: the run `air get` shows without --task-key.
func defaultTaskRun(r *jobs.Run) *jobs.Run {
	if keys := airTaskKeys(r); len(keys) > 0 {
		return taskRun(r, keys[0])
	}
	return r
}

// taskState returns the state to report for the task whose latest attempt is
// latest: that attempt's state, so one task's result doesn't stand for another
// (a successful train next to a failed eval, or an AI runtime task skipped
// because its prep task failed). It keeps the run's state when the task has
// none, and while the run is still active after an attempt that may be
// retried (one that failed, timed out or hit an internal error), since the task
// may be waiting to retry and runs/get doesn't reliably report how many retries
// are left. A cancelled or skipped task is never retried, so it keeps its own
// state.
// A task waiting to start (pending, queued or waiting to retry, but not blocked
// on an upstream task) reports PENDING, as a single-task run whose task waits
// does (see runWaitingForCompute).
func taskState(r *jobs.Run, latest *jobs.RunTask) *jobs.RunState {
	s := latest.State
	if s == nil {
		return r.State
	}
	if s.LifeCycleState != jobs.RunLifeCycleStateBlocked && isWaitingState(s.LifeCycleState) {
		return &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStatePending, StateMessage: s.StateMessage}
	}
	if mayRetry(s) && !isTerminal(r) {
		return r.State
	}
	return s
}

// mayRetry reports whether a finished task attempt is one Jobs may retry: it
// failed, timed out (with retry_on_timeout) or hit an internal error. A
// successful, cancelled or skipped attempt is final.
func mayRetry(s *jobs.RunState) bool {
	if !isTerminalState(s) {
		return false
	}
	return s.ResultState == jobs.RunResultStateFailed ||
		s.ResultState == jobs.RunResultStateTimedout ||
		s.LifeCycleState == jobs.RunLifeCycleStateInternalError
}

// leadTask returns the task run a run reports: Tasks[0], which is the AI
// runtime task once the run is narrowed (see taskRun). A run without an AI
// runtime task is not narrowed, so its first task stands in, as it always has;
// nil when the run has no tasks.
func leadTask(r *jobs.Run) *jobs.RunTask {
	if len(r.Tasks) == 0 {
		return nil
	}
	return &r.Tasks[0]
}

// firstTask returns the run's lead task (see leadTask), unwrapping a foreach
// sweep to the iterated task; nil when the run has no tasks.
func firstTask(r *jobs.Run) *jobs.RunTask {
	t := leadTask(r)
	if t == nil || t.ForEachTask == nil {
		return t
	}
	// The foreach inner task is a jobs.Task; expose the fields callers read via
	// a synthetic RunTask so they can treat both shapes uniformly.
	ai, genAI := taskSpec(t)
	return &jobs.RunTask{
		TaskKey:          t.TaskKey,
		AiRuntimeTask:    ai,
		GenAiComputeTask: genAI,
		EnvironmentKey:   t.ForEachTask.Task.EnvironmentKey,
	}
}

// airTaskKey returns the key of the AI runtime task a narrowed run shows, or ""
// for a run without one.
func airTaskKey(r *jobs.Run) string {
	if t := leadTask(r); t != nil && isAirTask(t) {
		return t.TaskKey
	}
	return ""
}

// isSweep reports whether the run's lead task fans out into iterations.
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
