package cache

import (
	"context"
	"encoding/json"
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
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := newFileCacheWithBaseDir[string](cacheDir, 60)
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
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "existing")

	// Create directory first
	err := os.MkdirAll(cacheDir, 0o700)
	require.NoError(t, err)

	cache, err := newFileCacheWithBaseDir[string](cacheDir, 60) // 1 hour for tests
	require.NoError(t, err)
	assert.NotNil(t, cache)
	assert.Equal(t, cacheDir, cache.baseDir)
}

func TestNewFileCacheInvalidPath(t *testing.T) {
	// Try to create cache in a location that should fail
	invalidPath := "/root/invalid/path/that/should/not/exist"

	cache, err := newFileCacheWithBaseDir[string](invalidPath, 60) // 1 hour for tests
	if err != nil {
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "failed to create cache directory")
	}
}

func TestFileCacheGetOrCompute(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](tempDir, 60) // 1 hour for tests
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
	cache, err := newFileCacheWithBaseDir[string](tempDir, 60) // 1 hour for tests
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
	cache, err := newFileCacheWithBaseDir[string](tempDir, 60) // 1 hour for tests
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

	// With locking, compute should only be called once even with concurrent requests
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls))
}

func TestFileCacheCleanupExpiredFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create some cache files manually - one expired, one valid, one corrupted
	now := time.Now()

	// Expired file
	expiredEntry := cacheEntry{
		Data:   json.RawMessage(`"expired-value"`),
		Expiry: now.Add(-time.Hour), // Expired 1 hour ago
	}
	expiredData, err := json.Marshal(expiredEntry)
	require.NoError(t, err)
	expiredFile := filepath.Join(tempDir, "expired.json")
	require.NoError(t, os.WriteFile(expiredFile, expiredData, 0o644))

	// Valid file
	validEntry := cacheEntry{
		Data:   json.RawMessage(`"valid-value"`),
		Expiry: now.Add(time.Hour), // Expires in 1 hour
	}
	validData, err := json.Marshal(validEntry)
	require.NoError(t, err)
	validFile := filepath.Join(tempDir, "valid.json")
	require.NoError(t, os.WriteFile(validFile, validData, 0o644))

	// Corrupted file
	corruptedFile := filepath.Join(tempDir, "corrupted.json")
	require.NoError(t, os.WriteFile(corruptedFile, []byte("invalid json"), 0o644))

	// Non-cache file (should be ignored)
	nonCacheFile := filepath.Join(tempDir, "readme.txt")
	require.NoError(t, os.WriteFile(nonCacheFile, []byte("readme"), 0o644))

	// Create cache - this should trigger cleanup
	_, err = newFileCacheWithBaseDir[string](tempDir, 60)
	require.NoError(t, err)

	// Check results
	_, err = os.Stat(expiredFile)
	assert.True(t, os.IsNotExist(err), "Expired file should be deleted")

	_, err = os.Stat(validFile)
	assert.False(t, os.IsNotExist(err), "Valid file should still exist")

	_, err = os.Stat(corruptedFile)
	assert.True(t, os.IsNotExist(err), "Corrupted file should be deleted")

	_, err = os.Stat(nonCacheFile)
	assert.False(t, os.IsNotExist(err), "Non-cache file should be ignored")
}

func TestFingerprintDeterministic(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](tempDir, 60)
	require.NoError(t, err)

	// Create two identical structs with fields in different JSON order
	fingerprint1 := struct {
		A string `json:"a"`
		B int    `json:"b"`
		C bool   `json:"c"`
	}{
		A: "value1",
		B: 42,
		C: true,
	}

	fingerprint2 := struct {
		C bool   `json:"c"`
		A string `json:"a"`
		B int    `json:"b"`
	}{
		C: true,
		A: "value1",
		B: 42,
	}

	expectedValue := "deterministic-value"
	var computeCalls int32

	// First call with fingerprint1
	result1, err := cache.GetOrCompute(ctx, fingerprint1, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return expectedValue, nil
	})
	require.NoError(t, err)
	assert.Equal(t, expectedValue, result1)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls))

	// Second call with fingerprint2 (should hit cache due to deterministic hashing, not compute again)
	result2, err := cache.GetOrCompute(ctx, fingerprint2, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "should-not-be-called", nil
	})
	require.NoError(t, err)

	assert.Equal(t, expectedValue, result2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls)) // Should still be 1
}
