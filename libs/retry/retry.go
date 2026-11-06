// Package retry retries calls that fail with transient server errors the SDK does not retry itself.
package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
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

// transientReadMessages are the transport errors, matched by message, that are worth retrying on a read:
//   - REFUSED_STREAM, the HTTP/2 refusal in issue #6472, where the server refused the stream before
//     processing the request.
//   - unexpected EOF, where the connection was cut mid-response. Observed on GET /clusters/get.
var transientReadMessages = []string{"REFUSED_STREAM", "unexpected EOF"}

// IsTransientRead returns true for errors worth retrying on an idempotent read: the transient
// server errors IsTransient covers, plus the transport errors in transientReadMessages. The SDK
// already retries the transport errors it recognizes (connection reset/refused, i/o timeout,
// TLS handshake) for its full RetryTimeout, so retrying every *url.Error would restart that
// multi-minute loop; these are the gaps it leaves. net/http bundles unexported http2
// types, so we match the message, as the SDK does for its own transient url errors.
func IsTransientRead(ctx context.Context, err error) bool {
	if IsTransient(ctx, err) {
		return true
	}
	if _, ok := errors.AsType[*url.Error](err); !ok {
		return false
	}
	return slices.ContainsFunc(transientReadMessages, func(m string) bool {
		return strings.Contains(err.Error(), m)
	})
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
		if rw := apiErr.ResponseWrapper; rw != nil && rw.Response != nil && rw.Response.Request != nil {
			req := rw.Response.Request
			endpoint = fmt.Sprintf(" from %s %s", req.Method, req.URL.Path)
		}
		msg = fmt.Sprintf("retrying after %d %s%s", apiErr.StatusCode, http.StatusText(apiErr.StatusCode), endpoint)
		// Include the context the caller added to the error, e.g. the path of the file being written.
		if err.Error() != apiErr.Error() {
			msg += ": " + err.Error()
		}
	}
	return msg
}
