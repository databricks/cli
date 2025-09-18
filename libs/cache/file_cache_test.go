package cache

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileCache(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := NewFileCache(cacheDir)
	require.NoError(t, err)
	assert.NotNil(t, cache)
	assert.Equal(t, cacheDir, cache.baseDir)
	assert.NotNil(t, cache.pending)

	// Verify directory was created
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestNewFileCacheWithExistingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "existing")

	// Create directory first
	err := os.MkdirAll(cacheDir, 0700)
	require.NoError(t, err)

	cache, err := NewFileCache(cacheDir)
	require.NoError(t, err)
	assert.NotNil(t, cache)
	assert.Equal(t, cacheDir, cache.baseDir)
}

func TestNewFileCacheInvalidPath(t *testing.T) {
	// Try to create cache in a location that should fail
	invalidPath := "/root/invalid/path/that/should/not/exist"

	cache, err := NewFileCache(invalidPath)
	if err != nil {
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "failed to create cache directory")
	}
}

func TestFileCacheGetOrCompute(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := NewFileCache(tempDir)
	require.NoError(t, err)

	fingerprint := "test-key"
	expectedValue := "computed-value"

	// First call should compute the value
	var computeCalls int32
	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (any, error) {
		atomic.AddInt32(&computeCalls, 1)
		return expectedValue, nil
	})

	require.NoError(t, err)
	assert.Equal(t, expectedValue, result)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls))

	// Second call should return cached value
	result2, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (any, error) {
		atomic.AddInt32(&computeCalls, 1)
		return "should-not-be-called", nil
	})

	require.NoError(t, err)
	assert.Equal(t, expectedValue, result2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls)) // Should still be 1
}

func TestFileCacheGetOrComputeError(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := NewFileCache(tempDir)
	require.NoError(t, err)

	fingerprint := "error-key"

	// Compute function returns error
	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (any, error) {
		return nil, assert.AnError
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestFileCacheGetOrComputeConcurrency(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cache, err := NewFileCache(tempDir)
	require.NoError(t, err)

	fingerprint := "concurrent-key"
	expectedValue := "concurrent-value"
	var computeCalls int32

	// Start multiple goroutines that try to compute the same key
	numGoroutines := 10
	results := make(chan any, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (any, error) {
				atomic.AddInt32(&computeCalls, 1)
				time.Sleep(10 * time.Millisecond) // Simulate work
				return expectedValue, nil
			})
			results <- result
			errors <- err
		}()
	}

	// Collect all results
	for i := 0; i < numGoroutines; i++ {
		result := <-results
		err := <-errors
		require.NoError(t, err)
		assert.Equal(t, expectedValue, result)
	}

	// Compute should have been called only once despite multiple concurrent requests
	assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls))
}

func TestFileCacheGetOrComputeContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewFileCache(tempDir)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	fingerprint := "cancelled-key"

	result, err := cache.GetOrCompute(ctx, fingerprint, func(ctx context.Context) (any, error) {
		return "should-not-be-reached", nil
	})

	assert.Nil(t, result)
	assert.Equal(t, context.Canceled, err)
}
