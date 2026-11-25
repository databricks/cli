package env

import "context"

// DeploymentHistoryVariable sets recording without the config field: "true" turns it on, any other
// non-empty value off. Exists for CLI acceptance tests; deliberately undocumented.
const DeploymentHistoryVariable = "DATABRICKS_BUNDLE_DEPLOYMENT_HISTORY"

// DeploymentHistoryChoice returns the bundle's recording setting (config, then env var) and whether
// it has one. Unset or empty is no choice, leaving recording to the workspace's enrollment.
func DeploymentHistoryChoice(ctx context.Context, configured *bool) (enabled, explicit bool) {
	if configured != nil {
		return *configured, true
	}
	value, _ := get(ctx, []string{DeploymentHistoryVariable})
	if value == "" {
		return false, false
	}
	return value == "true", true
}
