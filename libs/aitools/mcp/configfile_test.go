package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setFoldCase sets the case-folding seam for the test and restores it afterwards. The seam lets both
// comparisons run on any OS, where the platform default would pick only one.
func setFoldCase(t *testing.T, fold bool) {
	t.Helper()
	prev := foldCase
	t.Cleanup(func() { foldCase = prev })
	foldCase = fold
}

// TestSamePathFollowsFoldCase checks that paths compare case-insensitively only when the seam says so.
func TestSamePathFollowsFoldCase(t *testing.T) {
	const lower = "/home/u/.databrickscfg"
	const upper = "/HOME/U/.DATABRICKSCFG"

	setFoldCase(t, true)
	assert.True(t, samePath(lower, upper))
	assert.True(t, samePath(lower, lower))

	setFoldCase(t, false)
	assert.False(t, samePath(lower, upper))
	assert.True(t, samePath(lower, lower))
}

// TestFindEntryMatchesCaseFoldedPath checks that an entry whose file name differs only in case matches
// the config file when paths fold case, as they do on Windows and macOS.
func TestFindEntryMatchesCaseFoldedPath(t *testing.T) {
	setFoldCase(t, true)
	// Resolve the directory first, so the file the entry names and the file in the context match on
	// systems where the temp directory is a symlink.
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	configFile := filepath.Join(dir, ".databrickscfg")
	state := &State{Servers: []Entry{{
		Name:       "github-logfood",
		Service:    "system.ai.github",
		Profile:    "logfood",
		ConfigFile: filepath.Join(dir, ".DATABRICKSCFG"),
	}}}

	entry, err := findEntry(t.Context(), state, "system.ai.github", "logfood", configFile)
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, "github-logfood", entry.Name)
}

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
