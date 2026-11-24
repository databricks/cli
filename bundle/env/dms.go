package env

import "context"

// DeploymentHistoryVariable enables recording without setting the config field.
// Exists for CLI acceptance tests so the whole bundle suite runs with recording on.
// Deliberately undocumented.
const DeploymentHistoryVariable = "DATABRICKS_BUNDLE_DEPLOYMENT_HISTORY"

// RecordsDeploymentHistory reports whether recording is on, from config or env var. Single predicate
// for all recording code paths, keeping them in sync. Explicit config takes priority over the env var.
// The env var only counts as "true": anything
// else - a typo like "TRUE" or "yes" - leaves recording off rather than silently enabling it.
func RecordsDeploymentHistory(ctx context.Context, configured *bool) bool {
	if configured != nil {
		return *configured
	}
	value, _ := get(ctx, []string{DeploymentHistoryVariable})
	return value == "true"
}

// AutoDeployAttemptVariable names the auto-deploy attempt a deploy runs for. The auto-deploy runner
// sets it, and the deploy records it on the deployment version it creates.
const AutoDeployAttemptVariable = "DATABRICKS_AUTO_DEPLOY_ATTEMPT"

// AutoDeployAttempt returns the auto-deploy attempt to record on the deployment version.
func AutoDeployAttempt(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		AutoDeployAttemptVariable,
	})
}
