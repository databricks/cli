package env

import (
	"context"
	"strconv"
	"time"
)

// RetryIntervalMsVariable names the environment variable that overrides the retry interval for bundle operations.
const RetryIntervalMsVariable = "DATABRICKS_BUNDLE_RETRY_INTERVAL_MS"

// RetryIntervalMs returns the retry interval override (in milliseconds) for bundle operations.
func RetryIntervalMs(ctx context.Context) (string, bool) {
	return get(ctx, []string{RetryIntervalMsVariable})
}

// RetryInterval returns the retry interval for bundle operations, or def if the override is unset or malformed.
func RetryInterval(ctx context.Context, def time.Duration) time.Duration {
	v, ok := RetryIntervalMs(ctx)
	if !ok {
		return def
	}
	ms, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return time.Duration(ms) * time.Millisecond
}
