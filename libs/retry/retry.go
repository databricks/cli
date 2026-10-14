// Package retry retries calls that fail with transient server errors the SDK does not retry itself.
package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/sdk-go/core/apierr"
)

// MaxRetries is the number of additional attempts made after the first one fails.
const MaxRetries = 2

// IntervalMsVariable names the environment variable that overrides the pause between attempts.
const IntervalMsVariable = "DATABRICKS_BUNDLE_RETRY_INTERVAL_MS"

const defaultInterval = 15 * time.Second

// interval returns the pause between attempts: the override in milliseconds if set and valid, 15s otherwise.
func interval(ctx context.Context) time.Duration {
	v, ok := env.Lookup(ctx, IntervalMsVariable)
	if !ok {
		return defaultInterval
	}
	ms, err := strconv.Atoi(v)
	if err != nil {
		return defaultInterval
	}
	return time.Duration(ms) * time.Millisecond
}

// IsTransient returns true for 5xx and 408 errors that the client transport did not already
// retry. The transport (libs/apiclient) retries 429 and 503; this covers the remaining 5xx,
// e.g. 504 TEMPORARILY_UNAVAILABLE.
func IsTransient(ctx context.Context, err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return false
	}
	status := apiErr.HTTPStatusCode()
	if status == http.StatusServiceUnavailable || status == http.StatusTooManyRequests {
		// Already handled by the transport
		return false
	}
	return status >= 500 || status == http.StatusRequestTimeout
}

// IsTransientRead returns true for errors worth retrying on an idempotent read: the transient
// server errors IsTransient covers, plus the HTTP/2 REFUSED_STREAM in issue #6472, where the
// server refused the stream before processing the request so the read is safe to replay. The
// SDK already retries the transport errors it recognizes (connection reset/refused, i/o timeout,
// TLS handshake) for its full RetryTimeout, so retrying every *url.Error would restart that
// multi-minute loop; REFUSED_STREAM is the gap it leaves. net/http bundles unexported http2
// types, so we match the message, as the SDK does for its own transient url errors.
func IsTransientRead(ctx context.Context, err error) bool {
	if IsTransient(ctx, err) {
		return true
	}
	_, ok := errors.AsType[*url.Error](err)
	return ok && strings.Contains(err.Error(), "REFUSED_STREAM")
}

// With retries fn while check returns true for the error, up to MaxRetries times.
func With[T any](ctx context.Context, check func(context.Context, error) bool, fn func() (T, error)) (T, error) {
	wait := interval(ctx)
	for attempt := 0; ; attempt++ {
		result, err := fn()
		if err == nil || attempt >= MaxRetries || !check(ctx, err) {
			return result, err
		}
		log.Warnf(ctx, "%s", retryMessage(err))
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// retryMessage describes the error that triggered a retry.
func retryMessage(err error) string {
	msg := "retrying"
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		endpoint := ""
		if method, u, ok := apiclient.RequestOf(err); ok {
			endpoint = fmt.Sprintf(" from %s %s", method, u.Path)
		}
		status := apiErr.HTTPStatusCode()
		msg = fmt.Sprintf("retrying after %d %s%s", status, http.StatusText(status), endpoint)
		// Include the context the caller added to the error, e.g. the path of the file being written.
		if err.Error() != apiErr.Error() {
			msg += ": " + err.Error()
		}
	}
	return msg
}
