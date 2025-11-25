package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Cache provides an abstract interface for caching content to local disk.
// Implementations should handle storing and retrieving cached components
// using fingerprints for cache invalidation.
// Cache operations fail open: if caching fails, the compute function is still called.
type Cache[T any] interface {
	// GetOrCompute retrieves cached content for the given fingerprint, or computes it using the provided function.
	// If the content is found in cache, it is returned directly.
	// If not found, the compute function is called, its result is cached, and then returned.
	// The fingerprint can be any struct that will be serialized deterministically for cache key generation.
	// Cache failures do not block computation - if caching fails, compute is called anyway.
	// Returns an error only if the compute function fails.
	GetOrCompute(ctx context.Context, fingerprint any, compute func(ctx context.Context) (T, error)) (T, error)
}

// fingerprintToHash converts any struct to a deterministic string representation for use as a cache key.
func fingerprintToHash(fingerprint any) (string, error) {
	// Marshal map - json.Marshal sorts map keys alphabetically
	data, err := json.Marshal(fingerprint)
	if err != nil {
		return "", fmt.Errorf("failed to marshal normalized fingerprint: %w", err)
	}

	// Hash for consistent, reasonably-sized key.
	// hash[:] converts the [32]byte array returned by Sum256 to a []byte slice.
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
