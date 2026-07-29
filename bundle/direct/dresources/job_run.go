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
// output (e.g. state.result_state) sees a settled run. Only SUCCESS lets the
// deploy continue.
func (r *ResourceJobRun) WaitAfterCreate(ctx context.Context, id string, _ *JobRunState) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}

	// A run can take hours, so report progress like `bundle run` does. pageURL
	// outlives the callback so an abandoned wait can still link the run.
	var tracker progress.JobStateTracker
	var pageURL string
	run, err := r.client.Jobs.WaitGetRunJobTerminatedOrSkipped(ctx, runID, jobRunTimeout, func(run *jobs.Run) {
		pageURL = run.RunPageUrl
		logRunProgress(ctx, run, &tracker)
	})
	if err != nil {
		// The wait can end with the run still going (timeout, interrupt), so link
		// the run page.
		return nil, fmt.Errorf("%w%s", err, runPageLine(pageURL))
	}
	// FAILED, TIMEDOUT, CANCELED, SUCCESS_WITH_FAILURES and SKIPPED all fail the
	// deploy; the waiter already errored on INTERNAL_ERROR and on timeout.
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
	for _, task := range run.Tasks {
		if taskFailed(task) {
			fmt.Fprintf(&msg, "\ntask %q: %s", task.TaskKey, r.taskError(ctx, task))
		}
	}
	msg.WriteString(runPageLine(run.RunPageUrl))
	return errors.New(msg.String())
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
// the run page URL and the state the run ends in to the user: how many states a
// run passes through depends on how long its compute takes to start, which would
// make a deploy's output differ between runs of the same bundle.
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

// runIsTerminal reports whether a run is done, i.e. in one of the states the SDK
// waiter stops on.
func runIsTerminal(state jobs.RunLifeCycleState) bool {
	return state == jobs.RunLifeCycleStateTerminated ||
		state == jobs.RunLifeCycleStateSkipped ||
		state == jobs.RunLifeCycleStateInternalError
}

// reportRunLine names the run it describes, since resources deploy concurrently
// onto one output stream.
func reportRunLine(ctx context.Context, runID int64, msg string) {
	if cmdio.HasIO(ctx) {
		cmdio.LogString(ctx, fmt.Sprintf("job run %d: %s", runID, msg))
	}
}

// DoUpdate is intentionally not implemented: a run can't be modified in place,
// so any change recreates it (delete + a fresh RunNow).

// DoDelete deletes the run via jobs/runs/delete, on both destroy and the
// recreate path. The API rejects a still-active run; WaitAfterCreate leaves it
// terminal, so that error only surfaces when a wait was interrupted.
func (r *ResourceJobRun) DoDelete(ctx context.Context, id string, _ *JobRunState) error {
	runID, err := parseRunID(id)
	if err != nil {
		return err
	}
	return r.client.Jobs.DeleteRunByRunId(ctx, runID)
}

func parseRunID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: run id is not integer: %q: %w", id, err)
	}
	return result, nil
}
