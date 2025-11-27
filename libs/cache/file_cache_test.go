package cache

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileCache(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := newFileCacheWithBaseDir[string](ctx, cacheDir, 60)
	require.NoError(t, err)
	assert.NotNil(t, cache)
	assert.Equal(t, cacheDir, cache.baseDir)

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

	cache, err := newFileCacheWithBaseDir[string](ctx, cacheDir, 60) // 1 hour for tests
	require.NoError(t, err)
	assert.NotNil(t, cache)
	assert.Equal(t, cacheDir, cache.baseDir)
}

func TestNewFileCacheInvalidPath(t *testing.T) {
	ctx := context.Background()
	// Try to create cache in a location that should fail
	invalidPath := "/root/invalid/path/that/should/not/exist"

	cache, err := newFileCacheWithBaseDir[string](ctx, invalidPath, 60) // 1 hour for tests
	if err != nil {
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "failed to create cache directory")
	}
}

func TestFileCacheGetOrCompute(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](ctx, tempDir, 60) // 1 hour for tests
	require.NoError(t, err)

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
	cache, err := newFileCacheWithBaseDir[string](ctx, tempDir, 60) // 1 hour for tests
	require.NoError(t, err)

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
	cache, err := newFileCacheWithBaseDir[string](ctx, tempDir, 60) // 1 hour for tests
	require.NoError(t, err)

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
