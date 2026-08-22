package env

import "context"

// RecordDeploymentHistoryVariable enables recording without setting the config field.
// Exists for CLI acceptance tests so the whole bundle suite runs with recording on.
// Deliberately undocumented; see validate.ValidateRecordDeploymentHistory.
const RecordDeploymentHistoryVariable = "DATABRICKS_BUNDLE_RECORD_DEPLOYMENT_HISTORY"

// recordDeploymentHistoryEnv reports whether the environment turns on deployment
// history recording. Only "true" turns it on: anything else - including a typo like
// "TRUE" or "yes" - leaves a gated feature off rather than silently enabling it.
func recordDeploymentHistoryEnv(ctx context.Context) bool {
	value, _ := get(ctx, []string{RecordDeploymentHistoryVariable})
	return value == "true"
}

// RecordsDeploymentHistory reports whether recording is on, from config or env var.
// Single predicate for all recording code paths; keeps them in sync.
func RecordsDeploymentHistory(ctx context.Context, configured bool) bool {
	return configured || recordDeploymentHistoryEnv(ctx)
}
