package cache

import "context"

type noopFileCache struct{}

func (c *noopFileCache) getOrComputeJSON(ctx context.Context, fingerprint any, compute func(ctx context.Context) ([]byte, error)) ([]byte, error) {
	return compute(ctx)
}
