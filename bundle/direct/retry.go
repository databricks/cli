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

var defaultRetryInterval = 30 * time.Second

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
	var apiErr *apierr.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 504 && !apiErr.IsRetriable(ctx)
}

// retryWith retries fn while check returns true for the error, up to maxRetries times.
func retryWith[T any](ctx context.Context, check func(error) bool, fn func() (T, error)) (T, error) {
	interval := retryInterval(ctx)
	for attempt := 0; ; attempt++ {
		result, err := fn()
		if err == nil || attempt >= maxRetries || !check(err) {
			return result, err
		}
		var apiErr *apierr.APIError
		errors.As(err, &apiErr)
		endpoint := ""
		if rw := apiErr.ResponseWrapper; rw != nil && rw.Response != nil && rw.Response.Request != nil {
			req := rw.Response.Request
			endpoint = fmt.Sprintf(" from %s %s", req.Method, req.URL.Path)
		}
		log.Warnf(ctx, "retrying after %d %s%s", apiErr.StatusCode, http.StatusText(apiErr.StatusCode), endpoint)
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

// retryErr wraps retryOnTransient for functions that return only an error.
func retryErr(ctx context.Context, fn func() error) error {
	_, err := retryOnTransient(ctx, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
