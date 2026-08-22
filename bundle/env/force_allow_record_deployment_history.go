package env

import "context"

// ForceAllowRecordDeploymentHistoryVariable force-allows the recording feature for CLI tests and DMS development.
// Deliberately undocumented; see validate.ValidateRecordDeploymentHistory.
const ForceAllowRecordDeploymentHistoryVariable = "DATABRICKS_BUNDLE_FORCE_ALLOW_RECORD_DEPLOYMENT_HISTORY"

// ForceAllowRecordDeploymentHistory reports whether the environment force allows
// experimental.record_deployment_history despite it being gated off.
func ForceAllowRecordDeploymentHistory(ctx context.Context) bool {
	value, ok := get(ctx, []string{
		ForceAllowRecordDeploymentHistoryVariable,
	})
	return ok && value != ""
}
