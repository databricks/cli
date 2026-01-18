package configsync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveFiles_Success(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()

	yamlPath := filepath.Join(tmpDir, "subdir", "databricks.yml")
	modifiedContent := `resources:
  jobs:
    test_job:
      name: "Updated Job"
      timeout_seconds: 7200
`

	files := []FileChange{
		{
			Path:            yamlPath,
			OriginalContent: "original content",
			ModifiedContent: modifiedContent,
		},
	}

	err := SaveFiles(ctx, &bundle.Bundle{}, files)
	require.NoError(t, err)

	_, err = os.Stat(yamlPath)
	require.NoError(t, err)

	content, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	assert.Equal(t, modifiedContent, string(content))

	_, err = os.Stat(filepath.Dir(yamlPath))
	require.NoError(t, err)
}

func TestSaveFiles_MultipleFiles(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()

	file1Path := filepath.Join(tmpDir, "file1.yml")
	file2Path := filepath.Join(tmpDir, "subdir", "file2.yml")
	content1 := "content for file 1"
	content2 := "content for file 2"

	files := []FileChange{
		{
			Path:            file1Path,
			OriginalContent: "original 1",
			ModifiedContent: content1,
		},
		{
			Path:            file2Path,
			OriginalContent: "original 2",
			ModifiedContent: content2,
		},
	}

	err := SaveFiles(ctx, &bundle.Bundle{}, files)
	require.NoError(t, err)

	content, err := os.ReadFile(file1Path)
	require.NoError(t, err)
	assert.Equal(t, content1, string(content))

	content, err = os.ReadFile(file2Path)
	require.NoError(t, err)
	assert.Equal(t, content2, string(content))
}

func TestSaveFiles_EmptyList(t *testing.T) {
	ctx := context.Background()

	err := SaveFiles(ctx, &bundle.Bundle{}, []FileChange{})
	require.NoError(t, err)
}
