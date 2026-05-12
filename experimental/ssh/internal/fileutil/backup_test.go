package fileutil_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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
	assert.ErrorIs(t, err, fs.ErrNotExist)
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
	assert.ErrorIs(t, err, fs.ErrNotExist)
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

func TestBackupFile_StatError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod is not supported on windows")
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "file.json")

	// Create the .original.bak file so os.Stat would find it — but make the
	// parent directory unreadable so Stat returns a permission error instead.
	require.NoError(t, os.WriteFile(path+fileutil.SuffixOriginalBak, []byte("existing"), 0o600))
	require.NoError(t, os.Chmod(tmpDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(tmpDir, 0o700) })

	err := fileutil.BackupFile(t.Context(), path, []byte("data"))
	assert.Error(t, err)
}
