package env

import "context"

// RestrictedExecutionVariable names the environment variable that holds the flag whether
// restricted execution is enabled.
const RestrictedExecutionVariable = "DATABRICKS_BUNDLE_RESTRICTED_CODE_EXECUTION"

// RestrictedExecution returns the environment variable that holds the flag whether
// restricted execution is enabled.
func RestrictedExecution(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		RestrictedExecutionVariable,
	})
}
