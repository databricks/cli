package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// cleanupExpiredFiles removes expired cache files from disk.
// This runs synchronously once when the cache is created.
func (fc *FileCache[T]) cleanupExpiredFiles() {
	now := time.Now()

	_ = filepath.Walk(fc.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Only process .json cache files
		if filepath.Ext(info.Name()) != ".json" {
			return nil
		}

		// Try to read the cache entry
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var entry cacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			// Delete corrupted files
			_ = os.Remove(path)
			return nil
		}

		// Delete if expired
		if !entry.Expiry.IsZero() && now.After(entry.Expiry) {
			_ = os.Remove(path)
		}

		return nil
	})
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

// NewFileCache creates a new file-based cache using UserCacheDir() + "databricks" + version + cached component name.
// Including the CLI version in the path ensures cache isolation across different CLI versions.
func NewFileCache[T any](component string, expiryMinutes int, metrics Metrics) (*FileCache[T], error) {
	cacheBaseDir, err := getCacheBaseDir()
	if err != nil {
		return nil, err
	}

	// Include CLI version in cache path to avoid issues across versions
	version := build.GetInfo().Version
	baseDir := filepath.Join(cacheBaseDir, version, component)
	fc, err := newFileCacheWithBaseDir[T](baseDir, expiryMinutes)
	if err != nil {
		return nil, err
	}
	fc.metrics = metrics
	return fc, nil
}

// cacheEntry represents the structure of a cached item on disk.
type cacheEntry struct {
	Data   json.RawMessage `json:"data"`
	Expiry time.Time       `json:"expiry"`
}

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
func (fc *FileCache[T]) readFromCache(cachePath string) (T, bool) {
	var zero T

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return zero, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return zero, false
	}

	// Check if cache entry has expired
	if time.Now().After(entry.Expiry) {
		return zero, false
	}

	var result T
	if err := json.Unmarshal(entry.Data, &result); err != nil {
		return zero, false
	}

	return result, true
}

// writeToCache serializes and writes data to the cache file.
func (fc *FileCache[T]) writeToCache(cachePath string, data any) {
	// Serialize the data
	serializedData, err := json.Marshal(data)
	if err != nil {
		return // Silently fail on serialization errors
	}

	entry := cacheEntry{
		Data:   serializedData,
		Expiry: time.Now().Add(time.Duration(fc.expiryMinutes) * time.Minute),
	}

	entryData, err := json.Marshal(entry)
	if err != nil {
		return // Silently fail on serialization errors
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
		return
	}

	// Write to cache file
	_ = os.WriteFile(cachePath, entryData, 0o600)
}

// getCachePath returns the full path to the cache file for a given cache key.
func (fc *FileCache[T]) getCachePath(cacheKey string) string {
	return filepath.Join(fc.baseDir, cacheKey+".json")
}
