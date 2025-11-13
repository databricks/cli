package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/databricks/cli/bundle"

	"github.com/databricks/cli/libs/log"
)

// FileCache implements the Cache interface using local disk storage.
type FileCache[T any] struct {
	baseDir       string
	expiryMinutes int
	mu            sync.RWMutex
	computeOnce   map[string]*sync.Once // Ensure only one goroutine computes per key
	memCache      map[string]T          // In-memory cache for immediate access
	cleanupMgr    *CleanupManager       // Background cleanup manager
	metrics       *bundle.Metrics       // Telemetry metrics
}

// newFileCacheWithBaseDir creates a new file-based cache that stores data in the specified directory.
func newFileCacheWithBaseDir[T any](baseDir string, expiryMinutes int) (*FileCache[T], error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cleanupMgr := NewCleanupManager(DefaultCleanupConfig())

	fc := &FileCache[T]{
		baseDir:       baseDir,
		expiryMinutes: expiryMinutes,
		computeOnce:   make(map[string]*sync.Once),
		memCache:      make(map[string]T),
		cleanupMgr:    cleanupMgr,
	}

	// Start background cleanup (non-blocking)
	cleanupMgr.Start(context.Background(), baseDir)

	return fc, nil
}

func getCacheBaseDir() (string, error) {
	// Check if user has configured a custom cache directory
	if customCacheDir := os.Getenv("DATABRICKS_CACHE_FOLDER"); customCacheDir != "" {
		return customCacheDir, nil
	}

	// Use default cache directory
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache directory: %w", err)
	}
	return filepath.Join(userCacheDir, "databricks"), nil
}

// NewFileCache creates a new file-based cache using UserCacheDir() + "databricks" + cached component name.
func NewFileCache[T any](component string, expiryMinutes int, metrics *bundle.Metrics) (*FileCache[T], error) {
	cacheBaseDir, err := getCacheBaseDir()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(cacheBaseDir, component)
	fc, err := newFileCacheWithBaseDir[T](baseDir, expiryMinutes)
	if err != nil {
		return nil, err
	}
	fc.metrics = metrics
	return fc, nil
}

// cacheEntry represents the structure of a cached item on disk.
type cacheEntry struct {
	Data      json.RawMessage `json:"data"`
	Expiry    time.Time       `json:"expiry"`
	Timestamp time.Time       `json:"timestamp,omitempty"` // For backward compatibility
}

func (fc *FileCache[T]) addTelemetryMetric(key string) {
	if fc.metrics != nil {
		fc.metrics.SetBoolValue(key, true)
	}
}

// GetOrCompute retrieves cached content or computes it using the provided function.
func (fc *FileCache[T]) GetOrCompute(ctx context.Context, fingerprint any, compute func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	// Convert fingerprint to deterministic hash - this is our cache key
	cacheKey, err := fingerprintToHash(fingerprint)
	if err != nil {
		return zero, fmt.Errorf("failed to convert fingerprint to string: %w", err)
	}

	log.Debugf(ctx, "[Local Cache] using cache key: %s\n", cacheKey)
	fc.addTelemetryMetric("local.cache.attempt")

	cachePath := fc.getCachePath(cacheKey)

	// Check in-memory cache first (fast path)
	fc.mu.RLock()
	if data, found := fc.memCache[cacheKey]; found {
		fc.mu.RUnlock()
		log.Debugf(ctx, "[Local Cache] cache hit: in-memory\n")
		fc.addTelemetryMetric("local.cache.hit")
		return data, nil
	}
	fc.mu.RUnlock()

	// Try to read from disk cache
	if data, found := fc.readFromCache(cachePath); found {
		// Store in memory cache for faster future access
		fc.mu.Lock()
		fc.memCache[cacheKey] = data
		fc.mu.Unlock()
		log.Debugf(ctx, "[Local Cache] cache hit: disk-read\n")
		fc.addTelemetryMetric("local.cache.hit")
		return data, nil
	}

	// Get or create sync.Once for this cache key
	// Check cache again under write lock to avoid race condition
	fc.mu.Lock()
	if data, found := fc.memCache[cacheKey]; found {
		fc.mu.Unlock()
		log.Debugf(ctx, "[Local Cache] cache hit: in-memory (race avoided)\n")
		fc.addTelemetryMetric("local.cache.hit")
		return data, nil
	}
	once, exists := fc.computeOnce[cacheKey]
	if !exists {
		once = &sync.Once{}
		fc.computeOnce[cacheKey] = once
	}
	fc.mu.Unlock()

	// Use sync.Once to ensure only one goroutine computes the value
	// Store error in a separate variable that all goroutines can access
	var computeErr error
	once.Do(func() {
		// Check if context is already cancelled before computing
		select {
		case <-ctx.Done():
			log.Debugf(ctx, "[Local Cache] context cancelled before compute\n")
			computeErr = ctx.Err()
			return
		default:
		}

		// Compute the value
		result, err := compute(ctx)
		if err != nil {
			log.Debugf(ctx, "[Local Cache] error while computing: %v\n", err)
			fc.addTelemetryMetric("local.cache.error")
			computeErr = err
			return
		}

		// Store in memory cache immediately
		fc.mu.Lock()
		fc.memCache[cacheKey] = result
		fc.mu.Unlock()

		// Write to disk cache synchronously to ensure it persists before process exits
		log.Debugf(ctx, "[Local Cache] writing to cache\n")
		fc.writeToCache(cachePath, result)

		log.Debugf(ctx, "[Local Cache] cache miss, computed and stored result\n")
		fc.addTelemetryMetric("local.cache.miss")
	})

	// Check if computation failed
	if computeErr != nil {
		return zero, computeErr
	}

	// All goroutines retrieve the result from memCache after sync.Once completes
	fc.mu.RLock()
	result, found := fc.memCache[cacheKey]
	fc.mu.RUnlock()

	if !found {
		// This should never happen unless there was an error
		return zero, errors.New("cache inconsistency: value not found after computation")
	}

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

// writeToCache serializes and writes data to the cache file asynchronously.
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
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return
	}

	// Write to temporary file first, then rename for atomic operation
	tempPath, err := generateTempPath(cachePath)
	if err != nil {
		return
	}

	if err := os.WriteFile(tempPath, entryData, 0o644); err != nil {
		return
	}

	// Atomic rename
	_ = os.Rename(tempPath, cachePath)
}

// generateTempPath creates a temporary file path with a random component to prevent collisions.
func generateTempPath(cachePath string) (string, error) {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	randomSuffix := hex.EncodeToString(randomBytes)
	return cachePath + ".tmp." + randomSuffix, nil
}

// getCachePath returns the full path to the cache file for a given cache key.
func (fc *FileCache[T]) getCachePath(cacheKey string) string {
	return filepath.Join(fc.baseDir, cacheKey+".json")
}

// StopCleanup stops the background cleanup process.
// This is non-blocking and will not wait for cleanup to complete.
func (fc *FileCache[T]) StopCleanup() {
	if fc.cleanupMgr != nil {
		fc.cleanupMgr.Stop()
	}
}
