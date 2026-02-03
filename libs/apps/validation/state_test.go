package validation

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		path     string
		excluded bool
	}{
		// Git
		{".git", true},
		{".git/config", true},
		// Node.js
		{"node_modules", true},
		{"node_modules/express/index.js", true},
		{".next", true},
		{"dist", true},
		{"build", true},
		// Python
		{"__pycache__", true},
		{"__pycache__/module.pyc", true},
		{".venv", true},
		{"venv", true},
		{"file.pyc", true},
		{"file.pyo", true},
		{".pytest_cache", true},
		{".mypy_cache", true},
		{".ruff_cache", true},
		{"package.egg-info", true},
		{"package.egg-info/PKG-INFO", true},
		// Editor/OS
		{".DS_Store", true},
		{"file.swp", true},
		{".idea", true},
		{".vscode", true},
		// Temp files
		{"app.log", true},
		{"file.tmp", true},
		{"file.temp", true},
		// Databricks
		{StateFileName, true},
		{".databricks", true},
		{".databricks/bundle", true},
		// Should NOT be excluded
		{"src/main.js", false},
		{"package.json", false},
		{"README.md", false},
		{"app.py", false},
		{"databricks.yml", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.excluded, shouldExclude(tc.path))
		})
	}
}

func TestComputeChecksum(t *testing.T) {
	dir := t.TempDir()

	// Create some files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("content2"), 0o644))

	checksum1, err := ComputeChecksum(dir)
	require.NoError(t, err)
	assert.Len(t, checksum1, 64) // SHA256 hex

	// Same content = same checksum
	checksum2, err := ComputeChecksum(dir)
	require.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)

	// Modify file = different checksum
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("modified"), 0o644))
	checksum3, err := ComputeChecksum(dir)
	require.NoError(t, err)
	assert.NotEqual(t, checksum1, checksum3)
}

func TestComputeChecksumExcludesPatterns(t *testing.T) {
	dir := t.TempDir()

	// Create a source file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.js"), []byte("code"), 0o644))

	checksum1, err := ComputeChecksum(dir)
	require.NoError(t, err)

	// Add excluded files - checksum should not change
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "pkg.js"), []byte("dep"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, StateFileName), []byte("state"), 0o644))

	checksum2, err := ComputeChecksum(dir)
	require.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)
}

func TestLoadStateMissing(t *testing.T) {
	dir := t.TempDir()

	state, err := LoadState(dir)
	require.NoError(t, err)
	assert.Nil(t, state)
}

func TestSaveAndLoadState(t *testing.T) {
	dir := t.TempDir()

	now := time.Now().UTC().Truncate(time.Second)
	original := &State{
		State:       StateValidated,
		ValidatedAt: now,
		Checksum:    "abc123",
	}

	require.NoError(t, SaveState(dir, original))

	loaded, err := LoadState(dir)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, StateValidated, loaded.State)
	assert.Equal(t, "abc123", loaded.Checksum)
	assert.Equal(t, now.Unix(), loaded.ValidatedAt.Unix())
}

func TestSaveStateAtomic(t *testing.T) {
	dir := t.TempDir()

	state := &State{
		State:       StateDeployed,
		ValidatedAt: time.Now().UTC(),
		Checksum:    "xyz789",
	}

	require.NoError(t, SaveState(dir, state))

	// Verify no temp file left behind
	_, err := os.Stat(filepath.Join(dir, StateFileName+".tmp"))
	assert.True(t, os.IsNotExist(err))

	// Verify state file exists
	_, err = os.Stat(filepath.Join(dir, StateFileName))
	assert.NoError(t, err)
}

func TestLoadStateCorrupted(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, StateFileName)
	require.NoError(t, os.WriteFile(statePath, []byte("not valid json"), 0o644))

	state, err := LoadState(dir)
	require.Error(t, err)
	assert.Nil(t, state)
	assert.Contains(t, err.Error(), "corrupted")
}

func TestComputeChecksumEmptyDir(t *testing.T) {
	dir := t.TempDir()

	checksum, err := ComputeChecksum(dir)
	require.NoError(t, err)
	assert.Len(t, checksum, 64) // Should still produce valid checksum
}

func TestSaveStateCleansTempFile(t *testing.T) {
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, StateFileName+".tmp")

	// Create a leftover temp file
	require.NoError(t, os.WriteFile(tmpPath, []byte("leftover"), 0o644))

	state := &State{
		State:       StateValidated,
		ValidatedAt: time.Now().UTC(),
		Checksum:    "test123",
	}

	require.NoError(t, SaveState(dir, state))

	// Temp file should be gone
	_, err := os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err))
}
