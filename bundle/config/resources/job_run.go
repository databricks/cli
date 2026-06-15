package resources

import (
	"context"
	"net/url"
	"strconv"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// JobRun is the bundle config for a triggered job run. The run is described by
// the same fields as the Jobs RunNow API request, so we embed jobs.RunNow
// directly instead of re-declaring them.
type JobRun struct {
	BaseResource
	jobs.RunNow
}

func (r *JobRun) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, r)
}

func (r JobRun) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(r)
}

// Exists reports whether the run identified by id (a numeric run id) still
// exists in the workspace. A run is the unit of existence here: once RunNow has
// been called, the run is retrievable via GetRun for as long as the workspace
// retains its history.
func (r *JobRun) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	runID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return false, err
	}
	_, err = w.Jobs.GetRun(ctx, jobs.GetRunRequest{
		RunId: runID,
	})
	if err != nil {
		log.Debugf(ctx, "job run %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (r *JobRun) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "job_run",
		PluralName:    "job_runs",
		SingularTitle: "Job Run",
		PluralTitle:   "Job Runs",
	}
}

// GetName returns the in-product name. A run has no user-assigned name, so this
// is empty.
func (r *JobRun) GetName() string {
	return ""
}

func (r *JobRun) GetURL() string {
	return r.URL
}

// InitializeURL is a no-op for now: surfacing a stable run URL is deferred to a
// later milestone.
func (r *JobRun) InitializeURL(_ url.URL) {
}
