package env

import "context"

// RecordDeploymentHistoryVariable names the environment variable that turns on
// deployment history recording without setting experimental.record_deployment_history
// in the bundle. It exists for the CLI's own acceptance tests, which run the whole
// bundle suite with recording enabled: setting it here beats adding the field to every
// databricks.yml.
//
// Like ForceAllowRecordDeploymentHistoryVariable it is deliberately undocumented; see
// validate.ValidateRecordDeploymentHistory for why the feature is still gated off.
const RecordDeploymentHistoryVariable = "DATABRICKS_BUNDLE_RECORD_DEPLOYMENT_HISTORY"

// recordDeploymentHistoryEnv reports whether the environment turns on deployment
// history recording. Only "true" turns it on: anything else - including a typo like
// "TRUE" or "yes" - leaves a gated feature off rather than silently enabling it.
func recordDeploymentHistoryEnv(ctx context.Context) bool {
	value, _ := get(ctx, []string{RecordDeploymentHistoryVariable})
	return value == "true"
}

// RecordsDeploymentHistory reports whether this deploy records deployment history,
// from either the bundle setting or RecordDeploymentHistoryVariable. It is the single
// predicate the recording code paths branch on, so the env var and the config field
// cannot drift.
func RecordsDeploymentHistory(ctx context.Context, configured bool) bool {
	return configured || recordDeploymentHistoryEnv(ctx)
}
