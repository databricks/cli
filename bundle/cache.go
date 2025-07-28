package bundle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Cache provides an abstract interface for caching content to local disk.
// Implementations should handle storing and retrieving cached components
// using fingerprints for cache invalidation.
type Cache interface {
	// Read retrieves cached content for the given fingerprint.
	// Returns the cached data and true if found, or nil and false if not found or expired.
	Read(ctx context.Context, fingerprint string) ([]byte, bool)

	// Store saves content to the cache with the given fingerprint.
	// Returns an error if the cache operation fails.
	Store(ctx context.Context, fingerprint string, content []byte) error

	// Clear removes all cached content from the cache directory.
	Clear(ctx context.Context) error

	// ClearFingerprint removes cached content for a specific fingerprint.
	ClearFingerprint(ctx context.Context, fingerprint string) error
}

// FileCache implements the Cache interface using the local filesystem.
type FileCache struct {
	cachePath string
}

// NewFileCache creates a new filesystem-based cache at the specified path.
func NewFileCache(cachePath string) *FileCache {
	return &FileCache{
		cachePath: cachePath,
	}
}

// Read retrieves cached content for the given fingerprint.
func (fc *FileCache) Read(ctx context.Context, fingerprint string) ([]byte, bool) {
	filePath := fc.getFilePath(fingerprint)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	return data, true
}

// Store saves content to the cache with the given fingerprint.
func (fc *FileCache) Store(ctx context.Context, fingerprint string, content []byte) error {
	filePath := fc.getFilePath(fingerprint)
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Clear removes all cached content from the cache directory.
func (fc *FileCache) Clear(ctx context.Context) error {
	if _, err := os.Stat(fc.cachePath); os.IsNotExist(err) {
		return nil
	}

	return os.RemoveAll(fc.cachePath)
}

// ClearFingerprint removes cached content for a specific fingerprint.
func (fc *FileCache) ClearFingerprint(ctx context.Context, fingerprint string) error {
	filePath := fc.getFilePath(fingerprint)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	return nil
}

// getFilePath returns the full file path for a given fingerprint.
func (fc *FileCache) getFilePath(fingerprint string) string {
	return filepath.Join(fc.cachePath, fingerprint+".cache")
}

// GenerateFingerprint creates a SHA256 fingerprint from the provided data.
// This is a utility function for creating consistent fingerprints.
func GenerateFingerprint(data ...any) (string, error) {
	hasher := sha256.New()

	for _, item := range data {
		var bytes []byte
		var err error

		switch v := item.(type) {
		case string:
			bytes = []byte(v)
		case []byte:
			bytes = v
		case io.Reader:
			bytes, err = io.ReadAll(v)
			if err != nil {
				return "", fmt.Errorf("failed to read data for fingerprint: %w", err)
			}
		default:
			bytes, err = json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("failed to marshal data for fingerprint: %w", err)
			}
		}

		hasher.Write(bytes)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
