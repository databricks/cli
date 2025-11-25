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
	"github.com/databricks/cli/libs/log"
)

// Metrics is a local interface for tracking cache telemetry.
type Metrics interface {
	SetBoolValue(key string, value bool)
}

// FileCache implements the Cache interface using local disk storage.
type FileCache[T any] struct {
	baseDir       string
	expiryMinutes int
	mu            sync.Mutex
	metrics       Metrics
}

// newFileCacheWithBaseDir creates a new file-based cache that stores data in the specified directory.
func newFileCacheWithBaseDir[T any](baseDir string, expiryMinutes int) (*FileCache[T], error) {
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	fc := &FileCache[T]{
		baseDir:       baseDir,
		expiryMinutes: expiryMinutes,
	}

	// Clean up expired files synchronously
	fc.cleanupExpiredFiles()

	return fc, nil
}

// cleanupExpiredFiles removes expired cache files from disk based on file modification time.
// This runs synchronously once when the cache is created.
// Files older than expiryMinutes are deleted.
func (fc *FileCache[T]) cleanupExpiredFiles() {
	now := time.Now()
	expiryDuration := time.Duration(fc.expiryMinutes) * time.Minute

	err := filepath.Walk(fc.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log walk errors but continue cleanup
			log.Debugf(context.Background(), "[Local Cache] cleanup: failed to access path %s: %v", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Only process .json cache files
		if filepath.Ext(info.Name()) != ".json" {
			return nil
		}

		// Check if file is expired based on modification time
		age := now.Sub(info.ModTime())
		if age > expiryDuration {
			if err := os.Remove(path); err != nil {
				log.Debugf(context.Background(), "[Local Cache] cleanup: failed to remove expired file %s: %v", path, err)
			} else {
				log.Debugf(context.Background(), "[Local Cache] cleanup: removed expired file %s (age: %v)", path, age)
			}
		}

		return nil
	})
	if err != nil {
		log.Warnf(context.Background(), "[Local Cache] cleanup: failed to walk cache directory: %v", err)
	}
}

func getCacheBaseDir() (string, error) {
	// Check if user has configured a custom cache directory
	if customCacheDir := os.Getenv("DATABRICKS_CACHE_DIR"); customCacheDir != "" {
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

// NewFileCache creates a new file-based cache using UserCacheDir() + "databricks" + version + cached component name.
// Including the CLI version in the path ensures cache isolation across different CLI versions.
func NewFileCache[T any](component string, expiryMinutes int, metrics Metrics) (*FileCache[T], error) {
	cacheBaseDir, err := getCacheBaseDir()
	if err != nil {
		return nil, err
	}

	// Include CLI version in cache path to avoid issues across versions
	// Sanitize version string for use in file paths
	version := sanitizeVersion(build.GetInfo().Version)
	baseDir := filepath.Join(cacheBaseDir, version, component)
	fc, err := newFileCacheWithBaseDir[T](baseDir, expiryMinutes)
	if err != nil {
		return nil, err
	}
	fc.metrics = metrics
	return fc, nil
}

// Cache files are stored as JSON directly without metadata wrapper.
// Expiry is tracked using file modification time, not stored in the file itself.

func (fc *FileCache[T]) addTelemetryMetric(key string) {
	if fc.metrics != nil {
		fc.metrics.SetBoolValue(key, true)
	}
}

// GetOrCompute retrieves cached content or computes it using the provided function.
// Cache operations fail open: if caching fails, the compute function is still called.
func (fc *FileCache[T]) GetOrCompute(ctx context.Context, fingerprint any, compute func(ctx context.Context) (T, error)) (T, error) {
	// Convert fingerprint to deterministic hash - this is our cache key
	cacheKey, err := fingerprintToHash(fingerprint)
	if err != nil {
		// Fail open: if we can't generate cache key, just compute directly
		log.Debugf(ctx, "[Local Cache] failed to generate cache key, computing without cache: %v\n", err)
		return compute(ctx)
	}

	log.Debugf(ctx, "[Local Cache] using cache key: %s\n", cacheKey)
	fc.addTelemetryMetric("local.cache.attempt")

	cachePath := fc.getCachePath(cacheKey)

	// Try to read from disk cache
	if data, found := fc.readFromCache(cachePath); found {
		log.Debugf(ctx, "[Local Cache] cache hit\n")
		fc.addTelemetryMetric("local.cache.hit")
		return data, nil
	}

	// Cache miss - acquire lock to compute
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Check again after acquiring lock (another goroutine might have computed it)
	if data, found := fc.readFromCache(cachePath); found {
		log.Debugf(ctx, "[Local Cache] cache hit after lock\n")
		fc.addTelemetryMetric("local.cache.hit")
		return data, nil
	}

	// Compute the value
	log.Debugf(ctx, "[Local Cache] cache miss, computing\n")
	result, err := compute(ctx)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] error while computing: %v\n", err)
		fc.addTelemetryMetric("local.cache.error")
		return result, err
	}

	// Write to disk cache (failures are silent - cache write errors don't affect the result)
	fc.writeToCache(cachePath, result)
	log.Debugf(ctx, "[Local Cache] computed and stored result\n")
	fc.addTelemetryMetric("local.cache.miss")

	return result, nil
}

// readFromCache attempts to read and deserialize data from the cache file.
// Expiry is checked using file modification time for consistency with cleanup.
func (fc *FileCache[T]) readFromCache(cachePath string) (T, bool) {
	var zero T

	// Check file modification time for expiry
	info, err := os.Stat(cachePath)
	if err != nil {
		log.Debugf(context.Background(), "[Local Cache] failed to stat cache file: %v\n", err)
		return zero, false
	}

	age := time.Since(info.ModTime())
	expiryDuration := time.Duration(fc.expiryMinutes) * time.Minute
	if age > expiryDuration {
		return zero, false
	}

	// Read and deserialize the data
	data, err := os.ReadFile(cachePath)
	if err != nil {
		log.Debugf(context.Background(), "[Local Cache] failed to read cache file: %v\n", err)
		return zero, false
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		log.Debugf(context.Background(), "[Local Cache] failed to deserialize data: %v\n", err)
		return zero, false
	}

	return result, true
}

// writeToCache serializes and writes data to the cache file.
// Expiry is tracked by file modification time, not stored in the file.
func (fc *FileCache[T]) writeToCache(cachePath string, data any) {
	// Serialize the data directly
	serializedData, err := json.Marshal(data)
	if err != nil {
		log.Debugf(context.Background(), "[Local Cache] failed to serialize data: %v\n", err)
		return // Silently fail on serialization errors
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
		log.Debugf(context.Background(), "[Local Cache] failed to create directory: %v\n", err)
		return
	}

	// Write to cache file - the mtime will be used to track expiry
	err = os.WriteFile(cachePath, serializedData, 0o600)
	if err != nil {
		log.Debugf(context.Background(), "[Local Cache] failed to write to cache file: %v\n", err)
	}
}

// getCachePath returns the full path to the cache file for a given cache key.
func (fc *FileCache[T]) getCachePath(cacheKey string) string {
	return filepath.Join(fc.baseDir, cacheKey+".json")
}
