package lakebox

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stateCtx returns a context whose $HOME is a temp directory, so state file
// operations are isolated from the developer's real ~/.databricks/lakebox.json.
func stateCtx(t *testing.T) (context.Context, string) {
	t.Helper()
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	return ctx, filepath.Join(home, ".databricks", "lakebox.json")
}

func TestStateLoadMissingFileReturnsEmpty(t *testing.T) {
	ctx, _ := stateCtx(t)
	state, err := loadState(ctx)
	require.NoError(t, err)
	assert.Equal(t, &stateFile{Defaults: map[string]string{}}, state)
}

func TestStateGetDefaultMissingProfileReturnsEmpty(t *testing.T) {
	ctx, _ := stateCtx(t)
	assert.Equal(t, "", getDefault(ctx, "any-profile"))
}

func TestStateSetGetDefaultRoundTrip(t *testing.T) {
	ctx, _ := stateCtx(t)

	require.NoError(t, setDefault(ctx, "profile-a", "lakebox-a"))
	assert.Equal(t, "lakebox-a", getDefault(ctx, "profile-a"))
	assert.Equal(t, "", getDefault(ctx, "profile-b"))
}

func TestStateMultipleProfilesIndependent(t *testing.T) {
	ctx, _ := stateCtx(t)

	require.NoError(t, setDefault(ctx, "profile-a", "lakebox-a"))
	require.NoError(t, setDefault(ctx, "profile-b", "lakebox-b"))

	assert.Equal(t, "lakebox-a", getDefault(ctx, "profile-a"))
	assert.Equal(t, "lakebox-b", getDefault(ctx, "profile-b"))
}

func TestStateSetDefaultOverwrites(t *testing.T) {
	ctx, _ := stateCtx(t)

	require.NoError(t, setDefault(ctx, "profile-a", "lakebox-a"))
	require.NoError(t, setDefault(ctx, "profile-a", "lakebox-a-prime"))
	assert.Equal(t, "lakebox-a-prime", getDefault(ctx, "profile-a"))
}

func TestStateClearDefault(t *testing.T) {
	ctx, _ := stateCtx(t)

	require.NoError(t, setDefault(ctx, "profile-a", "lakebox-a"))
	require.NoError(t, setDefault(ctx, "profile-b", "lakebox-b"))

	require.NoError(t, clearDefault(ctx, "profile-a"))
	assert.Equal(t, "", getDefault(ctx, "profile-a"))
	assert.Equal(t, "lakebox-b", getDefault(ctx, "profile-b"))
}

func TestStateClearDefaultMissingProfileDoesNotCreateFile(t *testing.T) {
	ctx, path := stateCtx(t)

	require.NoError(t, clearDefault(ctx, "no-such-profile"))

	_, err := os.Stat(path)
	assert.ErrorIs(t, err, fs.ErrNotExist, "clearDefault must not create the state file when there's nothing to remove")
}

func TestStateSetDefaultSameValueDoesNotRewriteFile(t *testing.T) {
	ctx, path := stateCtx(t)

	require.NoError(t, setDefault(ctx, "profile-a", "lakebox-a"))
	before, err := os.Stat(path)
	require.NoError(t, err)

	// Re-set with the same value should be a no-op.
	require.NoError(t, setDefault(ctx, "profile-a", "lakebox-a"))
	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, before.ModTime(), after.ModTime(), "no-op setDefault must not rewrite the file")
}

func TestStateSetDefaultMissingNoFileBeforeWrite(t *testing.T) {
	ctx, path := stateCtx(t)

	// Loading state on a fresh tempdir must not create the file.
	assert.Equal(t, "", getDefault(ctx, "profile-a"))
	_, err := os.Stat(path)
	assert.ErrorIs(t, err, fs.ErrNotExist, "getDefault must not create the state file")
}

// Pre-existing files from earlier CLI versions carry a `last_profile` field
// the current schema doesn't know about. loadState must accept the file
// (silently dropping the unknown field) and saveState must rewrite without
// it, so the field naturally falls off on the next mutation.
func TestStateLoadIgnoresUnknownFields(t *testing.T) {
	ctx, path := stateCtx(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(`{
        "defaults": {"profile-a": "lakebox-a"},
        "last_profile": "profile-a"
    }`), 0o600))

	assert.Equal(t, "lakebox-a", getDefault(ctx, "profile-a"))

	require.NoError(t, setDefault(ctx, "profile-a", "lakebox-a-prime"))
	rewritten, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(rewritten), "last_profile")
}

func TestStateLoadReturnsErrorOnCorruptJSON(t *testing.T) {
	ctx, path := stateCtx(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("{not valid json"), 0o600))

	_, err := loadState(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

// Files written by saveState must round-trip through loadState even if the
// caller starts from an empty Defaults map.
func TestStateSaveCreatesParentDirs(t *testing.T) {
	ctx, path := stateCtx(t)

	// Confirm parent dir does not exist yet.
	_, err := os.Stat(filepath.Dir(path))
	assert.ErrorIs(t, err, fs.ErrNotExist)

	require.NoError(t, setDefault(ctx, "profile-a", "lakebox-a"))

	// File and parent dir now exist with sensible perms.
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	dirInfo, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())
}

// Defaults of nil on disk (legal but not what saveState produces) must still
// load to a usable empty map so callers can setDefault without nil-deref.
func TestStateLoadNilDefaultsMap(t *testing.T) {
	ctx, path := stateCtx(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(`{}`), 0o600))

	require.NoError(t, setDefault(ctx, "profile-a", "lakebox-a"))
	assert.Equal(t, "lakebox-a", getDefault(ctx, "profile-a"))
}
