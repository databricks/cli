package env

import "context"

// RecordDeploymentHistoryVariable enables recording without setting the config field.
// Exists for CLI acceptance tests so the whole bundle suite runs with recording on.
// Deliberately undocumented.
const RecordDeploymentHistoryVariable = "DATABRICKS_BUNDLE_RECORD_DEPLOYMENT_HISTORY"

// RecordsDeploymentHistory reports whether recording is on, from config or env var. Single predicate
// for all recording code paths, keeping them in sync. The env var only counts as "true": anything
// else - a typo like "TRUE" or "yes" - leaves recording off rather than silently enabling it.
func RecordsDeploymentHistory(ctx context.Context, configured bool) bool {
	if configured {
		return true
	}
	value, _ := get(ctx, []string{RecordDeploymentHistoryVariable})
	return value == "true"
}
