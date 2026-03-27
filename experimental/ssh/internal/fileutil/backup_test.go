package fileutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupFile_EmptyData(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "file.json")

	err := fileutil.BackupFile(t.Context(), path, []byte{})
	require.NoError(t, err)

	_, err = os.Stat(path + fileutil.SuffixOriginalBak)
	assert.True(t, os.IsNotExist(err))
}

func TestBackupFile_FirstBackup(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "file.json")
	data := []byte(`{"key": "value"}`)

	err := fileutil.BackupFile(t.Context(), path, data)
	require.NoError(t, err)

	content, err := os.ReadFile(path + fileutil.SuffixOriginalBak)
	require.NoError(t, err)
	assert.Equal(t, data, content)

	_, err = os.Stat(path + fileutil.SuffixLatestBak)
	assert.True(t, os.IsNotExist(err))
}

func TestBackupFile_SubsequentBackup(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "file.json")
	original := []byte(`{"key": "value"}`)
	updated := []byte(`{"key": "updated"}`)

	err := fileutil.BackupFile(t.Context(), path, original)
	require.NoError(t, err)

	err = fileutil.BackupFile(t.Context(), path, updated)
	require.NoError(t, err)

	// .original.bak must remain unchanged
	content, err := os.ReadFile(path + fileutil.SuffixOriginalBak)
	require.NoError(t, err)
	assert.Equal(t, original, content)

	// .latest.bak should have the updated content
	content, err = os.ReadFile(path + fileutil.SuffixLatestBak)
	require.NoError(t, err)
	assert.Equal(t, updated, content)
}

func TestBackupFile_WriteError(t *testing.T) {
	err := fileutil.BackupFile(t.Context(), "/nonexistent/dir/file.json", []byte("data"))
	assert.Error(t, err)
}
