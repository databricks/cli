// Package retry retries calls that fail with transient server errors the SDK does not retry itself.
package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// MaxRetries is the number of additional attempts made after the first one fails.
const MaxRetries = 2

// DefaultInterval is the pause between attempts.
const DefaultInterval = 15 * time.Second

// IsTransient returns true for 5xx and 408 errors that the SDK did not already retry.
// The SDK retries 504s matching its allTransientErrors patterns (e.g. "deadline exceeded");
// this covers the remaining 504s like TEMPORARILY_UNAVAILABLE.
func IsTransient(ctx context.Context, err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return false
	}
	if apiErr.IsRetriable(ctx) {
		// Already handled by SDK
		return false
	}
	return apiErr.StatusCode >= 500 || apiErr.StatusCode == http.StatusRequestTimeout
}

// With retries fn while check returns true for the error, up to MaxRetries times,
// waiting interval between attempts.
func With[T any](ctx context.Context, interval time.Duration, check func(error) bool, fn func() (T, error)) (T, error) {
	for attempt := 0; ; attempt++ {
		result, err := fn()
		if err == nil || attempt >= MaxRetries || !check(err) {
			return result, err
		}
		msg := "retrying"
		if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
			endpoint := ""
			if rw := apiErr.ResponseWrapper; rw != nil && rw.Response != nil && rw.Response.Request != nil {
				req := rw.Response.Request
				endpoint = fmt.Sprintf(" from %s %s", req.Method, req.URL.Path)
			}
			msg = fmt.Sprintf("retrying after %d %s%s", apiErr.StatusCode, http.StatusText(apiErr.StatusCode), endpoint)
		}
		log.Warnf(ctx, "%s", msg)
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-time.After(interval):
		}
	}
}
