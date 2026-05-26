package dresources

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

var retriableCodes = map[int]bool{
	408: true,
	500: true,
	502: true,
	503: true,
	504: true,
}

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

func isTransient(err error) bool {
	var apiErr *apierr.APIError
	return errors.As(err, &apiErr) && retriableCodes[apiErr.StatusCode]
}

// retryOnTransient retries fn on transient HTTP errors (408/500/502/503/504) up to maxRetries times.
func retryOnTransient[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	interval := retryInterval(ctx)
	for attempt := 0; ; attempt++ {
		result, err := fn()
		if err == nil || attempt >= maxRetries || !isTransient(err) {
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

// retryErr wraps retryOnTransient for functions that return only an error.
func retryErr(ctx context.Context, fn func() error) error {
	_, err := retryOnTransient(ctx, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
