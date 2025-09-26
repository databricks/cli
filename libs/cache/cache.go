package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// Cache provides an abstract interface for caching content to local disk.
// Implementations should handle storing and retrieving cached components
// using fingerprints for cache invalidation.
type Cache[T any] interface {
	// GetOrCompute retrieves cached content for the given fingerprint, or computes it using the provided function.
	// If the content is found in cache, it is returned directly.
	// If not found, the compute function is called, its result is cached, and then returned.
	// The fingerprint can be any struct that will be serialized deterministically for cache key generation.
	// Returns an error if the cache operation or compute function fails.
	GetOrCompute(ctx context.Context, fingerprint any, compute func(ctx context.Context) (T, error)) (T, error)
}

// fingerprintToHash converts any struct to a deterministic string representation for use as a cache key.
func fingerprintToHash(fingerprint any) (string, error) {
	// Serialize to JSON with sorted keys for deterministic output
	data, err := json.Marshal(fingerprint)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fingerprint: %w", err)
	}

	// Parse back to ensure consistent key ordering
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", fmt.Errorf("failed to unmarshal fingerprint: %w", err)
	}

	// Sort keys deterministically
	normalized := normalizeForFingerprint(obj)

	// Re-marshal with normalized structure
	normalizedData, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("failed to marshal normalized fingerprint: %w", err)
	}

	// Hash the result for a consistent, reasonably-sized key
	hash := sha256.Sum256(normalizedData)
	return hex.EncodeToString(hash[:]), nil
}

// normalizeForFingerprint recursively sorts map keys to ensure deterministic serialization.
func normalizeForFingerprint(obj any) any {
	switch v := obj.(type) {
	case map[string]any:
		// Sort keys
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Create ordered map
		result := make(map[string]any, len(v))
		for _, k := range keys {
			result[k] = normalizeForFingerprint(v[k])
		}
		return result
	case []any:
		// Normalize each element in the slice
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = normalizeForFingerprint(item)
		}
		return result
	default:
		// Primitive types are returned as-is
		return v
	}
}
