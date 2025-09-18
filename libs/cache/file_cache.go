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
)

// FileCache implements the Cache interface using local disk storage.
type FileCache struct {
	baseDir  string
	mu       sync.RWMutex
	pending  map[string]chan struct{} // Track pending writes
	memCache map[string]any           // In-memory cache for immediate access
}

// NewFileCache creates a new file-based cache that stores data in the specified directory.
func NewFileCache(baseDir string) (*FileCache, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &FileCache{
		baseDir:  baseDir,
		pending:  make(map[string]chan struct{}),
		memCache: make(map[string]any),
	}, nil
}

// cacheEntry represents the structure of a cached item on disk.
type cacheEntry struct {
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

// GetOrCompute retrieves cached content or computes it using the provided function.
func (fc *FileCache) GetOrCompute(ctx context.Context, fingerprint any, compute func(ctx context.Context) (any, error)) (any, error) {
	// Convert fingerprint to deterministic string
	fingerprintStr, err := FingerprintToString(fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to convert fingerprint to string: %w", err)
	}

	cacheKey := fc.getCacheKey(fingerprintStr)
	cachePath := fc.getCachePath(cacheKey)

	// Check in-memory cache first
	fc.mu.RLock()
	if data, found := fc.memCache[cacheKey]; found {
		fc.mu.RUnlock()
		return data, nil
	}
	fc.mu.RUnlock()

	// Try to read from disk cache
	if data, found := fc.readFromCache(cachePath); found {
		// Store in memory cache for faster future access
		fc.mu.Lock()
		fc.memCache[cacheKey] = data
		fc.mu.Unlock()
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
				return data, nil
			}
			fc.mu.RUnlock()
		case <-ctx.Done():
			return nil, ctx.Err()
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
		return nil, ctx.Err()
	default:
	}

	// Compute the value
	result, err := compute(ctx)
	if err != nil {
		return nil, err
	}

	// Store in memory cache immediately
	fc.mu.Lock()
	fc.memCache[cacheKey] = result
	fc.mu.Unlock()

	// Async write to disk cache
	go fc.writeToCache(cachePath, result)

	return result, nil
}

// readFromCache attempts to read and deserialize data from the cache file.
func (fc *FileCache) readFromCache(cachePath string) (any, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	var result any
	if err := json.Unmarshal(entry.Data, &result); err != nil {
		return nil, false
	}

	return result, true
}

// writeToCache serializes and writes data to the cache file asynchronously.
func (fc *FileCache) writeToCache(cachePath string, data any) {
	// Serialize the data
	serializedData, err := json.Marshal(data)
	if err != nil {
		return // Silently fail on serialization errors
	}

	entry := cacheEntry{
		Data:      serializedData,
		Timestamp: time.Now(),
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
func (fc *FileCache) getCacheKey(fingerprint string) string {
	hash := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(hash[:])
}

// getCachePath returns the full path to the cache file for a given cache key.
func (fc *FileCache) getCachePath(cacheKey string) string {
	// Create subdirectories based on first 2 characters for better file distribution
	subDir := cacheKey[:2]
	return filepath.Join(fc.baseDir, subDir, cacheKey+".json")
}
