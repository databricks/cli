package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

// Metrics is a local interface for tracking cache telemetry.
type Metrics interface {
	SetBoolValue(key string, value bool)
	SetDurationValue(key string, value time.Duration)
}

// FileCache implements the Cache interface using local disk storage.
type FileCache[T any] struct {
	baseDir       string
	expiryMinutes int
	mu            sync.Mutex
	metrics       Metrics
	cacheEnabled  bool // If true, cached values are returned; if false, cache is only used for measurement
}

// newFileCacheWithBaseDir creates a new file-based cache that stores data in the specified directory.
func newFileCacheWithBaseDir[T any](ctx context.Context, baseDir string, expiryMinutes int) (*FileCache[T], error) {
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	fc := &FileCache[T]{
		baseDir:       baseDir,
		expiryMinutes: expiryMinutes,
	}

	// Clean up expired files synchronously
	fc.cleanupExpiredFiles(ctx)

	return fc, nil
}

// isExpired checks if a file with the given modification time has expired.
func (fc *FileCache[T]) isExpired(modTime time.Time) bool {
	expiryThreshold := time.Now().Add(-time.Duration(fc.expiryMinutes) * time.Minute)
	return modTime.Before(expiryThreshold)
}

// cleanupExpiredFiles removes expired cache files from disk based on file modification time.
// This runs synchronously once when the cache is created.
// Files older than expiryMinutes are deleted.
func (fc *FileCache[T]) cleanupExpiredFiles(ctx context.Context) {
	err := filepath.Walk(fc.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log walk errors but continue cleanup
			log.Debugf(ctx, "[Local Cache] cleanup: failed to access path %s: %v", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Remove any leftover .tmp files (from failed atomic writes)
		if filepath.Ext(info.Name()) == ".tmp" {
			_ = os.Remove(path)
			return nil
		}

		// Only process .json cache files
		if filepath.Ext(info.Name()) != ".json" {
			return nil
		}

		// Check if file is expired based on modification time
		if fc.isExpired(info.ModTime()) {
			if err := os.Remove(path); err != nil {
				log.Tracef(ctx, "[Local Cache] cleanup: failed to remove expired file %s: %v", path, err)
			} else {
				log.Tracef(ctx, "[Local Cache] cleanup: removed expired file %s", path)
			}
		}

		return nil
	})
	if err != nil {
		log.Debugf(ctx, "[Local Cache] cleanup: failed to walk cache directory: %v", err)
	}
}

func getCacheBaseDir(ctx context.Context) (string, error) {
	// Check if user has configured a custom cache directory
	if customCacheDir := env.Get(ctx, "DATABRICKS_CACHE_DIR"); customCacheDir != "" {
		return customCacheDir, nil
	}

	// Use default cache directory
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache directory: %w", err)
	}
	return filepath.Join(userCacheDir, "databricks"), nil
}

// sanitizeVersion removes characters from version string that might be problematic in file paths.
// Particularly important for Windows which has restrictions on certain characters.
func sanitizeVersion(version string) string {
	// Replace + with - (used in version metadata like "1.0.0+abc123")
	version = strings.ReplaceAll(version, "+", "-")
	// Remove any other potentially problematic characters
	version = strings.ReplaceAll(version, ":", "-")
	version = strings.ReplaceAll(version, "/", "-")
	version = strings.ReplaceAll(version, "\\", "-")
	return version
}

// NewCache creates a new file-based cache using UserCacheDir() + "databricks" + version + cached component name.
// Including the CLI version in the path ensures cache isolation across different CLI versions.
// By default, the cache operates in measurement-only mode (cacheEnabled=false), which means it will:
// - Check if cached values exist
// - Measure how much time would have been saved
// - Emit metrics about potential savings
// - Always compute the value (never actually use the cache)
// Set DATABRICKS_CACHE_ENABLED=true to enable actual caching.
func NewCache[T any](ctx context.Context, component string, expiryMinutes int, metrics Metrics) Cache[T] {
	cacheBaseDir, err := getCacheBaseDir(ctx)
	if err != nil {
		return &NoopFileCache[T]{}
	}

	// Include CLI version in cache path to avoid issues across versions
	// Sanitize version string for use in file paths
	version := sanitizeVersion(build.GetInfo().Version)
	baseDir := filepath.Join(cacheBaseDir, version, component)
	fc, err := newFileCacheWithBaseDir[T](ctx, baseDir, expiryMinutes)
	if err != nil {
		return &NoopFileCache[T]{}
	}
	fc.metrics = metrics

	// Check if cache is enabled; default is false (measurement-only mode)
	fc.cacheEnabled = env.Get(ctx, "DATABRICKS_CACHE_ENABLED") == "true"
	return fc
}

func (fc *FileCache[T]) addTelemetryMetric(key string) {
	if fc.metrics != nil {
		fc.metrics.SetBoolValue(key, true)
	}
}

// GetOrCompute retrieves cached content or computes it using the provided function.
// Cache operations fail open: if caching fails, the compute function is still called.
// When cacheEnabled is false, the cache checks if values exist and measures potential time savings,
// but always computes and never returns cached values.
func (fc *FileCache[T]) GetOrCompute(ctx context.Context, fingerprint any, compute func(ctx context.Context) (T, error)) (T, error) {
	// Convert fingerprint to deterministic hash - this is our cache key
	cacheKey, err := fingerprintToHash(fingerprint)
	if err != nil {
		// Fail open: if we can't generate cache key, just compute directly
		log.Debugf(ctx, "[Local Cache] failed to generate cache key, computing without cache: %v", err)
		return compute(ctx)
	}

	log.Debugf(ctx, "[Local Cache] using cache key: %s", cacheKey)
	fc.addTelemetryMetric("local.cache.attempt")

	cachePath := fc.getCachePath(cacheKey)

	// Try to read from disk cache
	cachedData, cacheExists := fc.readFromCache(ctx, cachePath)

	// Record metrics
	if cacheExists {
		log.Debugf(ctx, "[Local Cache] cache hit")
		fc.addTelemetryMetric("local.cache.hit")

		// If cache is enabled, return the cached value
		if fc.cacheEnabled {
			return cachedData, nil
		}
	} else {
		log.Debugf(ctx, "[Local Cache] cache miss, computing")
		fc.addTelemetryMetric("local.cache.miss")
	}

	// Acquire lock to prevent concurrent computations and writes for the same cache key
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Compute the value and measure timing
	start := time.Now()
	result, err := compute(ctx)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] error while computing: %v", err)
		fc.addTelemetryMetric("local.cache.error")
		return result, err
	}

	// Record duration metrics
	if fc.metrics != nil {
		computeDuration := time.Since(start)
		fc.metrics.SetDurationValue("local.cache.compute_duration", computeDuration)
	}

	log.Debugf(ctx, "[Local Cache] computed and stored result")

	// Write to disk cache (failures are silent - cache write errors don't affect the result)
	fc.writeToCache(ctx, cachePath, result)

	return result, nil
}

// readFromCache attempts to read and deserialize data from the cache file.
// Expiry is checked using file modification time for consistency with cleanup.
func (fc *FileCache[T]) readFromCache(ctx context.Context, cachePath string) (T, bool) {
	var zero T

	// Check file modification time for expiry
	info, err := os.Stat(cachePath)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] failed to stat cache file: %v", err)
		return zero, false
	}

	if fc.isExpired(info.ModTime()) {
		return zero, false
	}

	// Read and deserialize the data
	data, err := os.ReadFile(cachePath)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] failed to read cache file: %v", err)
		return zero, false
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		log.Debugf(ctx, "[Local Cache] failed to deserialize data: %v", err)
		return zero, false
	}

	return result, true
}

// writeToCache serializes and writes data to the cache file atomically.
// Uses atomic write: writes to temp file first, then renames to actual cache file.
func (fc *FileCache[T]) writeToCache(ctx context.Context, cachePath string, data any) {
	// Serialize the data directly
	serializedData, err := json.Marshal(data)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] failed to serialize data: %v", err)
		return // Silently fail on serialization errors
	}

	// Create temporary file in the same directory for atomic operation
	tempFile, err := os.CreateTemp(fc.baseDir, ".cache-*.tmp")
	if err != nil {
		log.Debugf(ctx, "[Local Cache] failed to create temp cache file: %v", err)
		return
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath) // Clean up temp file if still exists
	}()

	// Write data to temp file
	if _, err := tempFile.Write(serializedData); err != nil {
		log.Debugf(ctx, "[Local Cache] failed to write to temp cache file: %v", err)
		return
	}

	if err := tempFile.Close(); err != nil {
		log.Debugf(ctx, "[Local Cache] failed to close temp cache file: %v", err)
		return
	}

	// On Windows, os.Rename fails if target exists, so remove it first
	// This is a best-effort operation - if it fails because file doesn't exist, that's fine
	_ = os.Remove(cachePath)

	// Atomically rename temp file to actual cache file
	if err := os.Rename(tempPath, cachePath); err != nil {
		log.Debugf(ctx, "[Local Cache] failed to rename temp cache file: %v", err)
	}
}

// getCachePath returns the full path to the cache file for a given cache key.
func (fc *FileCache[T]) getCachePath(cacheKey string) string {
	return filepath.Join(fc.baseDir, cacheKey+".json")
}
