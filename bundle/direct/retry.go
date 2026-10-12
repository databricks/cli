package direct

import (
	"context"
	"time"

	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/retry"
)

func retryInterval(ctx context.Context) time.Duration {
	return bundleenv.RetryInterval(ctx, retry.DefaultInterval)
}

func retryWith[T any](ctx context.Context, check func(error) bool, fn func() (T, error)) (T, error) {
	return retry.With(ctx, retryInterval(ctx), check, fn)
}

// retryOnTransient retries fn on transient 504 errors that the SDK did not already handle.
func retryOnTransient[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	return retryWith(ctx, func(err error) bool { return retry.IsTransient(ctx, err) }, fn)
}

// retryOnTransientErr wraps retryOnTransient for functions that return only an error.
func retryOnTransientErr(ctx context.Context, fn func() error) error {
	_, err := retryOnTransient(ctx, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
