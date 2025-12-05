package cache

import "context"

type NoopFileCache[T any] struct{}

func (c *NoopFileCache[T]) GetOrCompute(ctx context.Context, fingerprint any, compute func(ctx context.Context) (T, error)) (T, error) {
	return compute(ctx)
}
