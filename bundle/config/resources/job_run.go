package resources

import (
	"context"
	"net/url"
	"strconv"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// JobRun is the bundle config for a triggered job run, described by the same
// fields as the Jobs RunNow request (embedded). It re-triggers only when its own
// config changes, not when the targeted job (stable job_id) changes.
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

// Exists reports whether the run with the given numeric id still exists, for as
// long as the workspace retains its run history.
func (r *JobRun) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	runID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return false, err
	}
	_, err = w.Jobs.GetRun(ctx, jobs.GetRunRequest{
		RunId: runID,
	})
	if err != nil {
		log.Debugf(ctx, "job run %s does not exist: %v", id, err)
		if apierr.IsMissing(err) {
			return false, nil
		}
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

// GetName returns the in-product name, which is empty: a run has no name.
func (r *JobRun) GetName() string {
	return ""
}

func (r *JobRun) GetURL() string {
	return r.URL
}

// InitializeURL sets the run's workspace URL once both addressing IDs are known.
// Before deploy the run id (from state) or job id (an unresolved
// ${resources.jobs.*.id} reference) may be missing, so we skip rather than emit
// a broken jobs/0 URL.
func (r *JobRun) InitializeURL(baseURL url.URL) {
	if r.ID == "" || r.JobId == 0 {
		return
	}
	r.URL = workspaceurls.JobRunURL(baseURL, strconv.FormatInt(r.JobId, 10), r.ID)
}
