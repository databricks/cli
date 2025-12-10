package cache

import (
	"context"
	"encoding/json"
	"fmt"
)

// cacheImpl is the internal interface for cache implementations.
type cacheImpl interface {
	getOrComputeJSON(ctx context.Context, fingerprint any, compute func(ctx context.Context) ([]byte, error)) ([]byte, bool, error)
}

// Cache provides a concrete cache that works with any type through the generic GetOrCompute function.
// Create with NewCache() and use GetOrCompute[T]() for type-safe caching.
type Cache struct {
	impl cacheImpl
}

// GetOrCompute retrieves cached content for the given fingerprint, or computes it using the provided function.
// If the content is found in cache, it is returned directly.
// If not found, the compute function is called, its result is cached, and then returned.
// The fingerprint can be any struct that will be serialized deterministically for cache key generation.
// Cache operations fail open: if caching fails, the compute function is still called.
// Returns an error only if the compute function fails.
// The type parameter T is inferred from the compute function's return type.
func GetOrCompute[T any](c *Cache, ctx context.Context, fingerprint any, compute func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	// Wrap the compute function to serialize to JSON
	computeJSON := func(ctx context.Context) ([]byte, error) {
		result, err := compute(ctx)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}

	// Call the internal method
	jsonBytes, fromCache, err := c.impl.getOrComputeJSON(ctx, fingerprint, computeJSON)
	if err != nil {
		return zero, err
	}

	// Unmarshal into the target type
	var result T
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		// If we got corrupted data from cache, fail open and recompute
		if fromCache {
			result, computeErr := compute(ctx)
			if computeErr != nil {
				return zero, computeErr
			}
			return result, nil
		}
		// If compute function returned invalid JSON, that's a real error
		return zero, fmt.Errorf("failed to unmarshal computed data: %w", err)
	}

	return result, nil
}
