package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Grep(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test files
	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("Hello World\nGoodbye World"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("Testing 123\nAnother line"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test basic grep
	args := &GrepArgs{
		Pattern: "World",
	}
	result, err := provider.Grep(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, len(result.Matches))
}

func TestProvider_GrepCaseInsensitive(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test file
	err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("Hello WORLD\nGoodbye world"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test case insensitive grep
	args := &GrepArgs{
		Pattern:    "world",
		IgnoreCase: true,
	}
	result, err := provider.Grep(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Total)
}

func TestProvider_GrepMaxResults(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test file with many matches
	content := "test\ntest\ntest\ntest\ntest\ntest"
	err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte(content), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test max results limit
	args := &GrepArgs{
		Pattern:    "test",
		MaxResults: 3,
	}
	result, err := provider.Grep(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, 3, len(result.Matches))
}

func TestProvider_GrepRegex(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test file
	err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test123\ntest456\nabc789"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test regex pattern
	args := &GrepArgs{
		Pattern: "test[0-9]+",
	}
	result, err := provider.Grep(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Total)
}

func TestProvider_GrepPathLimit(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	// Create test files in subdirectory
	subdir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subdir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("match"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subdir, "file2.txt"), []byte("match"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test grep limited to subdirectory
	args := &GrepArgs{
		Pattern: "match",
		Path:    "subdir",
	}
	result, err := provider.Grep(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Contains(t, result.Matches[0].File, "subdir")
}

func TestProvider_GrepInvalidPattern(t *testing.T) {
	provider, tmpDir := setupTestProvider(t)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Test invalid regex pattern
	args := &GrepArgs{
		Pattern: "[invalid(",
	}
	result, err := provider.Grep(ctx, args)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"file.txt", true},
		{"file.go", true},
		{"file.ts", true},
		{"file.md", true},
		{"file.json", true},
		{"file.exe", false},
		{"file.bin", false},
		{"file", false},
		{"README.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isTextFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
