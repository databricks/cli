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
// fields as the Jobs RunNow request (embedded). It re-triggers when its own
// config changes; the job it targets is pinned by a stable job_id.
type JobRun struct {
	BaseResource
	jobs.RunNow

	// RerunToken feeds the computed idempotency_token, so a new value triggers a
	// fresh run: bump it to re-run identical config or retry a failed run. The
	// Jobs API keeps each token for ~60 days, so reverting to an earlier value
	// within that window rejoins the run it first triggered.
	RerunToken string `json:"rerun_token,omitempty"`

	// WaitForCompletion blocks the deploy until the run finishes and fails it on a
	// non-success result. Defaults to true; false returns once the run starts.
	WaitForCompletion *bool `json:"wait_for_completion,omitempty"`

	// Timeout bounds that wait, as a Go duration string. Defaults to 24h.
	Timeout string `json:"timeout,omitempty"`

	// ResolvedJobID is the run's job_id as loaded from state, used to build the run
	// URL. It is kept apart from RunNow.JobId so state loading preserves that
	// field's ${resources.jobs.*.id} reference and the plan dependency it carries.
	ResolvedJobID int64 `json:"resolved_job_id,omitempty" bundle:"internal"`
}

func (r *JobRun) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, r)
}

func (r JobRun) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(r)
}

// Exists reports whether the run still exists, for as long as the workspace
// retains its run history.
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

// GetName returns an empty string: a run is identified by its id.
func (r *JobRun) GetName() string {
	return ""
}

func (r *JobRun) GetURL() string {
	return r.URL
}

// InitializeURL sets the run's workspace URL, taking the job id from
// RunNow.JobId once resolved (deploy) or from ResolvedJobID (read-only
// commands). The URL stays empty until both ids are known.
func (r *JobRun) InitializeURL(baseURL url.URL) {
	jobID := r.JobId
	if jobID == 0 {
		jobID = r.ResolvedJobID
	}
	if r.ID == "" || jobID == 0 {
		return
	}
	r.URL = workspaceurls.JobRunURL(baseURL, strconv.FormatInt(jobID, 10), r.ID)
}
