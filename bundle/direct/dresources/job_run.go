package dresources

import (
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

// defaultJobRunTimeout is the wait when the config sets no timeout, matching
// `bundle run` (jobRunTimeout in bundle/run/job.go).
const defaultJobRunTimeout = 24 * time.Hour

// JobRunState is what we persist for a triggered run: the RunNow request.
type JobRunState struct {
	jobs.RunNow

	// RerunToken folds into the idempotency token, so changing it recreates the
	// run. Bundle-only, never sent to the API.
	RerunToken string `json:"rerun_token,omitempty"`

	// WaitForCompletion blocks the deploy until the run finishes (default true).
	// Bundle-only, and excluded from the token and from diffing as a deploy-time
	// knob.
	WaitForCompletion *bool `json:"wait_for_completion,omitempty"`

	// Timeout bounds that wait, as a Go duration (default 24h). Bundle-only and
	// excluded like WaitForCompletion.
	Timeout string `json:"timeout,omitempty"`
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

	// Bundle-only state fields, declared here because every state field needs a
	// remote counterpart (TestRemoteSuperset). makeJobRunRemote leaves them empty
	// and root ignore_remote_changes hides the resulting drift.
	RerunToken        string `json:"rerun_token,omitempty"`
	WaitForCompletion *bool  `json:"wait_for_completion,omitempty"`
	Timeout           string `json:"timeout,omitempty"`

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
		RunNow:            input.RunNow,
		RerunToken:        input.RerunToken,
		WaitForCompletion: input.WaitForCompletion,
		Timeout:           input.Timeout,
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
		// Bundle-only fields; see their doc comments.
		RerunToken:        "",
		WaitForCompletion: nil,
		Timeout:           "",
		RunId:             run.RunId,
		RunName:           run.RunName,
		State:             run.State,
		// Normalized to the URL form that resolves for non-admins.
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

// RemapState maps remote into the state shape for diffing. The bundle-only
// fields are empty in remote; root ignore_remote_changes hides that drift.
func (*ResourceJobRun) RemapState(remote *JobRunRemote) *JobRunState {
	return &JobRunState{
		RunNow:            remote.RunNow,
		RerunToken:        remote.RerunToken,
		WaitForCompletion: remote.WaitForCompletion,
		Timeout:           remote.Timeout,
	}
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunRemote, error) {
	token, err := idempotencyToken(ctx, config)
	if err != nil {
		return "", nil, err
	}
	timeout, err := runTimeout(config)
	if err != nil {
		return "", nil, err
	}

	// Copy so the token reaches the API and stays out of state.
	req := config.RunNow
	req.IdempotencyToken = token

	triggered, err := r.client.Jobs.RunNow(ctx, req)
	if err != nil {
		return "", nil, err
	}
	id := strconv.FormatInt(triggered.RunId, 10)

	// wait_for_completion: false returns as soon as the run is triggered, with no
	// remote; the validator rejects references to its state for that reason.
	if config.WaitForCompletion != nil && !*config.WaitForCompletion {
		return id, nil, nil
	}

	// The wait belongs in DoCreate: state persists only once it returns, so an
	// interrupted wait re-triggers RunNow and the idempotency_token rejoins the
	// same run.
	remote, err := r.waitForRun(ctx, triggered.RunId, timeout)
	if err != nil {
		return "", nil, err
	}
	return id, remote, nil
}

// runTimeout is how long DoCreate waits for the run. The validator parses the
// value up front, so an error here means hand-written state.
func runTimeout(config *JobRunState) (time.Duration, error) {
	if config.Timeout == "" {
		return defaultJobRunTimeout, nil
	}
	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q: %w", config.Timeout, err)
	}
	return timeout, nil
}

// waitForRun blocks until the run reaches a terminal state and returns its
// remote view. Only SUCCESS completes the deploy; any other terminal outcome is
// an error.
func (r *ResourceJobRun) waitForRun(ctx context.Context, runID int64, timeout time.Duration) (*JobRunRemote, error) {
	// The wait can last hours, so report progress like `bundle run` does.
	// prevState carries across poll callbacks.
	var prevState *jobs.RunState
	run, err := r.client.Jobs.WaitGetRunJobTerminatedOrSkipped(ctx, runID, timeout, func(run *jobs.Run) {
		prevState = logRunProgress(ctx, run, prevState)
	})
	if err != nil {
		return nil, err
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
		if task.State.LifeCycleState == jobs.RunLifeCycleStateInternalError ||
			task.State.ResultState == jobs.RunResultStateFailed {
			fmt.Fprintf(&msg, "\ntask %q: %s", task.TaskKey, r.taskError(ctx, task))
		}
	}
	if run.RunPageUrl != "" {
		fmt.Fprintf(&msg, "\nrun page: %s", workspaceurls.JobRunPageURL(ctx, run.RunPageUrl))
	}
	// A plain redeploy dedupes back onto this terminal run, even once the job is
	// fixed, so name the way out.
	msg.WriteString("\nset rerun_token to a new value to trigger a fresh run")
	return errors.New(msg.String())
}

// taskError returns the message the task reported. The run page link covers the
// stack trace.
func (r *ResourceJobRun) taskError(ctx context.Context, task jobs.RunTask) string {
	output, err := r.client.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{RunId: task.RunId})
	if err != nil {
		log.Debugf(ctx, "could not read output of task %s: %v", task.TaskKey, err)
		return string(task.State.LifeCycleState)
	}
	return output.Error
}

// logRunProgress mirrors `bundle run`'s monitor: the run page URL once, then
// each state change. It returns the state to remember for the next poll and
// writes through cmdIO when one is attached.
//
// Apply logs concurrent deploys to a shared writer, so each callback emits its
// lines in a single write and parallel runs stay order-independent.
func logRunProgress(ctx context.Context, run *jobs.Run, prev *jobs.RunState) *jobs.RunState {
	if !cmdio.HasIO(ctx) || run.State == nil {
		return prev
	}
	if prev != nil &&
		prev.LifeCycleState == run.State.LifeCycleState &&
		prev.ResultState == run.State.ResultState {
		return prev
	}
	event := &progress.JobProgressEvent{
		Timestamp: time.Now(),
		JobId:     run.JobId,
		RunId:     run.RunId,
		RunName:   run.RunName,
		State:     *run.State,
	}
	msg := event.String()
	if prev == nil && run.RunPageUrl != "" {
		// JobRunUrlEvent.String() already ends in a newline.
		msg = progress.NewJobRunUrlEvent(workspaceurls.JobRunPageURL(ctx, run.RunPageUrl)).String() + msg
	}
	cmdio.LogString(ctx, msg)
	log.Info(ctx, event.String())
	return run.State
}

// A run is immutable: any config change recreates the resource as a fresh
// RunNow and leaves the prior run in place.

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

// idempotencyToken hashes the create identity and the run config into a stable
// hex SHA-256 (64 chars, the Jobs API maximum): a retried create dedupes onto
// the same run, a config change (including rerun_token) triggers a new one, and
// the identity keeps identical configs apart across resource keys and
// deployments.
//
// The hash covers the SDK RunNow JSON, so an SDK field rename changes the token.
// That reaches only a create retried across such an upgrade, which then starts a
// fresh run.
func idempotencyToken(ctx context.Context, state *JobRunState) (string, error) {
	identity, ok := GetCreateIdentity(ctx)
	if !ok {
		return "", errors.New("internal error: job_run created without a create identity")
	}

	toHash := *state
	toHash.IdempotencyToken = ""
	// Deploy-time knobs, excluded so changing them keeps the token stable.
	toHash.WaitForCompletion = nil
	toHash.Timeout = ""
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
