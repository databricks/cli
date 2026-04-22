package cache

import "context"

type noopFileCache struct{}

func (c *noopFileCache) getOrComputeJSON(ctx context.Context, fingerprint any, compute func(ctx context.Context) ([]byte, error)) ([]byte, error) {
	return compute(ctx)
}

func (c *noopFileCache) getJSON(ctx context.Context, fingerprint any) ([]byte, bool) {
	return nil, false
}

func (c *noopFileCache) putJSON(ctx context.Context, fingerprint any, data []byte) {
}
