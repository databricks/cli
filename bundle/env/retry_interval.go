package env

import "context"

// RetryIntervalMsVariable names the environment variable that overrides the retry interval for bundle operations.
const RetryIntervalMsVariable = "DATABRICKS_BUNDLE_RETRY_INTERVAL_MS"

// RetryIntervalMs returns the retry interval override (in milliseconds) for bundle operations.
func RetryIntervalMs(ctx context.Context) (string, bool) {
	return get(ctx, []string{RetryIntervalMsVariable})
}
