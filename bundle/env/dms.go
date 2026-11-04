package env

import "context"

// DeploymentHistoryVariable enables recording without setting the config field.
// Exists for CLI acceptance tests so the whole bundle suite runs with recording on.
// Deliberately undocumented.
const DeploymentHistoryVariable = "DATABRICKS_BUNDLE_DEPLOYMENT_HISTORY"

// DevDeploymentHistoryOnlyVariable selects DMS-authoritative state for new deployments with history enabled.
// Used for development and acceptance tests: https://github.com/databricks/cli/pull/7169.
const DevDeploymentHistoryOnlyVariable = "DATABRICKS_BUNDLE_DEV_DEPLOYMENT_HISTORY_ONLY"

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
