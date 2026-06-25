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

// DoRead returns the run's remote state. We call GetRun both to read back the
// run's job_id and to confirm the run still exists: a run can be deleted
// out-of-band or age out of the workspace's run-history retention, and a
// not-found result lets the planner re-trigger it. RemoteType is the same as
// StateType: we don't track any remote-only fields, so no RemapState is needed.
func (r *ResourceJobRun) DoRead(ctx context.Context, id string) (*JobRunState, error) {
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
	// after launch. In this milestone a run is re-triggered solely by a local
	// change to its own RunNow config (every settable input is marked
	// recreate_on_changes in resources.yml and ignored for remote changes);
	// re-triggering on other signals (every deploy, file or referenced-value
	// changes) is a later milestone. We therefore record only the run's identity
	// (job_id) to confirm it still targets the expected job. Reading the run's
	// overriding parameters back here would only feed a remote diff we then have
	// to suppress, so we don't.
	return &JobRunState{RunNow: jobs.RunNow{JobId: run.JobId}}, nil
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunState, error) {
	// RunNow returns immediately with the new run id; waiting for completion is
	// a later milestone.
	wait, err := r.client.Jobs.RunNow(ctx, config.RunNow)
	if err != nil {
		return "", nil, err
	}
	// RunNow's response carries only the run id. We don't track remote-only
	// fields, so the faithful record of what we created is the config we sent;
	// echo it back as the remote state (RemoteType == StateType).
	remote := &JobRunState{RunNow: config.RunNow}
	return strconv.FormatInt(wait.RunId, 10), remote, nil
}

// DoUpdate is intentionally not implemented: there is no API to modify a run in
// place. Every request field is marked recreate_on_changes in resources.yml, so
// any config change goes through delete + create (a fresh RunNow).

// DoDelete is a no-op in this milestone. The project plan defines all changes as
// "recreate" (no-op delete + a fresh RunNow), with the run considered deployed as
// soon as it is triggered; waiting for completion is the next milestone. We also
// can't usefully call jobs/runs/delete here: it only deletes a non-active run and
// errors on an active one, and because we don't wait for completion the run is
// typically still active when DoDelete runs (including on the recreate path,
// which calls DoDelete before DoCreate). Real deletion can be revisited once runs
// are awaited.
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
