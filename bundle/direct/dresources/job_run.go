package dresources

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/run/progress"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// jobRunTimeout matches the timeout `bundle run` allows a run (bundle/run/job.go).
const jobRunTimeout = 24 * time.Hour

// JobRunState is what we persist for a triggered run: the RunNow request.
type JobRunState struct {
	jobs.RunNow
}

func (s *JobRunState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s JobRunState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// JobRunRemote embeds RunNow so every StateType path is a valid RemoteType path
// (see TestRemoteSuperset), plus the run's output-only fields for a faithful view.
type JobRunRemote struct {
	jobs.RunNow

	RunId      int64          `json:"run_id,omitempty"`
	RunName    string         `json:"run_name,omitempty"`
	State      *jobs.RunState `json:"state,omitempty"`
	RunPageUrl string         `json:"run_page_url,omitempty"`
	RunType    jobs.RunType   `json:"run_type,omitempty"`
}

// Custom marshaler needed because embedded RunNow's MarshalJSON would otherwise
// take over and drop the additional fields.
func (s *JobRunRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s JobRunRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceJobRun struct {
	client *databricks.WorkspaceClient
}

func (*ResourceJobRun) New(client *databricks.WorkspaceClient) *ResourceJobRun {
	return &ResourceJobRun{
		client: client,
	}
}

func (*ResourceJobRun) PrepareState(input *resources.JobRun) *JobRunState {
	return &JobRunState{
		RunNow: input.RunNow,
	}
}

// makeJobRunRemote maps the GetRun response into the RunNow-shaped remote: GET
// nests the params under overriding_parameters and returns job_parameters as a
// list, so both are flattened back into RunNow.
func makeJobRunRemote(run *jobs.Run) *JobRunRemote {
	var overriding jobs.RunParameters
	if run.OverridingParameters != nil {
		overriding = *run.OverridingParameters
	}
	var jobParameters map[string]string
	if len(run.JobParameters) > 0 {
		jobParameters = make(map[string]string, len(run.JobParameters))
		for _, p := range run.JobParameters {
			jobParameters[p.Name] = p.Value
		}
	}
	return &JobRunRemote{
		RunNow: jobs.RunNow{
			JobId:             run.JobId,
			JobParameters:     jobParameters,
			DbtCommands:       overriding.DbtCommands,
			JarParams:         overriding.JarParams,
			NotebookParams:    overriding.NotebookParams,
			PipelineParams:    overriding.PipelineParams,
			PythonNamedParams: overriding.PythonNamedParams,
			PythonParams:      overriding.PythonParams,
			SparkSubmitParams: overriding.SparkSubmitParams,
			SqlParams:         overriding.SqlParams,
			// Request-only fields GetRun never reports; listed so exhaustruct
			// flags any new SDK field.
			IdempotencyToken:  "",
			Only:              nil,
			PerformanceTarget: "",
			Queue:             nil,
			ForceSendFields:   nil,
		},
		RunId:      run.RunId,
		RunName:    run.RunName,
		State:      run.State,
		RunPageUrl: workspaceurls.ModernizeJobRunPageURL(run.RunPageUrl),
		RunType:    run.RunType,
	}
}

// DoRead returns the run as GetRun reports it; a 404 lets the planner
// re-trigger. Root ignore_remote_changes suppresses all remote drift, so a run
// is recreated only on a local config change.
func (r *ResourceJobRun) DoRead(ctx context.Context, id string) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}
	// var + field set (not a literal) avoids listing GetRunRequest's other fields.
	var req jobs.GetRunRequest
	req.RunId = runID
	run, err := r.client.Jobs.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}
	return makeJobRunRemote(run), nil
}

// RemapState extracts the embedded RunNow as the state used for diffing.
func (*ResourceJobRun) RemapState(remote *JobRunRemote) *JobRunState {
	return &JobRunState{RunNow: remote.RunNow}
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunRemote, error) {
	// RunNow returns only the new run id, so we return a nil remote and let the
	// framework read it back via DoRead.
	wait, err := r.client.Jobs.RunNow(ctx, config.RunNow)
	if err != nil {
		return "", nil, err
	}
	return strconv.FormatInt(wait.RunId, 10), nil, nil
}

// WaitAfterCreate blocks until the run finishes, so a resource referencing its
// output (e.g. state.result_state) sees a settled run. Only SUCCESS continues the deploy.
func (r *ResourceJobRun) WaitAfterCreate(ctx context.Context, id string, _ *JobRunState) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}

	// A run can take hours, so report progress like `bundle run` does. pageURL
	// outlives the poll so an abandoned wait can still link the run.
	var tracker progress.JobStateTracker
	var pageURL string
	// Polled here rather than through Jobs.WaitGetRunJobTerminatedOrSkipped: a run
	// whose task failed reports the deprecated life_cycle_state as INTERNAL_ERROR
	// (status.state is TERMINATED, termination code RUN_EXECUTION_ERROR), and the
	// SDK waiter halts on it with an error of its own, which hides the task that
	// failed.
	run, err := retries.Poll(ctx, jobRunTimeout, func() (*jobs.Run, *retries.Err) {
		var req jobs.GetRunRequest
		req.RunId = runID
		run, err := r.client.Jobs.GetRun(ctx, req)
		if err != nil {
			return nil, retries.Halt(err)
		}
		pageURL = cmp.Or(pageURL, run.RunPageUrl)
		logRunProgress(ctx, run, &tracker)
		if !runIsTerminal(run.State.LifeCycleState) {
			return nil, retries.Continues(run.State.StateMessage)
		}
		return run, nil
	})
	if err != nil {
		// The wait can end with the run still going (timeout, interrupt), so link
		// the run page: the next deploy triggers no second run, finished or not.
		if ctx.Err() != nil {
			// A cancelled context is reported as a timeout.
			return nil, fmt.Errorf("interrupted while waiting for the run to finish: %w%s", ctx.Err(), runPageLine(pageURL))
		}
		return nil, fmt.Errorf("%w%s", err, runPageLine(pageURL))
	}
	if run.State.ResultState != jobs.RunResultStateSuccess {
		return nil, r.runFailedError(ctx, run)
	}
	return makeJobRunRemote(run), nil
}

// runFailedError reports why the run did not succeed, naming each failed task
// and the error it reported.
func (r *ResourceJobRun) runFailedError(ctx context.Context, run *jobs.Run) error {
	outcome := string(run.State.ResultState)
	if outcome == "" {
		// A skipped run has no result_state; report the lifecycle state.
		outcome = string(run.State.LifeCycleState)
	}
	var msg strings.Builder
	// The framework already prefixes the resource key and the run id.
	fmt.Fprintf(&msg, "run did not succeed: %s", outcome)
	if run.State.StateMessage != "" {
		fmt.Fprintf(&msg, ": %s", run.State.StateMessage)
	}
	for _, task := range lastFailedAttempts(run.Tasks) {
		fmt.Fprintf(&msg, "\ntask %q: %s", task.TaskKey, r.taskError(ctx, task))
	}
	msg.WriteString(runPageLine(run.RunPageUrl))
	return errors.New(msg.String())
}

// lastFailedAttempts returns the failed tasks in the order the run reports them,
// one per task key: a task the Jobs API retried is reported once per attempt, and
// only its last one says how the run ended up.
func lastFailedAttempts(tasks []jobs.RunTask) []jobs.RunTask {
	latest := make(map[string]jobs.RunTask)
	var keys []string
	for _, task := range tasks {
		if !taskFailed(task) {
			continue
		}
		previous, seen := latest[task.TaskKey]
		if !seen {
			keys = append(keys, task.TaskKey)
		}
		if !seen || task.AttemptNumber > previous.AttemptNumber {
			latest[task.TaskKey] = task
		}
	}
	result := make([]jobs.RunTask, 0, len(keys))
	for _, key := range keys {
		result = append(result, latest[key])
	}
	return result
}

// taskFailed reports whether a task is a cause of the run's failure. A task left
// SKIPPED or UPSTREAM_FAILED never ran, so it has no error to report.
func taskFailed(task jobs.RunTask) bool {
	// State is deprecated in favour of Status, so it may be absent.
	if task.State == nil {
		return false
	}
	return task.State.LifeCycleState == jobs.RunLifeCycleStateInternalError ||
		task.State.ResultState == jobs.RunResultStateFailed ||
		task.State.ResultState == jobs.RunResultStateTimedout
}

// taskError returns the message the task reported, from the same place `bundle
// run` reads it. Only reached for a task taskFailed accepted, so State is set.
func (r *ResourceJobRun) taskError(ctx context.Context, task jobs.RunTask) string {
	var reported string
	output, err := r.client.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{RunId: task.RunId})
	if err != nil {
		log.Debugf(ctx, "could not read output of task %s: %v", task.TaskKey, err)
	} else {
		reported = output.Error
	}
	// Not every task type reports an error through GetRunOutput, so fall back to
	// what the run itself says about the task.
	return cmp.Or(reported, task.State.StateMessage, string(task.State.ResultState), string(task.State.LifeCycleState))
}

func runPageLine(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	return "\nrun page: " + workspaceurls.ModernizeJobRunPageURL(rawURL)
}

// logRunProgress logs every state change like `bundle run` does, but reports only
// the run page URL and the run's final state to the user: how many states a run
// passes through varies with how long its compute takes to start, so a deploy's
// output would not be reproducible.
func logRunProgress(ctx context.Context, run *jobs.Run, tracker *progress.JobStateTracker) {
	event, first := tracker.Poll(run)
	if event == nil {
		return
	}
	log.Info(ctx, event.String())
	if first && run.RunPageUrl != "" {
		line := "Run URL: " + workspaceurls.ModernizeJobRunPageURL(run.RunPageUrl)
		log.Info(ctx, line)
		reportRunLine(ctx, run.RunId, line)
	}
	if runIsTerminal(run.State.LifeCycleState) {
		reportRunLine(ctx, run.RunId, event.String())
	}
}

// runIsTerminal reports whether a run is done, i.e. in a state the SDK waiter stops on.
func runIsTerminal(state jobs.RunLifeCycleState) bool {
	return state == jobs.RunLifeCycleStateTerminated ||
		state == jobs.RunLifeCycleStateSkipped ||
		state == jobs.RunLifeCycleStateInternalError
}

// reportRunLine names the run, since resources deploy concurrently onto one stream.
func reportRunLine(ctx context.Context, runID int64, msg string) {
	if cmdio.HasIO(ctx) {
		cmdio.LogString(ctx, fmt.Sprintf("job run %d: %s", runID, msg))
	}
}

// DoUpdate is intentionally not implemented: a run can't be modified in place,
// so any change recreates it (delete + a fresh RunNow).

// DoDelete deletes the run via jobs/runs/delete, on both destroy and the
// recreate path. The API rejects a still-active run, which an interrupted wait
// leaves behind, so cancel it first.
func (r *ResourceJobRun) DoDelete(ctx context.Context, id string, _ *JobRunState) error {
	runID, err := parseRunID(id)
	if err != nil {
		return err
	}
	remote, err := r.DoRead(ctx, id)
	if err != nil {
		return err
	}
	if !runIsTerminal(remote.State.LifeCycleState) {
		err = r.cancelRun(ctx, runID)
		if err != nil {
			return err
		}
	}
	return r.client.Jobs.DeleteRunByRunId(ctx, runID)
}

// cancelRun cancels a run and waits for it to settle. Cancellation is
// asynchronous, so a delete issued right after would still be rejected.
func (r *ResourceJobRun) cancelRun(ctx context.Context, runID int64) error {
	waiter, err := r.client.Jobs.CancelRun(ctx, jobs.CancelRun{RunId: runID})
	if err != nil {
		return fmt.Errorf("cancelling run %d before deleting it: %w", runID, err)
	}
	_, err = waiter.Get()
	if err != nil {
		return fmt.Errorf("waiting for run %d to be cancelled: %w", runID, err)
	}
	return nil
}

func parseRunID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: run id is not integer: %q: %w", id, err)
	}
	return result, nil
}
