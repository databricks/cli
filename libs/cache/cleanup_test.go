package cache

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCleanupConfig(t *testing.T) {
	config := DefaultCleanupConfig()
	assert.Equal(t, 7*24*time.Hour, config.MaxAge)
	assert.False(t, config.DryRun)
}

func TestCleanupManager_Start_Stop(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := CleanupConfig{
		MaxAge: time.Hour,
		DryRun: false,
	}

	manager := NewCleanupManager(config)

	// Start the manager
	manager.Start(ctx, tempDir)
	assert.True(t, manager.running)

	// Stop the manager
	manager.Stop()

	// Wait for it to stop with timeout
	done := make(chan struct{})
	go func() {
		manager.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.False(t, manager.running)
	case <-time.After(5 * time.Second):
		t.Fatal("Cleanup manager did not stop within timeout")
	}
}

func TestCleanupManager_CleanupOldFiles(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := CleanupConfig{
		MaxAge: time.Hour,
		DryRun: false,
	}

	manager := NewCleanupManager(config)

	// Create test files with different ages
	now := time.Now()

	// Create an old file (should be deleted)
	oldFile := filepath.Join(tempDir, "old_file.json")
	oldEntry := cacheEntry{
		Data:      json.RawMessage(`"old_data"`),
		Timestamp: now.Add(-2 * time.Hour), // 2 hours old
	}
	oldData, err := json.Marshal(oldEntry)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(oldFile, oldData, 0o644))

	// Create a recent file (should not be deleted)
	recentFile := filepath.Join(tempDir, "recent_file.json")
	recentEntry := cacheEntry{
		Data:      json.RawMessage(`"recent_data"`),
		Timestamp: now.Add(-30 * time.Minute), // 30 minutes old
	}
	recentData, err := json.Marshal(recentEntry)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(recentFile, recentData, 0o644))

	// Create a non-cache file (should be ignored)
	nonCacheFile := filepath.Join(tempDir, "not_cache.txt")
	require.NoError(t, os.WriteFile(nonCacheFile, []byte("not cache"), 0o644))

	// Run cleanup manually
	manager.cleanup(ctx, tempDir)

	// Check results
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err), "Old file should be deleted")

	_, err = os.Stat(recentFile)
	assert.False(t, os.IsNotExist(err), "Recent file should not be deleted")

	_, err = os.Stat(nonCacheFile)
	assert.False(t, os.IsNotExist(err), "Non-cache file should not be deleted")
}

func TestCleanupManager_DryRun(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := CleanupConfig{
		MaxAge: time.Hour,
		DryRun: true, // Dry run mode
	}

	manager := NewCleanupManager(config)

	// Create an old file
	now := time.Now()
	oldFile := filepath.Join(tempDir, "old_file.json")
	oldEntry := cacheEntry{
		Data:      json.RawMessage(`"old_data"`),
		Timestamp: now.Add(-2 * time.Hour),
	}
	oldData, err := json.Marshal(oldEntry)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(oldFile, oldData, 0o644))

	// Run cleanup in dry run mode
	manager.cleanup(ctx, tempDir)

	// File should still exist in dry run mode
	_, err = os.Stat(oldFile)
	assert.False(t, os.IsNotExist(err), "File should not be deleted in dry run mode")
}

func TestCleanupManager_CorruptedFiles(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := CleanupConfig{
		MaxAge: time.Hour,
		DryRun: false,
	}

	manager := NewCleanupManager(config)

	// Create a corrupted cache file (invalid JSON)
	corruptedFile := filepath.Join(tempDir, "corrupted.json")
	require.NoError(t, os.WriteFile(corruptedFile, []byte("invalid json"), 0o644))

	// Set old modification time
	oldTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(corruptedFile, oldTime, oldTime))

	// Create a file with invalid cache entry structure
	invalidStructureFile := filepath.Join(tempDir, "invalid_structure.json")
	require.NoError(t, os.WriteFile(invalidStructureFile, []byte(`{"invalid": "structure"}`), 0o644))

	// Set old modification time
	require.NoError(t, os.Chtimes(invalidStructureFile, oldTime, oldTime))

	// Run cleanup - corrupted files should be deleted
	manager.cleanup(ctx, tempDir)

	// Both corrupted files should be deleted
	_, err := os.Stat(corruptedFile)
	assert.True(t, os.IsNotExist(err), "Corrupted file should be deleted")

	_, err = os.Stat(invalidStructureFile)
	assert.True(t, os.IsNotExist(err), "Invalid structure file should be deleted")
}

func TestCleanupManager_NonexistentDirectory(t *testing.T) {
	ctx := context.Background()
	nonexistentDir := "/nonexistent/directory"

	config := CleanupConfig{
		MaxAge: time.Hour,
		DryRun: false,
	}

	manager := NewCleanupManager(config)

	// This should not panic or error when directory doesn't exist
	manager.cleanup(ctx, nonexistentDir)
}

func TestShouldDeleteFile(t *testing.T) {
	tempDir := t.TempDir()

	manager := NewCleanupManager(DefaultCleanupConfig())
	now := time.Now()

	// Test with valid cache entry with expiry - expired file
	expiredFile := filepath.Join(tempDir, "expired.json")
	expiredEntry := cacheEntry{
		Data:   json.RawMessage(`"data"`),
		Expiry: now.Add(-time.Hour), // Expired 1 hour ago
	}
	expiredData, err := json.Marshal(expiredEntry)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(expiredFile, expiredData, 0o644))

	shouldDelete, age := manager.shouldDeleteFile(expiredFile, now)
	assert.True(t, shouldDelete, "Expired file should be marked for deletion")
	assert.GreaterOrEqual(t, age, time.Hour, "Age should reflect time since expiry")

	// Test with valid cache entry with expiry - not expired file
	validFile := filepath.Join(tempDir, "valid.json")
	validEntry := cacheEntry{
		Data:   json.RawMessage(`"data"`),
		Expiry: now.Add(time.Hour), // Expires in 1 hour
	}
	validData, err := json.Marshal(validEntry)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(validFile, validData, 0o644))

	shouldDelete, age = manager.shouldDeleteFile(validFile, now)
	assert.False(t, shouldDelete, "Valid file should not be marked for deletion")
	assert.Equal(t, time.Duration(0), age, "Age should be 0 for unexpired files")

	// Test with legacy timestamp field (backward compatibility)
	legacyFile := filepath.Join(tempDir, "legacy.json")
	legacyEntry := cacheEntry{
		Data:      json.RawMessage(`"data"`),
		Timestamp: now.Add(-2 * time.Hour), // Created 2 hours ago
	}
	legacyData, err := json.Marshal(legacyEntry)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(legacyFile, legacyData, 0o644))

	shouldDelete, age = manager.shouldDeleteFile(legacyFile, now)
	// Should not be deleted since MaxAge is 7 days by default, but 2 hours < 7 days
	assert.False(t, shouldDelete, "Legacy file should not be deleted if within MaxAge")
	assert.Greater(t, age, 2*time.Hour, "Age should be based on timestamp")

	// Test with invalid JSON - should use file modification time
	invalidFile := filepath.Join(tempDir, "invalid.json")
	require.NoError(t, os.WriteFile(invalidFile, []byte("invalid"), 0o644))
	// Set modification time to be old
	oldTime := now.Add(-8 * 24 * time.Hour) // 8 days ago (beyond default MaxAge)
	require.NoError(t, os.Chtimes(invalidFile, oldTime, oldTime))

	shouldDelete, age = manager.shouldDeleteFile(invalidFile, now)
	assert.True(t, shouldDelete, "Invalid file should be marked for deletion based on mod time")
	assert.Greater(t, age, 7*24*time.Hour, "Age should be based on modification time")
}

func TestCleanupIntegrationWithFileCache(t *testing.T) {
	tempDir := t.TempDir()

	// Create file cache which should start cleanup automatically
	cache, err := newFileCacheWithBaseDir[string](tempDir, 60) // 1 hour for tests
	require.NoError(t, err)
	require.NotNil(t, cache.cleanupMgr)

	// Stop cleanup to prevent interference with test
	cache.StopCleanup()
	cache.cleanupMgr.Wait()

	// Verify cache directory was created
	_, err = os.Stat(tempDir)
	assert.False(t, os.IsNotExist(err), "Cache directory should exist")
}

func TestCleanupManager_MultipleStartStop(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	manager := NewCleanupManager(DefaultCleanupConfig())

	// Start multiple times - should only start once
	manager.Start(ctx, tempDir)
	manager.Start(ctx, tempDir) // Second start should be ignored
	assert.True(t, manager.running)

	// Stop multiple times - should be safe
	manager.Stop()
	manager.Stop() // Second stop should be safe

	manager.Wait()
	assert.False(t, manager.running)
}

func TestCleanupFileWalk(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := CleanupConfig{
		MaxAge: time.Hour,
		DryRun: false,
	}

	manager := NewCleanupManager(config)

	// Create nested directory structure
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	now := time.Now()

	// Create old files in both root and subdirectory
	oldFile1 := filepath.Join(tempDir, "old1.json")
	oldFile2 := filepath.Join(subDir, "old2.json")

	for _, file := range []string{oldFile1, oldFile2} {
		oldEntry := cacheEntry{
			Data:      json.RawMessage(`"old_data"`),
			Timestamp: now.Add(-2 * time.Hour),
		}
		data, err := json.Marshal(oldEntry)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(file, data, 0o644))
	}

	// Run cleanup
	manager.cleanup(ctx, tempDir)

	// Both files should be deleted
	for _, file := range []string{oldFile1, oldFile2} {
		_, err := os.Stat(file)
		assert.True(t, os.IsNotExist(err), "File %s should be deleted", file)
	}

	// Subdirectory should still exist
	_, err := os.Stat(subDir)
	assert.False(t, os.IsNotExist(err), "Subdirectory should still exist")
}
