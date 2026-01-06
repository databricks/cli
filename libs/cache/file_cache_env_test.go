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

func TestCacheEnabledEnvVar(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		envValue     string
		expectCached bool
	}{
		{
			name:         "cache enabled with 'true'",
			envValue:     "true",
			expectCached: true,
		},
		{
			name:         "cache disabled with 'false'",
			envValue:     "false",
			expectCached: false,
		},
		{
			name:         "cache enabled when empty (default)",
			envValue:     "",
			expectCached: true,
		},
		{
			name:         "cache enabled with other value",
			envValue:     "yes",
			expectCached: true,
		},
		{
			name:         "cache enabled with '1'",
			envValue:     "1",
			expectCached: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique subdirectory for this test
			testDir := filepath.Join(tempDir, tt.name)

			// Set up context with environment variable
			testCtx := ctx
			if tt.envValue != "" {
				testCtx = env.Set(testCtx, "DATABRICKS_CACHE_ENABLED", tt.envValue)
			}
			testCtx = env.Set(testCtx, "DATABRICKS_CACHE_DIR", testDir)

			// Use NewCache to properly initialize the cache
			cache := NewCache(testCtx, "test-component", 60*time.Minute, nil)

			fingerprint := struct {
				Key string `json:"key"`
			}{
				Key: "test-key",
			}

			// First call - should always compute
			var computeCalls int32
			result, err := GetOrCompute[string](testCtx, cache, fingerprint, func(ctx context.Context) (string, error) {
				atomic.AddInt32(&computeCalls, 1)
				return "computed-value", nil
			})
			require.NoError(t, err)
			assert.Equal(t, "computed-value", result)
			assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls))

			// Second call - should use cache only if enabled
			result2, err := GetOrCompute[string](testCtx, cache, fingerprint, func(ctx context.Context) (string, error) {
				atomic.AddInt32(&computeCalls, 1)
				return "should-not-be-called", nil
			})
			require.NoError(t, err)

			if tt.expectCached {
				// Cache enabled - should return cached value
				assert.Equal(t, "computed-value", result2)
				assert.Equal(t, int32(1), atomic.LoadInt32(&computeCalls), "Should not recompute when cache is enabled")
			} else {
				// Cache disabled - should recompute
				assert.Equal(t, "should-not-be-called", result2)
				assert.Equal(t, int32(2), atomic.LoadInt32(&computeCalls), "Should recompute when cache is disabled")
			}
		})
	}
}

func TestCacheDirEnvVar(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	t.Run("uses DATABRICKS_CACHE_DIR when set", func(t *testing.T) {
		customCacheDir := filepath.Join(tempDir, "custom-cache")
		t.Setenv("DATABRICKS_CACHE_DIR", customCacheDir)

		cache := NewCache(ctx, "test-component", 60*time.Minute, nil)
		fc, ok := cache.impl.(*fileCache)
		require.True(t, ok)

		// Verify the cache directory is under the custom path
		assert.Contains(t, fc.baseDir, customCacheDir)
		assert.Contains(t, fc.baseDir, "test-component")

		// Verify directory was created
		_, err := os.Stat(customCacheDir)
		assert.NoError(t, err, "Custom cache directory should be created")
	})

	t.Run("uses default UserCacheDir when env var not set", func(t *testing.T) {
		os.Unsetenv("DATABRICKS_CACHE_DIR")

		cache := NewCache(ctx, "test-component", 60*time.Minute, nil)
		fc, ok := cache.impl.(*fileCache)
		require.True(t, ok)

		// Verify it's using the default path structure
		userCacheDir, err := os.UserCacheDir()
		require.NoError(t, err)
		expectedPrefix := filepath.Join(userCacheDir, "databricks")

		assert.Contains(t, fc.baseDir, expectedPrefix)
	})

	t.Run("handles invalid cache dir path", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping invalid path test on Windows - permission model differs")
		}

		// Set an invalid path (no permissions)
		t.Setenv("DATABRICKS_CACHE_DIR", "/root/invalid-cache-dir")

		cache := NewCache(ctx, "test-component", 60*time.Minute, nil)
		_, ok := cache.impl.(*noopFileCache)
		require.True(t, ok)
	})
}

func TestCacheIsolationByVersion(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	t.Setenv("DATABRICKS_CACHE_DIR", tempDir)

	// Create cache for component
	cache := NewCache(ctx, "test-component", 60*time.Minute, nil)
	fc, ok := cache.impl.(*fileCache)
	require.True(t, ok)

	// Verify the cache path structure: <cache-base>/<version>/<component>
	// The path should contain the component name
	assert.Contains(t, fc.baseDir, "test-component")

	// The path should be a subdirectory of tempDir
	assert.Contains(t, fc.baseDir, tempDir)

	// Verify there's at least one intermediate directory between tempDir and component
	// (the version directory)
	relativePath, err := filepath.Rel(tempDir, fc.baseDir)
	require.NoError(t, err)

	// Split by separator and count
	pathParts := filepath.SplitList(relativePath)
	// On most systems, SplitList is for PATH env var, not file paths
	// Use strings.Split instead
	if len(pathParts) == 1 {
		pathParts = strings.Split(relativePath, string(filepath.Separator))
	}

	// Should have at least 2 parts: <version>/<component>
	assert.GreaterOrEqual(t, len(pathParts), 2, "Cache path should include version directory: %s", relativePath)
}
