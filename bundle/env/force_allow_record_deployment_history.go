package env

import "context"

// ForceAllowRecordDeploymentHistoryVariable names the environment variable that force
// allows experimental.record_deployment_history. It is deliberately undocumented: the
// feature is complete but cannot be exposed to users yet (see
// validate.ValidateRecordDeploymentHistory for why), and this variable exists so the
// CLI's own tests and the developers working on DMS can exercise the code path meanwhile.
const ForceAllowRecordDeploymentHistoryVariable = "DATABRICKS_BUNDLE_FORCE_ALLOW_RECORD_DEPLOYMENT_HISTORY"

// ForceAllowRecordDeploymentHistory reports whether the environment force allows
// experimental.record_deployment_history despite it being gated off.
func ForceAllowRecordDeploymentHistory(ctx context.Context) bool {
	value, ok := get(ctx, []string{
		ForceAllowRecordDeploymentHistoryVariable,
	})
	return ok && value != ""
}
