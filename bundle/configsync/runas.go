package configsync

import "github.com/databricks/cli/bundle/deployplan"

// The plan skips a job's run_as keys that the config lacks as backend defaults.
// run_as holds a single principal key, so when the remote switches kind (for
// example user_name to service_principal_name) the new key is the edit itself,
// and dropping it while removing the old key leaves "run_as: {}".
const jobRunAsChildPattern = "resources.jobs.*.run_as.*"

// keepSkippedRunAsChange reports whether a run_as change the plan skipped as a
// backend default should still be considered for syncing.
func keepSkippedRunAsChange(path string, cd *deployplan.ChangeDesc) bool {
	return cd.Reason == deployplan.ReasonBackendDefault && matchPattern(jobRunAsChildPattern, path)
}
