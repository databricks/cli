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

// defaultJobRunTimeout bounds the wait when the config sets no timeout. Matches
// `bundle run` (jobRunTimeout in bundle/run/job.go).
const defaultJobRunTimeout = 24 * time.Hour

// JobRunState is what we persist for a triggered run: the RunNow request.
type JobRunState struct {
	jobs.RunNow

	// RerunToken folds into the idempotency token, so changing it recreates the
	// run. Bundle-only, never sent to the API.
	RerunToken string `json:"rerun_token,omitempty"`

	// WaitForCompletion controls whether the deploy blocks on the run finishing
	// (default true). Bundle-only; excluded from the token and from diffing so
	// toggling it never re-triggers the run.
	WaitForCompletion *bool `json:"wait_for_completion,omitempty"`

	// Timeout bounds that wait, as a Go duration (default 24h). Bundle-only, and
	// excluded from the token and from diffing like WaitForCompletion.
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

	// The bundle-only state fields. The framework has no notion of a state field
	// without a remote counterpart, so they are declared here to satisfy
	// TestRemoteSuperset; GetRun never returns them, so makeJobRunRemote zeroes
	// them and root ignore_remote_changes hides the drift.
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
			// Request-only fields GetRun never reports; listed so exhaustruct
			// flags any new SDK field.
			IdempotencyToken:  "",
			Only:              nil,
			PerformanceTarget: "",
			Queue:             nil,
			ForceSendFields:   nil,
		},
		// Bundle-only fields GetRun never reports; see their doc comments.
		RerunToken:        "",
		WaitForCompletion: nil,
		Timeout:           "",
		RunId:             run.RunId,
		RunName:           run.RunName,
		State:             run.State,
		// Published for reference by other resources, so rewrite the legacy URL
		// the API returns into the form that resolves for non-admins.
		RunPageUrl: workspaceurls.JobRunPageURL(ctx, run.RunPageUrl),
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
	return makeJobRunRemote(ctx, run), nil
}

// RemapState extracts the embedded RunNow as the state used for diffing. The
// bundle-only fields are always empty in remote; root ignore_remote_changes
// suppresses the resulting drift against the user's values in state.
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

	// Set the token on a copy so it reaches the API but never lands in state.
	req := config.RunNow
	req.IdempotencyToken = token

	triggered, err := r.client.Jobs.RunNow(ctx, req)
	if err != nil {
		return "", nil, err
	}
	id := strconv.FormatInt(triggered.RunId, 10)

	// wait_for_completion defaults to true; false triggers the run and returns
	// without gating the deploy on its outcome (no remote, so dependents that
	// read output fields see the run mid-flight).
	if config.WaitForCompletion != nil && !*config.WaitForCompletion {
		return id, nil, nil
	}

	// Wait here, not in WaitAfterCreate: state persists only after DoCreate
	// returns, so an interrupted wait saves nothing and re-triggers RunNow, which
	// the idempotency_token rejoins to the same run. Waiting after the save would
	// let the planner skip the wait on resume.
	remote, err := r.waitForRun(ctx, triggered.RunId, timeout)
	if err != nil {
		return "", nil, err
	}
	return id, remote, nil
}

// runTimeout returns how long DoCreate waits for the run. The validator rejects
// an unparseable value up front, so this only fires on a state written by hand.
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
	// The wait can last hours, so surface run progress the same way `bundle run`
	// does (the run page URL, then each state change) instead of a silent deploy
	// that looks hung. prevState carries across poll callbacks.
	var prevState *jobs.RunState
	run, err := r.client.Jobs.WaitGetRunJobTerminatedOrSkipped(ctx, runID, timeout, func(run *jobs.Run) {
		prevState = logRunProgress(ctx, run, prevState)
	})
	if err != nil {
		return nil, err
	}
	// Every non-SUCCESS terminal outcome (FAILED, TIMEDOUT, CANCELED,
	// SUCCESS_WITH_FAILURES, SKIPPED, ...) fails the deploy; the waiter already
	// errored on INTERNAL_ERROR and timeout above.
	if run.State.ResultState != jobs.RunResultStateSuccess {
		return nil, r.runFailedError(ctx, run)
	}
	return makeJobRunRemote(ctx, run), nil
}

// runFailedError reports why the run did not succeed, naming each failed task so
// the cause is in the error rather than only on the run page.
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
	// The idempotency token dedupes a plain redeploy back onto this same terminal
	// run -- even once the underlying job is fixed -- so name the way out.
	msg.WriteString("\nset rerun_token to a new value to trigger a fresh run")
	return errors.New(msg.String())
}

// taskError returns what the task reported. Only the message is included, not
// the stack trace: the run page link covers the full detail.
func (r *ResourceJobRun) taskError(ctx context.Context, task jobs.RunTask) string {
	output, err := r.client.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{RunId: task.RunId})
	if err != nil {
		log.Debugf(ctx, "could not read output of task %s: %v", task.TaskKey, err)
		return string(task.State.LifeCycleState)
	}
	return output.Error
}

// logRunProgress mirrors `bundle run`'s monitor: the run page URL once, then
// each state change. It returns the state to remember for the next poll, and is
// a no-op without cmdIO (the unit harness calls waitForRun directly).
//
// Apply logs concurrent deploys to a shared writer, so a callback's lines go out
// in a single write; identical blocks from parallel runs then stay
// order-independent.
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

// DoUpdate is intentionally not implemented: a run can't be modified in place,
// so any change recreates it (a fresh RunNow; the prior run is left in place).

// DoDelete is a noop: a run is immutable history, so destroy/recreate leave it
// in place. Deleting would tombstone its idempotency_token (see JobsRunNow) and
// break re-running the same config.
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

// idempotencyToken hashes the resource's create identity and its config into a
// stable token. The identity keeps identical configs in different deployments
// and under different resource keys distinct; a config change (including
// rerun_token) re-triggers; an unchanged retry reuses the run. The prior run id
// (set only when the previous run vanished) rotates the token off the deleted
// run's tombstoned one. Hex SHA-256 (64 chars, Jobs API max).
//
// The hash covers the SDK RunNow JSON, so an SDK field add/rename changes the
// token. That only matters for a create retried across such an upgrade (no-op
// redeploys compare state, not tokens): it would start a fresh run instead of
// deduping onto the interrupted one.
func idempotencyToken(ctx context.Context, state *JobRunState) (string, error) {
	identity, ok := GetCreateIdentity(ctx)
	if !ok {
		return "", errors.New("internal error: job_run created without a create identity")
	}

	toHash := *state
	toHash.IdempotencyToken = ""
	// Deploy-time toggles, not run identity: exclude them so flipping either one
	// never changes the token.
	toHash.WaitForCompletion = nil
	toHash.Timeout = ""
	canonical, err := json.Marshal(toHash)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	// NUL separators so adjacent parts can't collide via a shifted boundary.
	h.Write([]byte(identity.Deployment))
	h.Write([]byte{0})
	h.Write([]byte(identity.ResourceKey))
	h.Write([]byte{0})
	h.Write([]byte(identity.PriorID))
	h.Write([]byte{0})
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil)), nil
}
