// Package retry retries calls that fail with transient server errors the SDK does not retry itself.
package retry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/databricks/cli/libs/diag"
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

// IsTransient returns true for 5xx and 408 errors that the SDK did not already retry, for the
// SDK's client-side inactivity timeout, and for an HTTP/2 connection closed by its health check.
// The SDK retries 504s matching its allTransientErrors patterns (e.g. "deadline exceeded");
// this covers the remaining 504s like TEMPORARILY_UNAVAILABLE.
func IsTransient(ctx context.Context, err error) bool {
	if isInactivityTimeout(err) || isConnectionLost(err) {
		return true
	}
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

// isInactivityTimeout reports whether the SDK gave up on a request because the server sent nothing
// for HTTPTimeout ("request timed out after 1m30s of inactivity"). The SDK does not retry it. The
// workspace files API stalls like this for a few minutes at a time; the server may or may not have
// applied the request, which is the same uncertainty as a 504.
//
// We match the message because the SDK builds it with fmt.Errorf in httpclient/api_client.go
// and exports no sentinel error or type for it.
func isInactivityTimeout(err error) bool {
	urlErr, ok := errors.AsType[*url.Error](err)
	if !ok {
		return false
	}
	msg := urlErr.Err.Error()
	return strings.HasPrefix(msg, "request timed out after ") && strings.HasSuffix(msg, " of inactivity")
}

// isConnectionLost reports whether the HTTP/2 health check closed the connection because a ping went
// unanswered (see libs/httptransport). It replaces the inactivity timeout for a request on
// a dead connection, with the same uncertainty about whether the server applied it, and the retry
// goes out on a new connection.
//
// We match the message because net/http builds it with errors.New in closeForLostPing and exports
// no sentinel error or type for it.
func isConnectionLost(err error) bool {
	urlErr, ok := errors.AsType[*url.Error](err)
	return ok && urlErr.Err.Error() == "http2: client connection lost"
}

// transientReadMessages are the transport errors, matched by message, that are worth retrying on
// an idempotent call:
//   - REFUSED_STREAM, the HTTP/2 refusal in issue #6472, where the server refused the stream before
//     processing the request.
//   - unexpected EOF, where the connection was cut mid-response. Observed on GET /clusters/get and
//     POST /workspace-files/import-file.
var transientReadMessages = []string{"REFUSED_STREAM", "unexpected EOF"}

// windowsConnectionReset is WSAECONNRESET ("An existing connection was forcibly closed by the
// remote host"). The SDK retries a reset by matching the Unix text "connection reset by peer",
// which Windows does not use. No Unix errno has this value, so the check only matches on Windows.
const windowsConnectionReset = syscall.Errno(10054)

// IsTransientRead returns true for errors worth retrying on an idempotent call (a read or an
// overwriting write): the transient server errors IsTransient covers, plus the transport errors
// in transientReadMessages, a bare EOF and a connection reset on Windows. A bare EOF means the
// server closed the connection during the TLS handshake (seen on GET /preview/scim/v2/Me), or,
// over HTTP/1.1, before responding; HTTP/2 reports a cut mid-request as "unexpected EOF". The SDK
// already retries the transport errors it recognizes
// (connection reset/refused, i/o timeout, TLS handshake) for its full RetryTimeout, so retrying
// every *url.Error would restart that multi-minute loop; these are the gaps it leaves. net/http
// bundles unexported http2 types, so we match the message, as the SDK does for its own transient
// url errors.
func IsTransientRead(ctx context.Context, err error) bool {
	if IsTransient(ctx, err) {
		return true
	}
	if _, ok := errors.AsType[*url.Error](err); !ok {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, windowsConnectionReset) {
		return true
	}
	return slices.ContainsFunc(transientReadMessages, func(m string) bool {
		return strings.Contains(err.Error(), m)
	})
}

// With retries fn while check returns true for the error, up to MaxRetries times.
func With[T any](ctx context.Context, check func(context.Context, error) bool, fn func() (T, error)) (T, error) {
	for attempt := 0; ; attempt++ {
		result, err := fn()
		if err == nil || attempt >= MaxRetries || !check(ctx, err) {
			return result, err
		}
		if err := Pause(ctx, err); err != nil {
			var zero T
			return zero, err
		}
	}
}

// Pause reports that the call that failed with err is retried and waits out the pause between
// attempts, for callers that run their own retry loop. It returns ctx.Err() if ctx is done first.
func Pause(ctx context.Context, err error) error {
	log.Warnf(ctx, "%s", retryMessage(err))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(interval(ctx)):
		return nil
	}
}

// retryMessage describes the error that triggered a retry.
func retryMessage(err error) string {
	// The "retrying after " prefix is what acceptance tests suppress (SuppressLines.warn_retry).
	msg := "retrying after " + err.Error()
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		endpoint := ""
		if e := diag.FormatAPIErrorEndpoint(apiErr); e != "" {
			endpoint = " from " + e
		}
		status := diag.FormatAPIErrorStatus(apiErr)
		msg = fmt.Sprintf("retrying after %s%s", status, endpoint)
		// Include the context the caller added to the error, e.g. the path of the file being written.
		if err.Error() != apiErr.Error() {
			msg += ": " + err.Error()
		}
	}
	return msg
}
