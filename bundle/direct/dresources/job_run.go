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

// JobRunState is what we persist for a triggered run: the RunNow request that
// launched it.
type JobRunState struct {
	jobs.RunNow
}

func (s *JobRunState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s JobRunState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// JobRunRemote is the return type for DoRead. It embeds JobRunState so that every
// path in StateType is also a valid path in RemoteType (required by the
// framework). DoRead records only the run's identity; RunId is the remote
// identity, kept here for the detailed plan. The explicit marshaler is required
// because the embedded jobs.RunNow has its own MarshalJSON that would otherwise
// take over and drop RunId.
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
		RunNow: input.RunNow,
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
	// A run is immutable and fire-once: nothing about it changes on the backend
	// after launch, so the run must only be re-triggered when its own RunNow
	// config changes, never from remote drift. We therefore record only the run's
	// identity (job_id) to confirm it still targets the expected job; every
	// settable input is ignored for remote changes in resources.yml and
	// re-triggered solely by a local change via recreate_on_changes. Reading the
	// run's overriding parameters back here would only feed a remote diff we then
	// have to suppress, so we don't.
	var state JobRunState
	state.JobId = run.JobId
	return &JobRunRemote{JobRunState: state, RunId: run.RunId}, nil
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunRemote, error) {
	// RunNow returns immediately with the new run id; waiting for completion is
	// a later milestone.
	wait, err := r.client.Jobs.RunNow(ctx, config.RunNow)
	if err != nil {
		return "", nil, err
	}
	// RunNow's response carries only the run id, so we reconstruct the remote
	// state from the request we just sent (the faithful record of what we
	// created) plus the new run id.
	remote := &JobRunRemote{JobRunState: *config, RunId: wait.RunId}
	return strconv.FormatInt(wait.RunId, 10), remote, nil
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
