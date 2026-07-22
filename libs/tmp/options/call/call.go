// Package call defines the options used to configure individual calls
// against the Databricks API.
package call

import (
	"time"

	"github.com/databricks/sdk-go/core/ops"
	"github.com/databricks/cli/libs/tmp/options/internaloptions"
)

// Option configures a single call against the Databricks API.
type Option func(*internaloptions.CallOptions) error

// WithRetrier returns an Option that uses the given Retrier provider. If no
// retrier is provided, the call is not retried.
//
// The provider function must be thread-safe.
func WithRetrier(provider func() ops.Retrier) Option {
	return func(c *internaloptions.CallOptions) error {
		c.Retrier = provider
		return nil
	}
}

// WithDisableRetry is a convenience option that disables retries.
func WithDisableRetry() Option {
	return func(c *internaloptions.CallOptions) error {
		c.Retrier = nil
		return nil
	}
}

// WithTimeout returns an Option that sets the timeout for the call. If the
// context already has a deadline, it is updated to the minimum of the
// context's deadline and the timeout. A timeout of zero means no timeout.
//
// The timeout covers the entire execution, including retries.
func WithTimeout(t time.Duration) Option {
	return func(c *internaloptions.CallOptions) error {
		c.Timeout = t
		return nil
	}
}

// WithLimiter returns an Option that uses the given Limiter. If no limiter is
// provided, the call is not rate limited.
func WithLimiter(l ops.Limiter) Option {
	return func(c *internaloptions.CallOptions) error {
		c.RateLimiter = l
		return nil
	}
}
