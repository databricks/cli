package cache

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileCache(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)

	cache := NewCache[string](ctx, "test-component", 60, nil)
	fc, ok := cache.(*FileCache[string])
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(fc.baseDir, cacheDir))

	// Verify directory was created
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Check permissions - Windows has different permission semantics
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	} else {
		// On Windows, verify directory is accessible by trying to create a test file
		testFile := filepath.Join(cacheDir, "test_access")
		err := os.WriteFile(testFile, []byte("test"), 0o600)
		assert.NoError(t, err)
		if err == nil {
			_ = os.Remove(testFile)
		}
	}
}

func TestNewFileCacheWithExistingDirectory(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "existing")

	// Create directory first
	err := os.MkdirAll(cacheDir, 0o700)
	require.NoError(t, err)

	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)

	cache := NewCache[string](ctx, "test-component", 60, nil)
	fc, ok := cache.(*FileCache[string])
	require.True(t, ok)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(fc.baseDir, cacheDir))
}

func TestNewFileCacheInvalidPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping invalid path test on Windows")
	}

	ctx := context.Background()
	// Try to create cache in a location that should fail
	invalidPath := "/root/invalid/path/that/should/not/exist"
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", invalidPath)

	cache := NewCache[string](ctx, "test-component", 60, nil)
	_, ok := cache.(*NoopFileCache[string])
	require.True(t, ok)
}

func TestFileCacheGetOrCompute(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)

	cache := NewCache[string](ctx, "test-component", 60, nil)

	fingerprint := struct {
		Key   string `json:"key"`
		Value int    `json:"value"`
	}{
		Key:   "test-key",
		Value: 123,
	}
	expectedValue := "computed-value"

	// First call should compute the value
	var computeCalls int32
	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return expectedValue, nil
	})

	require.NoError(t, err)
	assert.Equal(t, expectedValue, result)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls))

	// Second call should return cached value without computing
	result2, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "should-not-be-called", nil
	})

	require.NoError(t, err)
	assert.Equal(t, expectedValue, result2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls))
}

func TestFileCacheGetOrComputeError(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)

	cache := NewCache[string](ctx, "test-component", 60, nil)

	fingerprint := struct {
		Key string `json:"key"`
	}{
		Key: "error-key",
	}

	// Compute function returns error
	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		return "", assert.AnError
	})

	assert.Empty(t, result)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestFileCacheGetOrComputeConcurrency(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", cacheDir)

	cache := NewCache[string](ctx, "test-component", 60, nil)

	fingerprint := struct {
		Key string `json:"key"`
	}{
		Key: "concurrent-key",
	}
	expectedValue := "concurrent-value"
	var computeCalls int32

	// Start multiple goroutines that try to compute the same key
	numGoroutines := 10
	results := make(chan any, numGoroutines)
	errors := make(chan error, numGoroutines)

	for range numGoroutines {
		go func() {
			result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
				atomic.AddInt32(&computeCalls, 1)
				time.Sleep(10 * time.Millisecond) // Simulate work
				return expectedValue, nil
			})
			results <- result
			errors <- err
		}()
	}

	// Collect all results
	for range numGoroutines {
		result := <-results
		err := <-errors
		require.NoError(t, err)
		assert.Equal(t, expectedValue, result)
	}

	// With locking, writes are serialized but compute may be called multiple times
	// since goroutines check cache before acquiring lock
	calls := atomic.LoadInt32(&computeCalls)
	assert.GreaterOrEqual(t, calls, int32(1), "compute should be called at least once")
	assert.LessOrEqual(t, calls, int32(numGoroutines), "compute should not be called more than number of goroutines")
}

func TestFileCacheCleanupExpiredFiles(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	expiryMinutes := 60

	// Create some cache files manually - one expired, one valid
	now := time.Now()

	// Expired file - create it and set mtime to make it appear old
	expiredFile := filepath.Join(tempDir, "expired.json")
	require.NoError(t, os.WriteFile(expiredFile, []byte(`"expired-value"`), 0o644))
	// Set mtime to 2 hours ago (older than expiry)
	oldTime := now.Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(expiredFile, oldTime, oldTime))

	// Valid file - recently created
	validFile := filepath.Join(tempDir, "valid.json")
	require.NoError(t, os.WriteFile(validFile, []byte(`"valid-value"`), 0o644))

	// Non-cache file (should be ignored)
	nonCacheFile := filepath.Join(tempDir, "readme.txt")
	require.NoError(t, os.WriteFile(nonCacheFile, []byte("readme"), 0o644))

	// Create cache - this should trigger cleanup
	_, err := newFileCacheWithBaseDir[string](ctx, tempDir, expiryMinutes)
	require.NoError(t, err)

	// Check results
	_, err = os.Stat(expiredFile)
	assert.True(t, os.IsNotExist(err), "Expired file should be deleted")

	_, err = os.Stat(validFile)
	assert.False(t, os.IsNotExist(err), "Valid file should still exist")

	_, err = os.Stat(nonCacheFile)
	assert.False(t, os.IsNotExist(err), "Non-cache file should be ignored")
}

func TestFileCacheInvalidJSON(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](ctx, tempDir, 60)
	require.NoError(t, err)

	// Enable cache for this test
	cache.cacheEnabled = true

	fingerprint := struct {
		Key string `json:"key"`
	}{
		Key: "test-invalid-json",
	}

	// Manually write invalid JSON to the cache file
	cacheKey, err := fingerprintToHash(fingerprint)
	require.NoError(t, err)
	cachePath := cache.getCachePath(cacheKey)
	err = os.WriteFile(cachePath, []byte("invalid json {{{"), 0o600)
	require.NoError(t, err)

	// GetOrCompute should fail open and recompute when cache contains invalid JSON
	var computeCalls int32
	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "recomputed-value", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "recomputed-value", result)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls), "Should recompute when cache has invalid JSON")
}

func TestFileCacheCorruptedData(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[int](ctx, tempDir, 60)
	require.NoError(t, err)

	// Enable cache for this test
	cache.cacheEnabled = true

	fingerprint := struct {
		Key string `json:"key"`
	}{
		Key: "test-corrupted",
	}

	// Write valid JSON but wrong type (string instead of int)
	cacheKey, err := fingerprintToHash(fingerprint)
	require.NoError(t, err)
	cachePath := cache.getCachePath(cacheKey)
	err = os.WriteFile(cachePath, []byte(`"not-an-integer"`), 0o600)
	require.NoError(t, err)

	// GetOrCompute should fail open and recompute when cache type doesn't match
	var computeCalls int32
	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (int, error) {
		atomic.AddInt32(&computeCalls, 1)
		return 42, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls), "Should recompute when cache type is wrong")
}

func TestFileCacheEmptyFingerprint(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](ctx, tempDir, 60)
	require.NoError(t, err)

	// Enable cache for this test
	cache.cacheEnabled = true

	// Empty struct fingerprint is valid
	fingerprint := struct{}{}

	var computeCalls int32
	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "value", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "value", result)

	// Second call should use cache
	result2, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "should-not-be-called", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "value", result2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls), "Empty fingerprint should work with cache")
}

func TestFileCacheMeasurementMode(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](ctx, tempDir, 60)
	require.NoError(t, err)

	// Keep cache disabled (measurement mode)
	cache.cacheEnabled = false

	fingerprint := struct {
		Key string `json:"key"`
	}{
		Key: "test-measurement",
	}

	// First call
	var computeCalls int32
	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "computed-value", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "computed-value", result)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls))

	// Second call - in measurement mode, should always recompute
	result2, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "recomputed-value", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "recomputed-value", result2)
	assert.Equal(t, int32(2), atomic.LoadInt32(&computeCalls), "Measurement mode should always recompute")

	// But cache file should still exist
	cacheFiles, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	require.NoError(t, err)
	assert.Len(t, cacheFiles, 1, "Cache file should be written even in measurement mode")
}

func TestFileCacheReadPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](ctx, tempDir, 60)
	require.NoError(t, err)

	// Enable cache for this test
	cache.cacheEnabled = true

	fingerprint := struct {
		Key string `json:"key"`
	}{
		Key: "test-permissions",
	}

	// First, populate the cache
	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		return "cached-value", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "cached-value", result)

	// Find the cache file and make it unreadable
	cacheFiles, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	require.NoError(t, err)
	require.Len(t, cacheFiles, 1)
	err = os.Chmod(cacheFiles[0], 0o000)
	require.NoError(t, err)

	// Restore permissions after test
	defer func() { _ = os.Chmod(cacheFiles[0], 0o600) }()

	// GetOrCompute should fail open and recompute when file is unreadable
	var computeCalls int32
	result2, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "recomputed-value", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "recomputed-value", result2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls), "Should recompute when cache file is unreadable")
}
