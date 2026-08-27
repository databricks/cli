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
// fields as the Jobs RunNow request (embedded). By default it re-fires when its
// own configuration changes; lifecycle.triggers can add further conditions.
type JobRun struct {
	BaseResource
	jobs.RunNow

	// Lifecycle shadows BaseResource.Lifecycle so job_runs can set triggers.
	Lifecycle *JobRunLifecycle `json:"lifecycle,omitempty"`

	// ResolvedJobID holds the run's job_id loaded from state, used only to build
	// the run URL. Keeping it separate from RunNow.JobId (a ${resources.jobs.*.id}
	// reference) lets state loading preserve that reference and its plan dependency.
	ResolvedJobID int64 `json:"resolved_job_id,omitempty" bundle:"internal"`
}

// HasOnBundleDeploy reports whether any trigger re-fires on every deploy.
func (r *JobRun) HasOnBundleDeploy() bool {
	if r.Lifecycle == nil {
		return false
	}
	for _, t := range r.Lifecycle.Triggers {
		if t.OnBundleDeploy != nil && *t.OnBundleDeploy {
			return true
		}
	}
	return false
}

// HasOnFileChange reports whether any trigger re-fires when matched files change.
func (r *JobRun) HasOnFileChange() bool {
	if r.Lifecycle == nil {
		return false
	}
	for _, t := range r.Lifecycle.Triggers {
		if t.OnFileChange != nil {
			return true
		}
	}
	return false
}

// ArmedTriggerNames returns the names of the trigger fields any entry arms, in
// schema order so that diagnostics naming them are stable.
func (r *JobRun) ArmedTriggerNames() []string {
	if r.Lifecycle == nil {
		return nil
	}
	var onBundleDeploy, onFileChange, onValueChange bool
	for _, t := range r.Lifecycle.Triggers {
		onBundleDeploy = onBundleDeploy || (t.OnBundleDeploy != nil && *t.OnBundleDeploy)
		onFileChange = onFileChange || t.OnFileChange != nil
		onValueChange = onValueChange || t.OnValueChange != nil
	}
	var names []string
	if onBundleDeploy {
		names = append(names, "on_bundle_deploy")
	}
	if onFileChange {
		names = append(names, "on_file_change")
	}
	if onValueChange {
		names = append(names, "on_value_change")
	}
	return names
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

// InitializeURL sets the run's workspace URL. The job id comes from RunNow.JobId
// when resolved (deploy) or ResolvedJobID from state (read-only commands); if
// either id is missing we skip rather than emit a broken jobs/0 URL.
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
