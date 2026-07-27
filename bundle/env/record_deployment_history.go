package env

import "context"

// EnableRecordDeploymentHistoryVariable names the environment variable that lifts the
// error on experimental.record_deployment_history. It is deliberately undocumented: the
// feature is complete but cannot be exposed to users yet (see
// validate.ValidateRecordDeploymentHistory for why), and this variable exists so the
// CLI's own tests and the developers working on DMS can exercise the code path meanwhile.
const EnableRecordDeploymentHistoryVariable = "DATABRICKS_BUNDLE_ENABLE_RECORD_DEPLOYMENT_HISTORY"

// EnableRecordDeploymentHistory reports whether the environment opts into
// experimental.record_deployment_history despite it being gated off.
func EnableRecordDeploymentHistory(ctx context.Context) bool {
	value, ok := get(ctx, []string{
		EnableRecordDeploymentHistoryVariable,
	})
	return ok && value != ""
}
