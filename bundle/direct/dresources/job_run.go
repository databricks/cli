package dresources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// JobRunState is what we persist for a triggered run: the RunNow request plus a
// snapshot of the targeted job's settings. The snapshot is what lets a change to
// the job re-trigger the run (RunNow only carries the stable job_id). A custom
// marshaler is required because the embedded jobs.RunNow has its own MarshalJSON
// which would otherwise take over and drop JobSettings.
type JobRunState struct {
	jobs.RunNow

	JobSettings *jobs.JobSettings `json:"job_settings,omitempty"`
}

func (s *JobRunState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s JobRunState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// JobRunRemote is the return type for DoRead. It embeds JobRunState so that every
// path in StateType is also a valid path in RemoteType (required by the
// framework). DoRead fills the embedded state from the GetRun response, mapping
// the run's job-level and overriding parameters back into the RunNow shape so the
// framework can diff them against the desired config. RunId is the remote
// identity, kept here for the detailed plan.
type JobRunRemote struct {
	JobRunState

	RunId int64 `json:"run_id,omitempty"`
}

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
		RunNow:      input.RunNow,
		JobSettings: input.JobSettings,
	}
}

func (*ResourceJobRun) RemapState(remote *JobRunRemote) *JobRunState {
	return &remote.JobRunState
}

func (r *ResourceJobRun) DoRead(ctx context.Context, id string) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}
	run, err := r.client.Jobs.GetRun(ctx, jobs.GetRunRequest{
		RunId:                 runID,
		IncludeHistory:        false,
		IncludeResolvedValues: false,
		PageToken:             "",
		ForceSendFields:       nil,
	})
	if err != nil {
		return nil, err
	}
	// Record the remote state as what we actually created: the fields GetRun
	// echoes back faithfully, namely the run's job_id and the overriding
	// parameters it was launched with. This is what the out-of-band diff compares
	// against the state saved by DoCreate.
	//
	// We deliberately do NOT map run.JobParameters here: GetRun resolves it to the
	// job's full parameter set (including defaults the run never overrode), which
	// is not what we created. Mapping it would only feed a diff we then have to
	// suppress, so job_parameters (along with the request-only fields and the
	// synthetic job_settings snapshot, none of which GetRun returns) is handled
	// via ignore_remote_changes in resources.yml.
	var state JobRunState
	state.JobId = run.JobId
	if p := run.OverridingParameters; p != nil {
		state.DbtCommands = p.DbtCommands
		state.JarParams = p.JarParams
		state.NotebookParams = p.NotebookParams
		state.PipelineParams = p.PipelineParams
		state.PythonNamedParams = p.PythonNamedParams
		state.PythonParams = p.PythonParams
		state.SparkSubmitParams = p.SparkSubmitParams
		state.SqlParams = p.SqlParams
	}
	return &JobRunRemote{JobRunState: state, RunId: run.RunId}, nil
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunRemote, error) {
	// Only the RunNow request is sent to the backend; JobSettings is a local-only
	// snapshot used to detect job changes, not part of the run-now payload.
	// RunNow returns immediately with the new run id; waiting for completion is
	// a later milestone.
	wait, err := r.client.Jobs.RunNow(ctx, config.RunNow)
	if err != nil {
		return "", nil, err
	}
	return strconv.FormatInt(wait.RunId, 10), nil, nil
}

// DoUpdate is intentionally not implemented: there is no API to modify a run in
// place. Every request field is marked recreate_on_changes in resources.yml, so
// any config change goes through delete + create (a fresh RunNow).

// DoDelete is a no-op: a triggered run cannot be "undeployed". On recreate the
// framework calls this before DoCreate, so a no-op delete followed by RunNow
// re-triggers the run, which is the intended behavior.
func (r *ResourceJobRun) DoDelete(ctx context.Context, id string, _ *JobRunState) error {
	return nil
}

func parseRunID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: run id is not integer: %q: %w", id, err)
	}
	return result, nil
}
