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

// DoRead returns a faithful view of the remote run; a 404 lets the planner
// re-trigger. Remote-only diffs never count as drift (root ignore_remote_changes);
// only a local change recreates it. RemoteType == StateType.
func (r *ResourceJobRun) DoRead(ctx context.Context, id string) (*JobRunState, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}
	var req jobs.GetRunRequest
	req.RunId = runID
	run, err := r.client.Jobs.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}
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
	// Mirror the run's job_parameters as GetRun reports them: the job's full
	// resolved set, not the override map we sent, so this never round-trips.
	if len(run.JobParameters) > 0 {
		state.JobParameters = make(map[string]string, len(run.JobParameters))
		for _, p := range run.JobParameters {
			state.JobParameters[p.Name] = p.Value
		}
	}
	return &state, nil
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunState, error) {
	// RunNow returns immediately with the new run id; waiting for completion is
	// a later milestone.
	wait, err := r.client.Jobs.RunNow(ctx, config.RunNow)
	if err != nil {
		return "", nil, err
	}
	// RunNow returns only the run id and we track no remote-only fields, so we
	// echo the sent config back as remote state (RemoteType == StateType).
	remote := &JobRunState{RunNow: config.RunNow}
	return strconv.FormatInt(wait.RunId, 10), remote, nil
}

// DoUpdate is intentionally not implemented: a run can't be modified in place,
// so any change recreates it (delete + a fresh RunNow).

// DoDelete deletes the run via jobs/runs/delete, on both destroy and the
// recreate path. The API rejects a still-active run; this milestone doesn't
// await completion, so that error surfaces to the user.
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
