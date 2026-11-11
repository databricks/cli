package configsync

import (
	"cmp"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// The plan skips a job's run_as keys that the config lacks as backend defaults.
// run_as holds a single principal key, so when the remote switches kind (for
// example user_name to service_principal_name) the new key is the edit itself,
// and dropping it while removing the old key leaves "run_as: {}".
const (
	jobRunAsPattern      = "resources.jobs.*.run_as"
	jobRunAsChildPattern = "resources.jobs.*.run_as.*"
)

// keepSkippedRunAsChange reports whether a run_as change the plan skipped as a
// backend default should still be considered for syncing.
func keepSkippedRunAsChange(path string, cd *deployplan.ChangeDesc) bool {
	return cd.Reason == deployplan.ReasonBackendDefault &&
		(matchPattern(jobRunAsPattern, path) || matchPattern(jobRunAsChildPattern, path))
}

// skipJobRunAs reports whether a change to the run_as object as a whole must
// not be synced. The plan diffs a run_as set on both sides key by key, so a
// change to the whole object means one side has none.
func skipJobRunAs(path string, cd *deployplan.ChangeDesc, remoteState any) bool {
	if !matchPattern(jobRunAsPattern, path) {
		return false
	}
	// A removed remote run_as is not synced, as before: the Jobs UI always
	// submits a principal, and removal is unverified on a real workspace.
	if cd.Remote == nil {
		return true
	}
	// The config has no run_as. Skip one unchanged since the last deploy.
	if structdiff.IsEqual(cd.Old, cd.Remote) {
		return true
	}
	return isCreatorRunAs(cd.Remote.(jobs.JobRunAs), remoteState.(*dresources.JobRemote))
}

// isCreatorRunAs reports whether run_as names the job's creator. Jobs updated by
// the terraform engine before it was removed carry an explicit run_as set to the
// job's identity, usually its creator, which is a default rather than an edit:
// https://github.com/databricks/terraform-provider-databricks/blob/76f95e4/jobs/resource_job.go#L844-L857
func isCreatorRunAs(runAs jobs.JobRunAs, job *dresources.JobRemote) bool {
	return cmp.Or(runAs.UserName, runAs.ServicePrincipalName, runAs.GroupName) == job.CreatorUserName
}
