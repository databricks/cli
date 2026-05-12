package env

import "context"

// HTTPTimeoutSecondsVariable names the environment variable that overrides the HTTP timeout for bundle operations.
const HTTPTimeoutSecondsVariable = "DATABRICKS_BUNDLE_HTTP_TIMEOUT_SECONDS"

// HTTPTimeoutSeconds returns the HTTP timeout override for bundle operations.
func HTTPTimeoutSeconds(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		HTTPTimeoutSecondsVariable,
	})
}
