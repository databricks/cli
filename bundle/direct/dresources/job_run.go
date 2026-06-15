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

// JobRunRemote is the return type for DoRead. It embeds jobs.RunNow so that
// every field of the state type (jobs.RunNow) is also a valid path in the
// remote type, as required by the framework. GetRun does not echo back the
// RunNow request, so the embedded RunNow is left zero on read; the actual
// remote identity lives in RunId. Drift on the embedded request fields is
// suppressed via ignore_remote_changes in resources.yml.
type JobRunRemote struct {
	jobs.RunNow

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

func (*ResourceJobRun) PrepareState(input *resources.JobRun) *jobs.RunNow {
	return &input.RunNow
}

func (*ResourceJobRun) RemapState(remote *JobRunRemote) *jobs.RunNow {
	return &remote.RunNow
}

func (r *ResourceJobRun) DoRead(ctx context.Context, id string) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}
	run, err := r.client.Jobs.GetRun(ctx, jobs.GetRunRequest{
		RunId: runID,
	})
	if err != nil {
		return nil, err
	}
	return &JobRunRemote{RunId: run.RunId}, nil
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, config *jobs.RunNow) (string, *JobRunRemote, error) {
	// RunNow returns immediately with the new run id; waiting for completion is
	// a later milestone.
	wait, err := r.client.Jobs.RunNow(ctx, *config)
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
func (r *ResourceJobRun) DoDelete(ctx context.Context, id string, _ *jobs.RunNow) error {
	return nil
}

func parseRunID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: run id is not integer: %q: %w", id, err)
	}
	return result, nil
}
