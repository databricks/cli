package cache

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	pending       map[string]chan struct{} // Track pending writes
	memCache      map[string]T             // In-memory cache for immediate access
	cleanupMgr    *CleanupManager          // Background cleanup manager
	metrics       *bundle.Metrics          // Telemetry metrics
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
		pending:       make(map[string]chan struct{}),
		memCache:      make(map[string]T),
		cleanupMgr:    cleanupMgr,
	}

	// Start background cleanup (non-blocking)
	cleanupMgr.Start(context.Background(), baseDir)

	return fc, nil
}

// NewFileCache creates a new file-based cache using UserCacheDir() + "databricks" + cached component name.
func NewFileCache[T any](component string, expiryMinutes int, metrics *bundle.Metrics) (*FileCache[T], error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user cache directory: %w", err)
	}

	baseDir := filepath.Join(userCacheDir, "databricks", component)
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

	// Convert fingerprint to deterministic string
	fingerprintHash, err := fingerprintToHash(fingerprint)
	log.Debugf(ctx, "[Local Cache] using fingerprint with hash: %s\n", fingerprintHash)

	fc.addTelemetryMetric("local.cache.attempt")

	if err != nil {
		log.Debugf(ctx, "[Local Cache] cache miss: non-compliant fingerprint\n")
		return zero, fmt.Errorf("failed to convert fingerprint to string: %w", err)
	}

	cacheKey := fc.getCacheKey(fingerprintHash)
	log.Debugf(ctx, "[Local Cache] using cache key: %s\n", cacheKey)

	cachePath := fc.getCachePath(cacheKey)
	log.Debugf(ctx, "[Local Cache] using cache path: %s\n", cachePath)

	// Check in-memory cache first
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

	// Check if there's a pending write for this key
	fc.mu.Lock()
	if pendingCh, exists := fc.pending[cacheKey]; exists {
		fc.mu.Unlock()
		// Wait for pending write to complete
		select {
		case <-pendingCh:
			// Try reading from memory cache again
			fc.mu.RLock()
			if data, found := fc.memCache[cacheKey]; found {
				fc.mu.RUnlock()
				log.Debugf(ctx, "[Local Cache] cache hit: in-memory from pending write\n")
				fc.addTelemetryMetric("local.cache.hit")
				return data, nil
			}
			fc.mu.RUnlock()
		case <-ctx.Done():
			log.Debugf(ctx, "[Local Cache] cache miss: no hit while waiting for pending write\n")
			fc.addTelemetryMetric("local.cache.miss")
			return zero, ctx.Err()
		}
	} else {
		// Mark this key as pending
		pendingCh := make(chan struct{})
		fc.pending[cacheKey] = pendingCh
		fc.mu.Unlock()

		defer func() {
			fc.mu.Lock()
			delete(fc.pending, cacheKey)
			close(pendingCh)
			fc.mu.Unlock()
		}()
	}

	// Check if context is already cancelled before computing
	select {
	case <-ctx.Done():
		log.Debugf(ctx, "[Local Cache] cache miss: context is already cancelled\n")
		fc.addTelemetryMetric("local.cache.miss")
		return zero, ctx.Err()
	default:
	}

	// Compute the value
	result, err := compute(ctx)
	if err != nil {
		log.Debugf(ctx, "[Local Cache] error while caching: %v\n", err)
		fc.addTelemetryMetric("local.cache.error")
		return zero, err
	}

	// Store in memory cache immediately
	fc.mu.Lock()
	fc.memCache[cacheKey] = result
	fc.mu.Unlock()

	// Async write to disk cache
	log.Debugf(ctx, "[Local Cache] async writing to cache path: %s\n", cachePath)
	go fc.writeToCache(cachePath, result)

	log.Debugf(ctx, "[Local Cache] cache miss, but stored the compute result for future calls\n")
	fc.addTelemetryMetric("local.cache.miss")
	return result, nil
}

// readFromCache attempts to read and deserialize data from the cache file.
func (fc *FileCache[T]) readFromCache(cachePath string) (T, bool) {
	var zero T

	fc.mu.RLock()
	defer fc.mu.RUnlock()

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return zero, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
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

// getCacheKey generates a safe cache key from the fingerprint.
func (fc *FileCache[T]) getCacheKey(fingerprint string) string {
	hash := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(hash[:])
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
