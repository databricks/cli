package atomicfile

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")

	require.NoError(t, Write(path, []byte("hello"), 0o600))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))
}

func TestWriteUsesGivenMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not honor unix file modes")
	}
	path := filepath.Join(t.TempDir(), "out")

	require.NoError(t, Write(path, []byte("x"), 0o644))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestWriteDoesNotPreserveReplacedMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not honor unix file modes")
	}
	path := filepath.Join(t.TempDir(), "out")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

	// Replacing a 0600 file with perm 0644 yields 0644, not the old mode.
	require.NoError(t, Write(path, []byte("new"), 0o644))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestWriteOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out")
	require.NoError(t, os.WriteFile(path, []byte("old contents"), 0o600))

	require.NoError(t, Write(path, []byte("new"), 0o600))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new", string(got))
}

func TestWriteLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out")

	require.NoError(t, Write(path, []byte("x"), 0o600))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "out", entries[0].Name())
}

func TestWriteMissingDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-dir", "out")

	err := Write(path, []byte("x"), 0o600)
	require.Error(t, err)

	// The failure must not leave the target behind.
	_, statErr := os.Stat(path)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestWriteMkDirCreatesParents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "out")

	require.NoError(t, Write(path, []byte("x"), 0o600, MkDir(0o700)))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "x", string(got))

	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Dir(path))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	}
}
