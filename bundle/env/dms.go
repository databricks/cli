package env

import "context"

// DeploymentHistoryVariable enables recording without setting the config field.
// Exists for CLI acceptance tests so the whole bundle suite runs with recording on.
// Deliberately undocumented.
const DeploymentHistoryVariable = "DATABRICKS_BUNDLE_DEPLOYMENT_HISTORY"

// DeploymentHistoryChoice returns whether the bundle turned recording on or off, from config or
// env var, and whether it chose at all. Explicit config takes priority over the env var.
// The env var only counts as "true" or "false": anything else - unset, or a typo like "TRUE" or
// "yes" - is no choice, which leaves recording to the workspace's enrollment.
func DeploymentHistoryChoice(ctx context.Context, configured *bool) (enabled, explicit bool) {
	if configured != nil {
		return *configured, true
	}
	value, _ := get(ctx, []string{DeploymentHistoryVariable})
	switch value {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}
