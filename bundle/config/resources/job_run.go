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

// JobRunLifecycle extends Lifecycle with run-fire triggers.
type JobRunLifecycle struct {
	Lifecycle

	// Triggers that cause the run to re-fire (in addition to config changes).
	Triggers []JobRunTrigger `json:"triggers,omitempty"`

	// Resolved fingerprint for the planner; not user config.
	TriggersState *JobRunTriggersState `json:"triggers_state,omitempty" bundle:"internal"`
}

// JobRunTrigger is one lifecycle.triggers entry.
type JobRunTrigger struct {
	OnBundleDeploy *bool   `json:"on_bundle_deploy,omitempty"`
	OnFileChange   *string `json:"on_file_change,omitempty"` // path or glob relative to the defining YAML file; must resolve under the sync root
}

// JobRunTriggersState is the resolved fingerprint of lifecycle.triggers.
type JobRunTriggersState struct {
	OnBundleDeploy string            `json:"on_bundle_deploy,omitempty"`
	OnFileChange   map[string]string `json:"on_file_change,omitempty"`
}

// IsEmpty reports whether no trigger is armed. An empty state is left off the
// job run entirely, so this has to cover every field above: a new fingerprint
// added without extending it would be dropped instead of persisted.
func (s JobRunTriggersState) IsEmpty() bool {
	return s.OnBundleDeploy == "" && len(s.OnFileChange) == 0
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
