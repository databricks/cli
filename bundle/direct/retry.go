package direct

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
)

const maxRetries = 2

var defaultRetryInterval = 15 * time.Second

func retryInterval(ctx context.Context) time.Duration {
	v, ok := bundleenv.RetryIntervalMs(ctx)
	if !ok {
		return defaultRetryInterval
	}
	ms, err := strconv.Atoi(v)
	if err != nil {
		return defaultRetryInterval
	}
	return time.Duration(ms) * time.Millisecond
}

// isTransient returns true for 504 errors that the SDK did not already retry.
// The SDK retries 504s matching its allTransientErrors patterns (e.g. "deadline exceeded");
// this covers the remaining 504s like TEMPORARILY_UNAVAILABLE.
func isTransient(ctx context.Context, err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return false
	}
	if apiErr.IsRetriable(ctx) {
		// Already handled by SDK
		return false
	}
	return apiErr.StatusCode == http.StatusGatewayTimeout
}

// retryWith retries fn while check returns true for the error, up to maxRetries times.
func retryWith[T any](ctx context.Context, check func(error) bool, fn func() (T, error)) (T, error) {
	interval := retryInterval(ctx)
	for attempt := 0; ; attempt++ {
		result, err := fn()
		if err == nil || attempt >= maxRetries || !check(err) {
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

// retryOnTransient retries fn on transient 504 errors that the SDK did not already handle.
func retryOnTransient[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	return retryWith(ctx, func(err error) bool { return isTransient(ctx, err) }, fn)
}

// retryOnTransientErr wraps retryOnTransient for functions that return only an error.
func retryOnTransientErr(ctx context.Context, fn func() error) error {
	_, err := retryOnTransient(ctx, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
