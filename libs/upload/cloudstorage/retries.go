package cloudstorage

import (
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/databricks/sdk-go/core/apierr"
	"github.com/databricks/sdk-go/core/apiretry"
	"github.com/databricks/sdk-go/core/ops"
)

// maxRetries caps how many times a single request is retried before giving up.
var maxRetries = 4

// retryStatusCodes are the HTTP statuses retried for an idempotent request.
var retryStatusCodes = []int{
	http.StatusRequestTimeout,     // 408
	http.StatusTooManyRequests,    // 429
	http.StatusBadGateway,         // 502
	http.StatusServiceUnavailable, // 503
	http.StatusGatewayTimeout,     // 504
}

// IsRetryableStatus reports whether an HTTP status code should be retried. It is
// exported for callers (such as a resumable upload) that run their own
// resume-aware retry loop over Attempt rather than using Send.
func IsRetryableStatus(code int) bool {
	return slices.Contains(retryStatusCodes, code)
}

func newRetrier() ops.Retrier {
	return &retrier{maxRetries: maxRetries}
}

type retrier struct {
	backoff    ops.BackoffPolicy
	maxRetries int
	attempts   int
}

func (r *retrier) IsRetriable(err error) (time.Duration, bool) {
	r.attempts++
	if r.attempts > r.maxRetries {
		return 0, false
	}
	if !IsRetriable(err) {
		return 0, false
	}
	// Honor a Retry-After hint (carried on a throttling response's APIError)
	// when it exceeds the next backoff delay.
	return max(r.backoff.Delay(), apiretry.RetryDurationHint(err)), true
}

// IsRetryable reports whether an error from Attempt is worth retrying: a stalled
// attempt is, a short read (deterministic) is not, a retryable HTTP status (as
// an APIError) is, and any other failure is retried only if it is a transient
// network error (which excludes context cancellation and deadline). It is
// exported for callers (such as a resumable upload) that run their own
// resume-aware retry loop over Attempt.
func IsRetriable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errShortRead) {
		return false
	}
	if errors.Is(err, errStalled) {
		return true
	}
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		return IsRetryableStatus(apiErr.HTTPStatusCode())
	}
	return apiretry.IsTransientNetworkError(err)
}
