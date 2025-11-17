package cache

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileCacheExpiryBehavior tests that the cache writes files with correct expiry
func TestFileCacheExpiryBehavior(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create cache with 1 minute expiry
	cache, err := newFileCacheWithBaseDir[string](tempDir, 1)
	require.NoError(t, err)

	fingerprint := struct {
		Key string `json:"key"`
	}{
		Key: "test-expiry",
	}

	// Compute and store a value
	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		return "test-value", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "test-value", result)

	// Find the cache file and verify it has the correct expiry
	cacheFiles, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	require.NoError(t, err)
	require.Len(t, cacheFiles, 1)

	// Read the cache file and check expiry
	data, err := os.ReadFile(cacheFiles[0])
	require.NoError(t, err)

	var entry cacheEntry
	err = json.Unmarshal(data, &entry)
	require.NoError(t, err)

	// Verify expiry is set and is approximately 1 minute from now
	assert.False(t, entry.Expiry.IsZero(), "Expiry should be set")
	expectedExpiry := time.Now().Add(time.Minute)
	timeDiff := entry.Expiry.Sub(expectedExpiry).Abs()
	assert.Less(t, timeDiff, 10*time.Second, "Expiry should be approximately 1 minute from creation time")
}

// TestReadFromCacheRespectsExpiry tests that readFromCache returns false for expired entries
func TestReadFromCacheRespectsExpiry(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](tempDir, 1)
	require.NoError(t, err)

	// Create an expired cache file
	expiredEntry := cacheEntry{
		Data:   json.RawMessage(`"expired-value"`),
		Expiry: time.Now().Add(-time.Hour), // Expired 1 hour ago
	}
	expiredData, err := json.Marshal(expiredEntry)
	require.NoError(t, err)

	expiredFile := filepath.Join(tempDir, "expired.json")
	require.NoError(t, os.WriteFile(expiredFile, expiredData, 0o644))

	// Try to read from expired cache - should return false
	result, found := cache.readFromCache(expiredFile)
	assert.False(t, found, "Should not find expired cache entry")
	assert.Equal(t, "", result, "Result should be zero value for expired entry")

	// Create a valid (non-expired) cache file
	validEntry := cacheEntry{
		Data:   json.RawMessage(`"valid-value"`),
		Expiry: time.Now().Add(time.Hour), // Expires in 1 hour
	}
	validData, err := json.Marshal(validEntry)
	require.NoError(t, err)

	validFile := filepath.Join(tempDir, "valid.json")
	require.NoError(t, os.WriteFile(validFile, validData, 0o644))

	// Try to read from valid cache - should return true
	result, found = cache.readFromCache(validFile)
	assert.True(t, found, "Should find valid cache entry")
	assert.Equal(t, "valid-value", result, "Should return correct value for valid entry")
}
