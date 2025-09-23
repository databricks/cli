package cache

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/log"
)

// CleanupConfig holds configuration for cache cleanup.
type CleanupConfig struct {
	MaxAge       time.Duration // Maximum age of cache files before cleanup
	ScanInterval time.Duration // How often to scan for old files
	DryRun       bool          // If true, only logs what would be deleted
}

// DefaultCleanupConfig returns sensible defaults for cache cleanup.
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		MaxAge:       7 * 24 * time.Hour, // 7 days
		ScanInterval: 24 * time.Hour,     // Daily cleanup
		DryRun:       false,
	}
}

// CleanupManager manages background cleanup of cache files.
type CleanupManager struct {
	config    CleanupConfig
	stopCh    chan struct{}
	stoppedCh chan struct{}
	mu        sync.Mutex
	running   bool
	stopped   bool
}

// NewCleanupManager creates a new cleanup manager with the given configuration.
func NewCleanupManager(config CleanupConfig) *CleanupManager {
	return &CleanupManager{
		config:    config,
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
}

// Start begins the background cleanup process.
// This is non-blocking and will not prevent the main process from exiting.
func (cm *CleanupManager) Start(ctx context.Context, cacheDir string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.running || cm.stopped {
		return // Already running or stopped
	}

	cm.running = true

	go func() {
		defer func() {
			cm.mu.Lock()
			cm.running = false
			cm.mu.Unlock()
			close(cm.stoppedCh)
		}()

		log.Debugf(ctx, "[Cache Cleanup] Starting cleanup manager for directory: %s", cacheDir)

		// Perform initial cleanup
		cm.cleanup(ctx, cacheDir)

		// Set up periodic cleanup
		ticker := time.NewTicker(cm.config.ScanInterval)
		defer ticker.Stop()

		for {
			select {
			case <-cm.stopCh:
				log.Debugf(ctx, "[Cache Cleanup] Cleanup manager stopped")
				return
			case <-ticker.C:
				cm.cleanup(ctx, cacheDir)
			}
		}
	}()
}

// Stop gracefully stops the cleanup manager.
// This is non-blocking and returns immediately.
func (cm *CleanupManager) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.running || cm.stopped {
		return
	}

	cm.stopped = true
	close(cm.stopCh)
}

// Wait waits for the cleanup manager to stop completely.
// This should only be used in tests or shutdown scenarios where you need to wait.
func (cm *CleanupManager) Wait() {
	<-cm.stoppedCh
}

// cleanup performs the actual cleanup of old cache files.
func (cm *CleanupManager) cleanup(ctx context.Context, cacheDir string) {
	log.Debugf(ctx, "[Cache Cleanup] Starting cleanup scan of directory: %s", cacheDir)

	// Check if cache directory exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		log.Debugf(ctx, "[Cache Cleanup] Cache directory does not exist: %s", cacheDir)
		return
	}

	var deletedCount, scannedCount int
	var totalSize, deletedSize int64
	cutoff := time.Now().Add(-cm.config.MaxAge)

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Debugf(ctx, "[Cache Cleanup] Error accessing path %s: %v", path, err)
			return nil // Continue with other files
		}

		// Skip directories and non-cache files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		scannedCount++
		totalSize += info.Size()

		shouldDelete, fileAge := cm.shouldDeleteFile(ctx, path, cutoff)
		if shouldDelete {
			deletedSize += info.Size()
			deletedCount++

			if cm.config.DryRun {
				log.Debugf(ctx, "[Cache Cleanup] Would delete old cache file: %s (age: %v)", path, fileAge)
			} else {
				if err := os.Remove(path); err != nil {
					log.Debugf(ctx, "[Cache Cleanup] Failed to delete cache file %s: %v", path, err)
				} else {
					log.Debugf(ctx, "[Cache Cleanup] Deleted old cache file: %s (age: %v)", path, fileAge)
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Debugf(ctx, "[Cache Cleanup] Error during cleanup scan: %v", err)
	}

	action := "deleted"
	if cm.config.DryRun {
		action = "would delete"
	}

	log.Debugf(ctx, "[Cache Cleanup] Cleanup complete: scanned %d files (%.2f MB), %s %d files (%.2f MB)",
		scannedCount, float64(totalSize)/(1024*1024),
		action, deletedCount, float64(deletedSize)/(1024*1024))
}

// shouldDeleteFile determines if a cache file should be deleted based on its age.
func (cm *CleanupManager) shouldDeleteFile(ctx context.Context, path string, cutoff time.Time) (bool, time.Duration) {
	// Try to read the cache entry to get the timestamp
	data, err := os.ReadFile(path)
	if err != nil {
		// If we can't read the file, use file modification time as fallback
		if info, statErr := os.Stat(path); statErr == nil {
			age := time.Since(info.ModTime())
			return info.ModTime().Before(cutoff), age
		}
		return true, time.Duration(0) // Delete unreadable files
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// If we can't parse the cache entry, use file modification time as fallback
		if info, statErr := os.Stat(path); statErr == nil {
			age := time.Since(info.ModTime())
			return info.ModTime().Before(cutoff), age
		}
		return true, time.Duration(0) // Delete unparseable files
	}

	age := time.Since(entry.Timestamp)
	return entry.Timestamp.Before(cutoff), age
}
