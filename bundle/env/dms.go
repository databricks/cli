package env

import "context"

// DeploymentHistoryVariable turns recording on ("true") or off (any other non-empty value)
// without setting the config field.
// Exists for CLI acceptance tests so the whole bundle suite runs with recording on.
// Deliberately undocumented.
const DeploymentHistoryVariable = "DATABRICKS_BUNDLE_DEPLOYMENT_HISTORY"

// DeploymentHistoryChoice returns whether the bundle turned recording on or off, from config or
// env var, and whether it chose at all. Explicit config takes priority over the env var.
// The env var turns recording on only as "true": any other non-empty value - a typo like "TRUE",
// or an opt-out spelled "0" - turns it off rather than silently enabling it. Unset or empty is
// no choice, which leaves recording to the workspace's enrollment.
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
