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
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// jobRunWaitTimeout caps how long WaitAfterCreate waits for a terminal state:
// large enough for a legitimate long run, but bounded so a CI deploy can't hang
// forever. Matches `bundle run` (jobRunTimeout in bundle/run/job.go).
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

// RemapState extracts the embedded RunNow as the state used for diffing.
func (*ResourceJobRun) RemapState(remote *JobRunRemote) *JobRunState {
	return &JobRunState{RunNow: remote.RunNow}
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunRemote, error) {
	token, err := idempotencyToken(ctx, config)
	if err != nil {
		return "", nil, err
	}
	// Set the token on a copy so it reaches the API but never lands in state.
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
	// Log the run's UI link on the first poll: the wait can last up to
	// jobRunWaitTimeout, so a silent deploy would look hung. TERMINATED/SKIPPED
	// pass even with a FAILED result_state; only INTERNAL_ERROR fails the deploy.
	logged := false
	run, err := r.client.Jobs.WaitGetRunJobTerminatedOrSkipped(ctx, runID, jobRunWaitTimeout, func(run *jobs.Run) {
		if logged || run.RunPageUrl == "" {
			return
		}
		logged = true
		// The direct-engine unit harness runs without cmdIO, so guard LogString.
		if cmdio.HasIO(ctx) {
			cmdio.LogString(ctx, "Waiting for job run to complete: "+run.RunPageUrl)
		}
	})
	if err != nil {
		return nil, err
	}
	return makeJobRunRemote(run), nil
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

// idempotencyToken hashes the run's identity (resource key) plus its config into
// a stable token. The resource key makes two resources with identical config
// produce distinct tokens (so both run); the config makes a change re-trigger; a
// retry with neither changed reuses the run. Hex SHA-256 (64 chars, Jobs API max).
func idempotencyToken(ctx context.Context, state *JobRunState) (string, error) {
	toHash := *state
	toHash.IdempotencyToken = ""
	canonical, err := json.Marshal(toHash)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(ResourceIdentity(ctx)))
	h.Write([]byte{0}) // separator so key‖config can't collide via a shifted boundary
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil)), nil
}
