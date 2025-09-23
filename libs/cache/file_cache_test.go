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
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := newFileCacheWithBaseDir[string](cacheDir)
	require.NoError(t, err)
	assert.NotNil(t, cache)
	assert.Equal(t, cacheDir, cache.baseDir)
	assert.NotNil(t, cache.pending)

	// Verify directory was created
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Check permissions - Windows has different permission semantics
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	} else {
		// On Windows, verify directory is accessible by trying to create a test file
		testFile := filepath.Join(cacheDir, "test_access")
		err := os.WriteFile(testFile, []byte("test"), 0o644)
		assert.NoError(t, err)
		if err == nil {
			_ = os.Remove(testFile) // Clean up (ignore removal error)
		}
	}
}

func TestNewFileCacheWithExistingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "existing")

	// Create directory first
	err := os.MkdirAll(cacheDir, 0o700)
	require.NoError(t, err)

	cache, err := newFileCacheWithBaseDir[string](cacheDir)
	require.NoError(t, err)
	assert.NotNil(t, cache)
	assert.Equal(t, cacheDir, cache.baseDir)
}

func TestNewFileCacheInvalidPath(t *testing.T) {
	// Try to create cache in a location that should fail
	invalidPath := "/root/invalid/path/that/should/not/exist"

	cache, err := newFileCacheWithBaseDir[string](invalidPath)
	if err != nil {
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "failed to create cache directory")
	}
}

func TestFileCacheGetOrCompute(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](tempDir)
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

	// Second call should return cached value
	result2, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "should-not-be-called", nil
	})

	require.NoError(t, err)
	assert.Equal(t, expectedValue, result2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls)) // Should still be 1

	// Allow time for async writes to complete before test cleanup
	time.Sleep(50 * time.Millisecond)
}

func TestFileCacheGetOrComputeError(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](tempDir)
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
	cache, err := newFileCacheWithBaseDir[string](tempDir)
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

	// Compute should have been called only once despite multiple concurrent requests
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls))

	// Allow time for async writes to complete before test cleanup
	time.Sleep(50 * time.Millisecond)
}

func TestFileCacheGetOrComputeContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](tempDir)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	fingerprint := struct {
		Key string `json:"key"`
	}{
		Key: "cancelled-key",
	}

	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (string, error) {
		return "should-not-be-reached", nil
	})

	assert.Empty(t, result)
	assert.Equal(t, context.Canceled, err)
}

func TestFingerprintDeterministic(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := newFileCacheWithBaseDir[string](tempDir)
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

	// Second call with fingerprint2 (should hit cache, not compute again)
	result2, err := cache.GetOrCompute(ctx, fingerprint2, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "should-not-be-called", nil
	})
	require.NoError(t, err)
	assert.Equal(t, expectedValue, result2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls)) // Should still be 1

	// Allow time for async writes to complete before test cleanup
	time.Sleep(50 * time.Millisecond)
}
