package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIgnoreFile(t *testing.T) {
	var ign bool
	var err error

	f := newIgnoreFile(vfs.MustNew("testdata"), ".gitignore")
	ign, err = f.MatchesPath("root.foo")
	require.NoError(t, err)
	assert.True(t, ign)
	ign, err = f.MatchesPath("i'm included")
	require.NoError(t, err)
	assert.False(t, ign)
}

func TestIgnoreFileDoesntExist(t *testing.T) {
	var ign bool
	var err error

	// Files that don't exist are treated as an empty gitignore file.
	f := newIgnoreFile(vfs.MustNew("testdata"), "thispathdoesntexist")
	ign, err = f.MatchesPath("i'm included")
	require.NoError(t, err)
	assert.False(t, ign)
}

func TestIgnoreFileTaint(t *testing.T) {
	var ign bool
	var err error

	tempDir := t.TempDir()
	gitIgnorePath := filepath.Join(tempDir, ".gitignore")

	// Files that don't exist are treated as an empty gitignore file.
	f := newIgnoreFile(vfs.MustNew(tempDir), ".gitignore")
	ign, err = f.MatchesPath("hello")
	require.NoError(t, err)
	assert.False(t, ign)

	// Now create the .gitignore file.
	err = os.WriteFile(gitIgnorePath, []byte("hello"), 0o644)
	require.NoError(t, err)

	// Verify that the match still doesn't happen (no spontaneous reload).
	ign, err = f.MatchesPath("hello")
	require.NoError(t, err)
	assert.False(t, ign)

	// Now taint the file to force a reload and verify that the match does happen.
	f.Taint()
	ign, err = f.MatchesPath("hello")
	require.NoError(t, err)
	assert.True(t, ign)
}

func TestStringIgnoreRules(t *testing.T) {
	var ign bool
	var err error

	f := newStringIgnoreRules([]string{"hello"})
	ign, err = f.MatchesPath("hello")
	require.NoError(t, err)
	assert.True(t, ign)
	ign, err = f.MatchesPath("world")
	require.NoError(t, err)
	assert.False(t, ign)
}
