package workspace

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePath_TraversalAttempt(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test directory traversal attempt
	args := &ReadFileArgs{
		FilePath: "../../../etc/passwd",
	}
	content, err := provider.ReadFile(ctx, args)
	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "outside base directory")
}

func TestValidatePath_AbsolutePath(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test absolute path attempt
	args := &ReadFileArgs{
		FilePath: "/etc/passwd",
	}
	content, err := provider.ReadFile(ctx, args)
	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "absolute paths not allowed")
}

func TestValidatePath_SymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create a symlink pointing outside workspace
	outsideDir, err := os.MkdirTemp("", "outside-*")
	require.NoError(t, err)
	defer os.RemoveAll(outsideDir)

	outsideFile := filepath.Join(outsideDir, "secret.txt")
	err = os.WriteFile(outsideFile, []byte("secret"), 0644)
	require.NoError(t, err)

	symlinkPath := filepath.Join(tmpDir, "escape")
	err = os.Symlink(outsideDir, symlinkPath)
	require.NoError(t, err)

	ctx := context.Background()

	// Test reading through symlink
	args := &ReadFileArgs{
		FilePath: "escape/secret.txt",
	}
	content, err := provider.ReadFile(ctx, args)
	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "outside base directory")
}

func TestValidatePath_ValidRelativePaths(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test file in subdirectory
	subdir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subdir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(subdir, "test.txt")
	testContent := "valid content"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test valid relative path
	args := &ReadFileArgs{
		FilePath: "subdir/test.txt",
	}
	content, err := provider.ReadFile(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestValidatePath_DotPath(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "test content"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test path with dot
	args := &ReadFileArgs{
		FilePath: "./test.txt",
	}
	content, err := provider.ReadFile(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestValidatePath_ParentDirectoryWithinWorkspace(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create nested structure
	subdir := filepath.Join(tmpDir, "a", "b", "c")
	err := os.MkdirAll(subdir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(tmpDir, "a", "test.txt")
	testContent := "test content"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test parent directory reference that stays within workspace
	args := &ReadFileArgs{
		FilePath: "a/b/../test.txt",
	}
	content, err := provider.ReadFile(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestValidatePath_WriteAttemptOutsideWorkspace(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test write attempt outside workspace
	args := &WriteFileArgs{
		FilePath: "../../../tmp/malicious.txt",
		Content:  "malicious content",
	}
	err := provider.WriteFile(ctx, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside base directory")
}

func TestValidatePath_EditAttemptOutsideWorkspace(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test edit attempt outside workspace
	args := &EditFileArgs{
		FilePath:  "../../../../etc/passwd",
		OldString: "root",
		NewString: "hacked",
	}
	err := provider.EditFile(ctx, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside base directory")
}

func TestValidatePath_NonExistentParent(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test reading file with non-existent parent
	// Path validation should pass (path is within workspace),
	// but ReadFile should fail because file doesn't exist
	args := &ReadFileArgs{
		FilePath: "nonexistent/subdir/file.txt",
	}
	content, err := provider.ReadFile(ctx, args)
	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestValidatePath_NullByteInjection(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test null byte injection attempt
	args := &ReadFileArgs{
		FilePath: "test.txt\x00../../../etc/passwd",
	}
	content, err := provider.ReadFile(ctx, args)
	// Should either error or clean the path (Go's filepath.Clean handles this)
	assert.Error(t, err)
	assert.Empty(t, content)
}
