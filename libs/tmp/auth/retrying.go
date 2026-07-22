package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/databricks/sdk-go/core/apiretry"
	"github.com/databricks/sdk-go/core/ops"
)

// defaultRetriableStatuses is the set of HTTP status codes the default token
// retrier treats as transient.
var defaultRetriableStatuses = []int{
	http.StatusTooManyRequests,    // 429
	http.StatusBadGateway,         // 502
	http.StatusServiceUnavailable, // 503
	http.StatusGatewayTimeout,     // 504
}

// NewRetryingTokenProvider wraps a [TokenProvider] with retry logic for
// transient failures.
//
// By default, the return TokenProvider will retry on transient error
// failures (network errors, timeouts, ...) as well as the following HTTP
// status codes:
//
//   - 429 Too Many Requests
//   - 502 Bad Gateway
//   - 503 Service Unavailable
//   - 504 Gateway Timeout
//
// The provided options are applied after the defaults, allowing callers to
// override the default timeout and retry behavior.
func NewRetryingTokenProvider(inner TokenProvider, opts ...ops.Option) TokenProvider {
	defaults := []ops.Option{
		ops.WithTimeout(1 * time.Minute),
		ops.WithRetrier(func() ops.Retrier {
			return apiretry.NewRetrier(
				ops.BackoffPolicy{
					Initial: 200 * time.Millisecond,
					Maximum: 10 * time.Second,
				},
				apiretry.RetrierConfig{StatusCodes: defaultRetriableStatuses},
			)
		}),
	}
	return &retryingTokenProvider{
		inner: inner,
		opts:  append(defaults, opts...),
	}
}

// retryingTokenProvider wraps a TokenProvider with retry logic for transient
// failures during token acquisition. Each retry calls the underlying Token()
// method, which creates a fresh HTTP request.
type retryingTokenProvider struct {
	inner TokenProvider
	opts  []ops.Option
}

// Token returns a token from the underlying provider, retrying on transient
// errors.
func (r *retryingTokenProvider) Token(ctx context.Context) (*Token, error) {
	return ops.ExecuteWithResult(ctx, r.inner.Token, r.opts...)
}
