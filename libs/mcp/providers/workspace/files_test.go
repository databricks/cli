package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/mcp/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
)

func setupTestProvider(t *testing.T) (*Provider, string) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "workspace-test-*")
	require.NoError(t, err)

	// Create session and set work directory
	sess := session.NewSession()
	err = sess.SetWorkDir(tmpDir)
	require.NoError(t, err)

	// Create provider
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := NewProvider(sess, logger)
	require.NoError(t, err)

	return provider, tmpDir
}

func TestProvider_ReadFile(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Write a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!\nLine 2\nLine 3"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test reading full file
	args := &ReadFileArgs{FilePath: "test.txt"}
	content, err := provider.ReadFile(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestProvider_WriteFile(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	testContent := "Test content"

	// Test writing new file
	args := &WriteFileArgs{
		FilePath: "new-file.txt",
		Content:  testContent,
	}
	err := provider.WriteFile(ctx, args)
	assert.NoError(t, err)

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "new-file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestProvider_WriteFileWithSubdirectory(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	testContent := "Test content"

	// Test writing file in subdirectory
	args := &WriteFileArgs{
		FilePath: "subdir/nested/file.txt",
		Content:  testContent,
	}
	err := provider.WriteFile(ctx, args)
	assert.NoError(t, err)

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "subdir/nested/file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestProvider_EditFile(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Write a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "Hello, World!\nThis is a test."
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test editing file
	args := &EditFileArgs{
		FilePath:  "test.txt",
		OldString: "World",
		NewString: "Go",
	}
	err = provider.EditFile(ctx, args)
	assert.NoError(t, err)

	// Verify file was edited
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Hello, Go!")
}

func TestProvider_EditFileNonUnique(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Write a test file with duplicate content
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "test test test"
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test editing file with non-unique string
	args := &EditFileArgs{
		FilePath:  "test.txt",
		OldString: "test",
		NewString: "replaced",
	}
	err = provider.EditFile(ctx, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "appears 3 times")
}

func TestProvider_EditFileNotFound(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Write a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "Hello, World!"
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test editing with string that doesn't exist
	args := &EditFileArgs{
		FilePath:  "test.txt",
		OldString: "NotFound",
		NewString: "replaced",
	}
	err = provider.EditFile(ctx, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProvider_ReadFileWithRange(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Write a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test reading with offset
	args := &ReadFileArgs{
		FilePath: "test.txt",
		Offset:   2,
		Limit:    2,
	}
	content, err := provider.ReadFile(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, "Line 2\nLine 3", content)
}

func TestApplyLineRange(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		offset   int
		limit    int
		expected string
	}{
		{
			name:     "no offset or limit",
			content:  "line1\nline2\nline3",
			offset:   0,
			limit:    0,
			expected: "line1\nline2\nline3",
		},
		{
			name:     "offset 2",
			content:  "line1\nline2\nline3\nline4",
			offset:   2,
			limit:    0,
			expected: "line2\nline3\nline4",
		},
		{
			name:     "limit 2",
			content:  "line1\nline2\nline3\nline4",
			offset:   0,
			limit:    2,
			expected: "line1\nline2",
		},
		{
			name:     "offset 2 limit 2",
			content:  "line1\nline2\nline3\nline4\nline5",
			offset:   2,
			limit:    2,
			expected: "line2\nline3",
		},
		{
			name:     "offset beyond end",
			content:  "line1\nline2",
			offset:   10,
			limit:    0,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyLineRange([]byte(tt.content), tt.offset, tt.limit)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}
