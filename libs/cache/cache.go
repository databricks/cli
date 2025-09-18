package cache

import (
	"context"
)

// Cache provides an abstract interface for caching content to local disk.
// Implementations should handle storing and retrieving cached components
// using fingerprints for cache invalidation.
type Cache interface {
	// GetOrCompute retrieves cached content for the given fingerprint, or computes it using the provided function.
	// If the content is found in cache, it is returned directly.
	// If not found, the compute function is called, its result is cached, and then returned.
	// Returns an error if the cache operation or compute function fails.
	GetOrCompute(ctx context.Context, fingerprint string, compute func(ctx context.Context) (any, error)) (any, error)
}
