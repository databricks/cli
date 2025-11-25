package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileCacheExpiryBehavior tests that the cache writes files and respects expiry based on mtime
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

	// Find the cache file and verify it was created
	cacheFiles, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	require.NoError(t, err)
	require.Len(t, cacheFiles, 1)

	// Verify the file contains the expected data (stored directly, not wrapped)
	data, err := os.ReadFile(cacheFiles[0])
	require.NoError(t, err)
	assert.Equal(t, `"test-value"`, string(data))

	// Verify mtime is recent (within last 10 seconds)
	info, err := os.Stat(cacheFiles[0])
	require.NoError(t, err)
	age := time.Since(info.ModTime())
	assert.Less(t, age, 10*time.Second, "File should have been created recently")
}

// TestReadFromCacheRespectsExpiry tests that readFromCache returns false for expired entries based on mtime
func TestReadFromCacheRespectsExpiry(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](tempDir, 1) // 1 minute expiry
	require.NoError(t, err)

	// Create an expired cache file by setting its mtime to 2 hours ago
	expiredFile := filepath.Join(tempDir, "expired.json")
	require.NoError(t, os.WriteFile(expiredFile, []byte(`"expired-value"`), 0o644))
	oldTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(expiredFile, oldTime, oldTime))

	// Try to read from expired cache - should return false
	result, found := cache.readFromCache(expiredFile)
	assert.False(t, found, "Should not find expired cache entry")
	assert.Equal(t, "", result, "Result should be zero value for expired entry")

	// Create a valid (non-expired) cache file with recent mtime
	validFile := filepath.Join(tempDir, "valid.json")
	require.NoError(t, os.WriteFile(validFile, []byte(`"valid-value"`), 0o644))

	// Try to read from valid cache - should return true
	result, found = cache.readFromCache(validFile)
	assert.True(t, found, "Should find valid cache entry")
	assert.Equal(t, "valid-value", result, "Should return correct value for valid entry")
}
