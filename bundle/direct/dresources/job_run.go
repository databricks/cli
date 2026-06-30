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

// DoRead returns the run's remote state. GetRun confirms the run still exists
// (it can be deleted out-of-band or age out of run-history retention; a
// not-found lets the planner re-trigger it) and reads back its job_id.
// RemoteType == StateType, so no RemapState is needed.
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
	// We record only the run's identity (job_id). A run is immutable, and this
	// milestone re-triggers solely on local RunNow config changes (every input
	// is recreate_on_changes in resources.yml); reading the run's parameters
	// back would only feed a remote diff we'd have to suppress.
	return &JobRunState{RunNow: jobs.RunNow{
		DbtCommands:       nil,
		IdempotencyToken:  "",
		JarParams:         nil,
		JobId:             run.JobId,
		JobParameters:     nil,
		NotebookParams:    nil,
		Only:              nil,
		PerformanceTarget: "",
		PipelineParams:    nil,
		PythonNamedParams: nil,
		PythonParams:      nil,
		Queue:             nil,
		SparkSubmitParams: nil,
		SqlParams:         nil,
		ForceSendFields:   nil,
	}}, nil
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *JobRunState) (string, *JobRunState, error) {
	// RunNow returns immediately with the new run id; waiting for completion is
	// a later milestone.
	wait, err := r.client.Jobs.RunNow(ctx, config.RunNow)
	if err != nil {
		return "", nil, err
	}
	// RunNow's response carries only the run id and we track no remote-only
	// fields, so we echo the sent config back as remote state (RemoteType == StateType).
	remote := &JobRunState{RunNow: config.RunNow}
	return strconv.FormatInt(wait.RunId, 10), remote, nil
}

// DoUpdate is intentionally not implemented: a run can't be modified in place.
// Every field is recreate_on_changes in resources.yml, so any change recreates
// (delete + a fresh RunNow).

// DoDelete is a no-op this milestone: all changes are modeled as recreate and a
// run is considered deployed once triggered. jobs/runs/delete can't help either:
// it rejects active runs, and since we don't await completion the run is usually
// still active when DoDelete runs (the recreate path calls it before DoCreate).
// Revisit once runs are awaited.
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
