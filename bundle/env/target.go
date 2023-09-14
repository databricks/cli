package env

import "context"

// TargetVariable names the environment variable that holds the bundle target to use.
const TargetVariable = "DATABRICKS_BUNDLE_TARGET"

// Target returns the bundle target environment variable.
func Target(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		TargetVariable,

		// Primary variable name for the bundle target until v0.203.2.
		// See https://github.com/databricks/cli/pull/670.
		"DATABRICKS_BUNDLE_ENV",
	})
}
