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

// MkDir wraps os.MkdirAll, which is a no-op on a directory that already exists
// and never changes its mode. These two cases pin that down: whether the
// requested mode matches the existing directory or not, the directory keeps the
// mode it already had and the write still succeeds.
func TestWriteMkDirExistingDirSameMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not honor unix directory modes")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o700))
	path := filepath.Join(dir, "out")

	require.NoError(t, Write(path, []byte("x"), 0o600, MkDir(0o700)))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "x", string(got))

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}

// Write replaces a symlink at path with a regular file rather than following it
// to the target: os.Rename swaps the path entry itself. This is intentional — an
// atomic replace should not write through a link into some other file.
func TestWriteReplacesSymlinkWithoutFollowing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks is restricted on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	require.NoError(t, os.WriteFile(target, []byte("original target"), 0o600))

	link := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink(target, link))

	require.NoError(t, Write(link, []byte("new"), 0o600))

	// The link path now holds a regular file with the new content...
	info, err := os.Lstat(link)
	require.NoError(t, err)
	assert.Zero(t, info.Mode()&os.ModeSymlink, "link should no longer be a symlink")
	got, err := os.ReadFile(link)
	require.NoError(t, err)
	assert.Equal(t, "new", string(got))

	// ...and the original target is untouched, proving the link was not followed.
	targetContent, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "original target", string(targetContent))
}

func TestWriteMkDirExistingDirDifferentMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not honor unix directory modes")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o700))
	path := filepath.Join(dir, "out")

	// Ask for 0o755 even though the directory already exists at 0o700.
	require.NoError(t, Write(path, []byte("x"), 0o600, MkDir(0o755)))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "x", string(got))

	// The existing directory keeps 0o700; MkDir does not widen it to 0o755.
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}
