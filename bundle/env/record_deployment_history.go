package env

import "context"

// recordDeploymentHistoryVariable names the environment variable that opts the
// bundle into recording deployment history. It is the environment-variable
// equivalent of setting experimental.record_deployment_history in the bundle
// configuration.
const recordDeploymentHistoryVariable = "DATABRICKS_BUNDLE_RECORD_DEPLOYMENT_HISTORY"

// RecordDeploymentHistory returns the environment variable that opts the bundle
// into recording deployment history.
func RecordDeploymentHistory(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		recordDeploymentHistoryVariable,
	})
}
