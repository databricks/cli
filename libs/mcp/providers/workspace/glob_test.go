package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Glob(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test files
	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "file3.md"), []byte("test"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test glob pattern
	args := &GlobArgs{
		Pattern: "*.txt",
	}
	result, err := provider.Glob(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Contains(t, result.Files, "file1.txt")
	assert.Contains(t, result.Files, "file2.txt")
}

func TestProvider_GlobSubdirectories(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test files in subdirectories
	subdir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subdir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte("test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subdir, "file2.go"), []byte("test"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test glob pattern with subdirectories
	args := &GlobArgs{
		Pattern: "*/*.go",
	}
	result, err := provider.Glob(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Contains(t, result.Files, filepath.Join("subdir", "file2.go"))
}

func TestProvider_GlobAllFiles(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test files
	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "file2.md"), []byte("test"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test glob all files
	args := &GlobArgs{
		Pattern: "*",
	}
	result, err := provider.Glob(ctx, args)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, result.Total, 2)
}

func TestProvider_GlobNoMatches(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test file
	err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test glob with no matches
	args := &GlobArgs{
		Pattern: "*.nonexistent",
	}
	result, err := provider.Glob(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Files)
}

func TestProvider_GlobSpecificFile(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test file
	err := os.WriteFile(filepath.Join(tmpDir, "specific.txt"), []byte("test"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test glob for specific file
	args := &GlobArgs{
		Pattern: "specific.txt",
	}
	result, err := provider.Glob(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Contains(t, result.Files, "specific.txt")
}

func TestProvider_GlobSorted(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test files
	err := os.WriteFile(filepath.Join(tmpDir, "c.txt"), []byte("test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("test"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test glob returns sorted results
	args := &GlobArgs{
		Pattern: "*.txt",
	}
	result, err := provider.Glob(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, []string{"a.txt", "b.txt", "c.txt"}, result.Files)
}
