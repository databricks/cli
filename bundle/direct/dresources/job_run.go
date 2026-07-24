package dresources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/run/progress"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// jobRunWaitTimeout bounds the wait so a deploy can't hang forever. Matches
// `bundle run` (jobRunTimeout in bundle/run/job.go).
const jobRunWaitTimeout = 24 * time.Hour

// JobRunState is what we persist for a triggered run: the RunNow request.
type JobRunState struct {
	jobs.RunNow

	// RerunToken folds into the idempotency token, so changing it recreates the
	// run. Bundle-only, never sent to the API.
	RerunToken string `json:"rerun_token,omitempty"`

	// Wait controls whether the deploy blocks on the run finishing (default
	// true). Bundle-only; excluded from the token and from diffing so toggling it
	// never re-triggers the run.
	Wait *bool `json:"wait,omitempty"`
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

	// RerunToken and Wait are bundle-only; they exist here only to keep RemoteType
	// a superset of StateType (TestRemoteSuperset). GetRun never returns them, so
	// makeJobRunRemote zeroes them and root ignore_remote_changes hides the drift.
	RerunToken string `json:"rerun_token,omitempty"`
	Wait       *bool  `json:"wait,omitempty"`

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
		RunNow:     input.RunNow,
		RerunToken: input.RerunToken,
		Wait:       input.Wait,
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
		// Bundle-only fields GetRun never reports; see their doc comments.
		RerunToken: "",
		Wait:       nil,
		RunId:      run.RunId,
		RunName:    run.RunName,
		State:      run.State,
		RunPageUrl: run.RunPageUrl,
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

// RemapState extracts the embedded RunNow as the state used for diffing. The
// bundle-only fields are always empty in remote; root ignore_remote_changes
// suppresses the resulting drift against the user's values in state.
func (*ResourceJobRun) RemapState(remote *JobRunRemote) *JobRunState {
	return &JobRunState{RunNow: remote.RunNow, RerunToken: remote.RerunToken, Wait: remote.Wait}
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunRemote, error) {
	token, err := idempotencyToken(ctx, config)
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

	// wait defaults to true; wait: false triggers the run and returns without
	// gating the deploy on its outcome (no remote, so dependents see it mid-run).
	if config.Wait != nil && !*config.Wait {
		return id, nil, nil
	}

	// Wait here, not in WaitAfterCreate: state persists only after DoCreate
	// returns, so an interrupted wait saves nothing and re-triggers RunNow, which
	// the idempotency_token rejoins to the same run. Waiting after the save would
	// let the planner skip the wait on resume.
	remote, err := r.waitForRun(ctx, triggered.RunId)
	if err != nil {
		return "", nil, err
	}
	return id, remote, nil
}

// waitForRun blocks until the run reaches a terminal state and returns its
// remote view. Only SUCCESS completes the deploy; any other terminal outcome is
// an error.
func (r *ResourceJobRun) waitForRun(ctx context.Context, runID int64) (*JobRunRemote, error) {
	// The wait can last up to jobRunWaitTimeout, so surface run progress the same
	// way `bundle run` does (the run page URL, then each state change) instead of
	// a silent deploy that looks hung. prevState carries across poll callbacks.
	var prevState *jobs.RunState
	run, err := r.client.Jobs.WaitGetRunJobTerminatedOrSkipped(ctx, runID, jobRunWaitTimeout, func(run *jobs.Run) {
		prevState = logRunProgress(ctx, run, prevState)
	})
	if err != nil {
		return nil, err
	}
	// Every non-SUCCESS terminal outcome (FAILED, TIMEDOUT, CANCELED,
	// SUCCESS_WITH_FAILURES, SKIPPED, ...) fails the deploy; the waiter already
	// errored on INTERNAL_ERROR and timeout above.
	if run.State.ResultState != jobs.RunResultStateSuccess {
		outcome := string(run.State.ResultState)
		if outcome == "" {
			// A skipped run has no result_state; report the lifecycle state.
			outcome = string(run.State.LifeCycleState)
		}
		// The idempotency token dedupes a plain redeploy back onto this same
		// terminal run, so point at rerun_token as the way to start a fresh one.
		if run.State.StateMessage != "" {
			return nil, fmt.Errorf("job run %d did not succeed: %s: %s; bump rerun_token to start a fresh run", runID, outcome, run.State.StateMessage)
		}
		return nil, fmt.Errorf("job run %d did not succeed: %s; bump rerun_token to start a fresh run", runID, outcome)
	}
	return makeJobRunRemote(run), nil
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
		msg = progress.NewJobRunUrlEvent(run.RunPageUrl).String() + msg
	}
	cmdio.LogString(ctx, msg)
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

// idempotencyToken hashes the resource key, an optional prior run id, and the
// config into a stable token. The key keeps identical configs distinct, a config
// change (including rerun_token) re-triggers, and an unchanged retry reuses the
// run. The prior id (set only when the previous run vanished) rotates the token
// off the deleted run's tombstoned one. Hex SHA-256 (64 chars, Jobs API max).
//
// The hash covers the SDK RunNow JSON, so an SDK field add/rename changes the
// token. That only matters for a create retried across such an upgrade (no-op
// redeploys compare state, not tokens): it would start a fresh run instead of
// deduping onto the interrupted one.
func idempotencyToken(ctx context.Context, state *JobRunState) (string, error) {
	toHash := *state
	toHash.IdempotencyToken = ""
	// wait is a deploy-time toggle, not run identity: exclude it so flipping it
	// never changes the token.
	toHash.Wait = nil
	canonical, err := json.Marshal(toHash)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(ResourceIdentity(ctx)))
	h.Write([]byte{0}) // separator so key‖config can't collide via a shifted boundary
	// Rotate the token when re-creating a vanished run, so RunNow starts fresh
	// instead of hitting the deleted run's tombstoned token.
	if priorID := PriorResourceID(ctx); priorID != "" {
		h.Write([]byte(priorID))
		h.Write([]byte{0})
	}
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil)), nil
}
