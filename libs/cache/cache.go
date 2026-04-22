package cache

import (
	"context"
	"encoding/json"

	"github.com/databricks/cli/libs/log"
)

// cacheImpl is the internal interface for cache implementations.
type cacheImpl interface {
	// getOrComputeJSON retrieves cached JSON bytes or computes them.
	// The compute function must return JSON-encoded data as []byte.
	// The returned []byte is also expected to be JSON-encoded.
	getOrComputeJSON(ctx context.Context, fingerprint any, compute func(ctx context.Context) ([]byte, error)) ([]byte, error)

	// getJSON returns cached JSON bytes for fingerprint, or (nil, false) on
	// miss or when caching is disabled. Never computes, never writes.
	getJSON(ctx context.Context, fingerprint any) ([]byte, bool)

	// putJSON writes data to the cache under fingerprint, overwriting any
	// existing entry. When caching is disabled it is a no-op. Failures are
	// silent (logged at debug).
	putJSON(ctx context.Context, fingerprint any, data []byte)
}

// Cache provides a concrete cache that works with any type through the generic GetOrCompute function.
// Create with NewCache() and use GetOrCompute[T]() for type-safe caching.
type Cache struct {
	impl cacheImpl
}

// Get returns the cached value for the given fingerprint, or (zero, false) on
// miss. Unlike GetOrCompute it never invokes compute and never writes. Use
// this when the caller wants a read-only probe and will handle a miss
// explicitly, without the cache-level "error while computing" log that an
// erroring compute callback would emit.
func Get[T any](ctx context.Context, c *Cache, fingerprint any) (T, bool) {
	var zero T
	data, ok := c.impl.getJSON(ctx, fingerprint)
	if !ok {
		return zero, false
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		log.Debugf(ctx, "[Local Cache] failed to unmarshal cached data: %v", err)
		return zero, false
	}
	return result, true
}

// GetOrCompute retrieves cached content for the given fingerprint, or computes it using the provided function.
// If the content is found in cache, it is returned directly.
// If not found, the compute function is called, its result is cached, and then returned.
// The fingerprint can be any struct that will be serialized deterministically for cache key generation.
// Cache operations fail open: if caching fails, the compute function is still called.
// Returns an error only if the compute function fails.
// The type parameter T is inferred from the compute function's return type.
func GetOrCompute[T any](ctx context.Context, c *Cache, fingerprint any, compute func(ctx context.Context) (T, error)) (T, error) {
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
	jsonBytes, err := c.impl.getOrComputeJSON(ctx, fingerprint, computeJSON)
	if err != nil {
		return zero, err
	}

	// Unmarshal into the target type
	var result T
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		// Fail open: if cached data is corrupted, log and recompute
		log.Debugf(ctx, "[Local Cache] failed to unmarshal cached data, recomputing: %v", err)
		return compute(ctx)
	}

	return result, nil
}

// Put serializes value to JSON and writes it to the cache under fingerprint,
// overwriting any existing entry. Failures are silent; when caching is
// disabled it is a no-op. Use this when the caller wants an unconditional
// write (e.g. recording a negative sentinel) rather than the read-then-write
// semantics of GetOrCompute.
func Put[T any](ctx context.Context, c *Cache, fingerprint any, value T) {
	data, err := json.Marshal(value)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] failed to marshal value for cache write: %v", err)
		return
	}
	c.impl.putJSON(ctx, fingerprint, data)
}
