package cache

import (
	"context"
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
	baseDir string
	mu      sync.RWMutex
	pending map[string]chan struct{} // Track pending writes
}

// NewFileCache creates a new file-based cache that stores data in the specified directory.
func NewFileCache(baseDir string) (*FileCache, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &FileCache{
		baseDir: baseDir,
		pending: make(map[string]chan struct{}),
	}, nil
}

// cacheEntry represents the structure of a cached item on disk.
type cacheEntry struct {
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

// GetOrCompute retrieves cached content or computes it using the provided function.
func (fc *FileCache) GetOrCompute(ctx context.Context, fingerprint string, compute func(ctx context.Context) (any, error)) (any, error) {
	cacheKey := fc.getCacheKey(fingerprint)
	cachePath := fc.getCachePath(cacheKey)

	// Try to read from cache first
	if data, found := fc.readFromCache(cachePath); found {
		return data, nil
	}

	// Check if there's a pending write for this key
	fc.mu.Lock()
	if pendingCh, exists := fc.pending[cacheKey]; exists {
		fc.mu.Unlock()
		// Wait for pending write to complete
		select {
		case <-pendingCh:
			// Try reading again after write completes
			if data, found := fc.readFromCache(cachePath); found {
				return data, nil
			}
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

	// Compute the value
	result, err := compute(ctx)
	if err != nil {
		return nil, err
	}

	// Async write to cache
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
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return
	}

	// Write to temporary file first, then rename for atomic operation
	tempPath := cachePath + ".tmp"
	if err := os.WriteFile(tempPath, entryData, 0644); err != nil {
		return
	}

	// Atomic rename
	_ = os.Rename(tempPath, cachePath)
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
