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
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// jobRunWaitTimeout bounds how long WaitAfterCreate waits for a run to reach a
// terminal state. Large on purpose so a long but legitimate run (migration,
// training) doesn't fail the deploy, while an unattended (CI) deploy still can't
// hang forever. Matches `bundle run`'s budget (jobRunTimeout in bundle/run/job.go).
const jobRunWaitTimeout = 24 * time.Hour

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

// makeJobRunRemote maps GetRun into the RunNow-shaped remote, flattening
// overriding_parameters and the job_parameters list back into RunNow.
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
		RunPageUrl: run.RunPageUrl,
		RunType:    run.RunType,
	}
}

// DoRead returns the run as GetRun reports it; a 404 lets the planner
// re-trigger. ignore_remote_changes suppresses drift, so a run is recreated
// only on a local config change.
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
	token, err := idempotencyToken(config)
	if err != nil {
		return "", nil, err
	}
	// Copy so the token reaches the API but never lands in state (keeps state
	// token-free and plans clean).
	req := config.RunNow
	req.IdempotencyToken = token

	// RunNow returns only the run id; return a nil remote and let the framework
	// read it back via DoRead.
	wait, err := r.client.Jobs.RunNow(ctx, req)
	if err != nil {
		return "", nil, err
	}
	return strconv.FormatInt(wait.RunId, 10), nil, nil
}

func (r *ResourceJobRun) WaitAfterCreate(ctx context.Context, id string, _ *JobRunState) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}
	logRunPageURL(ctx, r.client, runID)
	// TERMINATED/SKIPPED succeed even with a FAILED/TIMEDOUT/CANCELED
	// result_state (surfaced as state, not a deploy failure); only INTERNAL_ERROR
	// fails the deploy.
	run, err := r.client.Jobs.WaitGetRunJobTerminatedOrSkipped(ctx, runID, jobRunWaitTimeout, nil)
	if err != nil {
		return nil, err
	}
	return makeJobRunRemote(run), nil
}

// logRunPageURL best-effort surfaces the run's UI link before WaitAfterCreate
// blocks. The wait can legitimately last hours (jobRunWaitTimeout), so an
// otherwise-silent deploy would look hung. A failure here is non-fatal: the URL
// is only a progress aid, and the wait below reports any real error.
func logRunPageURL(ctx context.Context, client *databricks.WorkspaceClient, runID int64) {
	var req jobs.GetRunRequest
	req.RunId = runID
	run, err := client.Jobs.GetRun(ctx, req)
	if err != nil {
		log.Debugf(ctx, "job_run: could not fetch run page URL before waiting: %v", err)
		return
	}
	if run.RunPageUrl == "" {
		return
	}
	msg := "Waiting for job run to complete: " + run.RunPageUrl
	// A deploy always has cmdIO installed (root command's PersistentPreRunE), but
	// the direct-engine unit harness calls this without one, so guard the panic.
	if cmdio.HasIO(ctx) {
		cmdio.LogString(ctx, msg)
	} else {
		log.Debugf(ctx, "%s", msg)
	}
}

// DoUpdate is intentionally not implemented: a run can't be modified in place,
// so any change recreates it (a fresh RunNow; the prior run is left in place).

// DoDelete is a noop: a run is an immutable historical record, so destroy and
// recreate leave it in place. Dedup relies on the run id in state and the
// idempotency_token, so deleting the run would only tombstone its token and
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

// idempotencyToken derives a stable token from the desired state so a retried
// run-now dedupes to the existing run instead of starting a duplicate. Hashing
// the whole JobRunState means future state fields (e.g. milestone-3 triggers)
// join dedup automatically. Hex SHA-256 (64 chars, the Jobs API max), computed
// with idempotency_token cleared.
func idempotencyToken(state *JobRunState) (string, error) {
	toHash := *state
	toHash.IdempotencyToken = ""
	canonical, err := json.Marshal(toHash)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}
