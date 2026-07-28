package dresources

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// jobRunTimeout bounds the wait for a run to finish, matching `bundle run`
// (jobRunTimeout in bundle/run/job.go).
const jobRunTimeout = 24 * time.Hour

// JobRunState is what we persist for a triggered run: the RunNow request.
type JobRunState struct {
	jobs.RunNow

	// RerunToken folds into the idempotency token, so changing it recreates the
	// run. Bundle-only, never sent to the API.
	RerunToken string `json:"rerun_token,omitempty"`
}

func (s *JobRunState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s JobRunState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// JobRunRemote embeds RunNow so every StateType path is a valid RemoteType path
// (see TestRemoteSuperset), plus the run's output-only fields.
type JobRunRemote struct {
	jobs.RunNow

	// RerunToken is bundle-only. GetRun never reports it, so it stays empty here
	// and root ignore_remote_changes hides the drift against state.
	RerunToken string `json:"rerun_token,omitempty"`

	RunId   int64          `json:"run_id,omitempty"`
	RunName string         `json:"run_name,omitempty"`
	State   *jobs.RunState `json:"state,omitempty"`
	// Normalized to the path form that resolves for non-admins, so whatever
	// references it gets a usable link. See workspaceurls.JobRunPageURL.
	RunPageUrl string       `json:"run_page_url,omitempty"`
	RunType    jobs.RunType `json:"run_type,omitempty"`
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
		RunNow:     input.RunNow,
		RerunToken: input.RerunToken,
	}
}

// makeJobRunRemote maps the GetRun response into the RunNow-shaped remote: GET
// nests the params under overriding_parameters and returns job_parameters as a
// list, so both are flattened back into RunNow.
func makeJobRunRemote(ctx context.Context, run *jobs.Run) *JobRunRemote {
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
			// Request-only fields, listed so exhaustruct flags any new SDK field.
			IdempotencyToken:  "",
			Only:              nil,
			PerformanceTarget: "",
			Queue:             nil,
			ForceSendFields:   nil,
		},
		// Bundle-only; see its doc comment.
		RerunToken: "",
		RunId:      run.RunId,
		RunName:    run.RunName,
		State:      run.State,
		RunPageUrl: workspaceurls.JobRunPageURL(ctx, run.RunPageUrl),
		RunType:    run.RunType,
	}
}

// DoRead returns the run as GetRun reports it; a 404 lets the planner
// re-trigger. Root ignore_remote_changes suppresses remote drift, so a local
// config change is what recreates a run.
func (r *ResourceJobRun) DoRead(ctx context.Context, id string) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}
	// Assigned through a var to satisfy exhaustruct without listing every field.
	var req jobs.GetRunRequest
	req.RunId = runID
	run, err := r.client.Jobs.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}
	return makeJobRunRemote(ctx, run), nil
}

// RemapState maps remote into the state shape for diffing. RerunToken is empty
// in remote; root ignore_remote_changes hides that drift.
func (*ResourceJobRun) RemapState(remote *JobRunRemote) *JobRunState {
	return &JobRunState{
		RunNow:     remote.RunNow,
		RerunToken: remote.RerunToken,
	}
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunRemote, error) {
	// The framework always attaches an identity on create, so its absence is a
	// wiring bug rather than a user error.
	identity, ok := GetCreateIdentity(ctx)
	if !ok {
		return "", nil, errors.New("internal error: job_run created without a create identity")
	}
	token, err := idempotencyToken(identity, config)
	if err != nil {
		return "", nil, err
	}

	// Copy so the token reaches the API and stays out of state.
	req := config.RunNow
	req.IdempotencyToken = token
	triggered, err := r.client.Jobs.RunNow(ctx, req)
	if err != nil {
		return "", nil, fmt.Errorf("triggering a run of job %d: %w", req.JobId, err)
	}

	// Waiting here rather than in WaitAfterCreate keeps the retry idempotent:
	// state persists only once DoCreate returns, so an interrupted wait re-issues
	// RunNow and the token rejoins this same run.
	remote, err := r.waitForRun(ctx, identity.ResourceKey, triggered.RunId)
	if err != nil {
		return "", nil, err
	}
	return strconv.FormatInt(triggered.RunId, 10), remote, nil
}

// waitForRun blocks until the run reaches a terminal state and returns its
// remote view. Only SUCCESS completes the deploy; any other outcome is an error.
func (r *ResourceJobRun) waitForRun(ctx context.Context, resourceKey string, runID int64) (*JobRunRemote, error) {
	// A run can take hours, so report progress like `bundle run` does. prevState
	// suppresses repeats; pageURL survives the callback so an abandoned wait can
	// still link the run.
	var prevState *jobs.RunState
	var pageURL string
	run, err := r.client.Jobs.WaitGetRunJobTerminatedOrSkipped(ctx, runID, jobRunTimeout, func(run *jobs.Run) {
		pageURL = run.RunPageUrl
		prevState = logRunProgress(ctx, resourceKey, run, prevState)
	})
	if err != nil {
		// The run may have hit INTERNAL_ERROR, or we gave up on timeout or
		// interrupt while it kept going. Only the wrapped error tells them apart.
		return nil, fmt.Errorf("waiting for job run %d: %w%s", runID, err, runPageLine(ctx, pageURL))
	}
	// FAILED, TIMEDOUT, CANCELED, SUCCESS_WITH_FAILURES and SKIPPED all fail the
	// deploy; the waiter already errored on INTERNAL_ERROR and on timeout.
	if run.State.ResultState != jobs.RunResultStateSuccess {
		return nil, r.runFailedError(ctx, run)
	}
	return makeJobRunRemote(ctx, run), nil
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
	fmt.Fprintf(&msg, "job run %d did not succeed: %s", run.RunId, outcome)
	if run.State.StateMessage != "" {
		fmt.Fprintf(&msg, ": %s", run.State.StateMessage)
	}
	for _, task := range run.Tasks {
		if taskFailed(task) {
			fmt.Fprintf(&msg, "\ntask %q: %s", task.TaskKey, r.taskError(ctx, task))
		}
	}
	msg.WriteString(runPageLine(ctx, run.RunPageUrl))
	// No state was saved, so a redeploy re-issues run-now with the same token and
	// gets this same terminal run back, even once the job itself is fixed.
	msg.WriteString("\nredeploying reports this same run; set rerun_token to a new value to run the job again")
	return errors.New(msg.String())
}

// taskFailed reports whether a task caused the run to fail rather than being a
// casualty of it. Tasks left SKIPPED or UPSTREAM_FAILED by an earlier failure
// add noise without naming the problem.
func taskFailed(task jobs.RunTask) bool {
	// State is deprecated in favour of Status, so it may be absent.
	if task.State == nil {
		return false
	}
	return task.State.LifeCycleState == jobs.RunLifeCycleStateInternalError ||
		task.State.ResultState == jobs.RunResultStateFailed ||
		task.State.ResultState == jobs.RunResultStateTimedout
}

// taskError returns the message the task reported, from the same place
// `bundle run` reads it. Only called for a task taskFailed accepted, so State is
// set.
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

// runPageLine returns a line linking the run page, or an empty string when the
// URL is unknown.
func runPageLine(ctx context.Context, rawURL string) string {
	if rawURL == "" {
		return ""
	}
	return "\nrun page: " + workspaceurls.JobRunPageURL(ctx, rawURL)
}

// logRunProgress mirrors `bundle run`'s monitor: the run page URL once, then
// each state change. It returns the state to remember for the next poll.
func logRunProgress(ctx context.Context, resourceKey string, run *jobs.Run, prev *jobs.RunState) *jobs.RunState {
	if run.State == nil {
		return prev
	}
	if prev != nil &&
		prev.LifeCycleState == run.State.LifeCycleState &&
		prev.ResultState == run.State.ResultState {
		return prev
	}
	if prev == nil && run.RunPageUrl != "" {
		logRunLine(ctx, resourceKey, "Run URL: "+workspaceurls.JobRunPageURL(ctx, run.RunPageUrl))
	}
	logRunLine(ctx, resourceKey, (&progress.JobProgressEvent{
		Timestamp: time.Now(),
		JobId:     run.JobId,
		RunId:     run.RunId,
		RunName:   run.RunName,
		State:     *run.State,
	}).String())
	return run.State
}

// logRunLine reports one line about a run to the user and the log. Runs deploy
// concurrently onto one stream, so the user-facing copy is tagged with the
// resource key; the log already carries it via log.WithPrefix.
func logRunLine(ctx context.Context, resourceKey, msg string) {
	log.Info(ctx, msg)
	if cmdio.HasIO(ctx) {
		cmdio.LogString(ctx, resourceKey+": "+msg)
	}
}

// DoUpdate is intentionally not implemented: a run is immutable, so any config
// change recreates the resource as a fresh RunNow.

// DoDelete is a noop: a run is immutable history, so destroy and recreate leave
// it in place. That also keeps its idempotency_token usable, which the Jobs API
// reserves for good once a run is deleted (see JobsRunNow).
func (*ResourceJobRun) DoDelete(_ context.Context, _ string, _ *JobRunState) error {
	return nil
}

func parseRunID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: run id is not integer: %q: %w", id, err)
	}
	return result, nil
}

// idempotencyToken hashes the create identity and the run config into a hex
// SHA-256 (64 chars, the Jobs API maximum). It is stable, so a retried create
// dedupes onto the same run; it changes with the config, so an edited run (or a
// bumped rerun_token) starts a new one; and the identity keeps identical configs
// apart across resource keys and deployments.
//
// Hashing the SDK RunNow JSON means an SDK field rename changes the token, which
// only affects a create retried across such an upgrade: it starts a fresh run.
func idempotencyToken(identity CreateIdentity, state *JobRunState) (string, error) {
	toHash := *state
	// Not part of the run's identity: it is what we are computing.
	toHash.IdempotencyToken = ""
	canonical, err := json.Marshal(toHash)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	// NUL separators keep adjacent parts from colliding on a shifted boundary.
	h.Write([]byte(identity.Deployment))
	h.Write([]byte{0})
	h.Write([]byte(identity.ResourceKey))
	h.Write([]byte{0})
	h.Write([]byte(identity.PriorID))
	h.Write([]byte{0})
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil)), nil
}
