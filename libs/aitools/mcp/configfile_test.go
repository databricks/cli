package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveConfigFileResolvesSymlinkBeforeDotDot checks that ".." after a symlink is taken from the
// link's target, as the OS reads the path, not from the link's own directory.
func TestResolveConfigFileResolvesSymlinkBeforeDotDot(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	target := filepath.Join(base, "a", "b")
	require.NoError(t, os.MkdirAll(target, 0o755))
	want := filepath.Join(base, "a", ".databrickscfg")
	require.NoError(t, os.WriteFile(want, []byte("[logfood]\n"), 0o600))
	link := filepath.Join(base, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	// filepath.Join would clean the ".." away, so the path is joined by hand.
	sep := string(filepath.Separator)
	got, err := resolveConfigFile(t.Context(), link+sep+".."+sep+".databrickscfg")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestResolveConfigFileWithoutHomeAndAbsoluteName checks that an absolute name resolves with no home
// directory, and that a "~" name reports the missing home directory.
func TestResolveConfigFileWithoutHomeAndAbsoluteName(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), "")
	name := filepath.Join(t.TempDir(), ".databrickscfg")

	got, err := resolveConfigFile(ctx, name)
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(got))

	_, err = resolveConfigFile(ctx, "~/.databrickscfg")
	require.ErrorIs(t, err, env.ErrNoHomeEnv)
}

// TestResolveConfigFileKeepsPathThroughMissingDir checks that a path through a missing directory is kept
// as written. Cleaning "missing/.." would name the parent's file, which the OS never opens through a
// missing directory.
func TestResolveConfigFileKeepsPathThroughMissingDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cfg"), []byte("[logfood]\n"), 0o600))
	sep := string(filepath.Separator)
	name := dir + sep + "missing" + sep + ".." + sep + "cfg"

	got, err := resolveConfigFile(t.Context(), name)
	require.NoError(t, err)
	assert.Equal(t, name, got)
}

// TestSamePathComparesExistingFilesByIdentity checks that two names of one existing file compare equal,
// as a hard link does, while names of files that do not exist are compared as text.
func TestSamePathComparesExistingFilesByIdentity(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	require.NoError(t, os.WriteFile(a, []byte("[logfood]\n"), 0o600))
	if err := os.Link(a, b); err != nil {
		t.Skipf("cannot create hard link: %v", err)
	}

	assert.True(t, samePath(a, b))
	assert.False(t, samePath(filepath.Join(dir, "x"), filepath.Join(dir, "X")))
}
