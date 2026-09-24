package storage

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func put(t testing.TB, store Store, key string, entry Entry) {
	t.Helper()
	require.NoError(t, store.WithLock(t.Context(), func(locked LockedStore) error {
		return locked.Put(key, entry)
	}))
}

func deleteEntry(t testing.TB, store Store, key string) {
	t.Helper()
	require.NoError(t, store.WithLock(t.Context(), func(locked LockedStore) error {
		return locked.Delete(key)
	}))
}

func setup(t *testing.T) string {
	tempHomeDir := t.TempDir()
	return filepath.Join(tempHomeDir, "token-cache.json")
}

func TestStoreAndLookup(t *testing.T) {
	c, err := NewFileStore(t.Context(), WithFileLocation(setup(t)))
	require.NoError(t, err)
	put(t, c, "x", Entry{Token: &oauth2.Token{
		AccessToken: "abc",
	}})

	put(t, c, "y", Entry{Token: &oauth2.Token{
		AccessToken: "bcd",
	}})

	got, err := c.Lookup("x")
	require.NoError(t, err)
	assert.Equal(t, "abc", got.Token.AccessToken)

	_, err = c.Lookup("z")
	assert.Equal(t, ErrNotFound, err)
}

func TestNoStoreFileReturnsErrNotConfigured(t *testing.T) {
	l, err := NewFileStore(t.Context(), WithFileLocation(setup(t)))
	require.NoError(t, err)
	_, err = l.Lookup("x")
	assert.Equal(t, ErrNotFound, err)
}

func TestLoadCorruptFile(t *testing.T) {
	f := setup(t)
	err := os.MkdirAll(filepath.Dir(f), ownerExecReadWrite)
	require.NoError(t, err)
	err = os.WriteFile(f, []byte("abc"), ownerExecReadWrite)
	require.NoError(t, err)

	_, err = NewFileStore(t.Context(), WithFileLocation(f))
	assert.EqualError(t, err, "load: parse: invalid character 'a' looking for beginning of value")
}

func TestLoadWrongVersion(t *testing.T) {
	f := setup(t)
	err := os.MkdirAll(filepath.Dir(f), ownerExecReadWrite)
	require.NoError(t, err)
	err = os.WriteFile(f, []byte(`{"version": 823, "things": []}`), ownerExecReadWrite)
	require.NoError(t, err)

	_, err = NewFileStore(t.Context(), WithFileLocation(f))
	assert.EqualError(t, err, "load: needs version 1, got version 823")
}

func TestFileStoreTransactionsReloadBeforeEveryMutation(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	first, err := NewFileStore(ctx)
	require.NoError(t, err)
	second, err := NewFileStore(ctx)
	require.NoError(t, err)

	require.NoError(t, first.WithLock(ctx, func(locked LockedStore) error {
		return locked.Put("first", Entry{Token: &oauth2.Token{AccessToken: "one"}})
	}))
	// A second independent store must reload the first write before adding
	// another key.
	require.NoError(t, second.WithLock(ctx, func(locked LockedStore) error {
		return locked.Put("second", Entry{Token: &oauth2.Token{AccessToken: "two"}})
	}))

	firstEntry, err := first.Lookup("first")
	require.NoError(t, err)
	secondEntry, err := second.Lookup("second")
	require.NoError(t, err)
	assert.Equal(t, "one", firstEntry.Token.AccessToken)
	assert.Equal(t, "two", secondEntry.Token.AccessToken)
}

func TestFileStoreProcessWriter(t *testing.T) {
	if os.Getenv("DATABRICKS_TOKEN_STORE_HELPER") != "1" {
		return
	}
	ctx := env.WithUserHomeDir(t.Context(), os.Getenv(env.HomeEnvVar()))
	store, err := NewFileStore(ctx)
	require.NoError(t, err)
	key := os.Getenv("DATABRICKS_TOKEN_STORE_KEY")
	require.NoError(t, store.WithLock(ctx, func(locked LockedStore) error {
		return locked.Put(key, Entry{Token: &oauth2.Token{AccessToken: key}})
	}))
}

func TestFileStoreIndependentProcessesPreserveDistinctKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv(env.HomeEnvVar(), home)
	ctx := env.WithUserHomeDir(t.Context(), home)
	store, err := NewFileStore(ctx)
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)

	commands := make([]*exec.Cmd, 0, 2)
	for _, key := range []string{"process-a", "process-b"} {
		cmd := exec.CommandContext(t.Context(), executable, "-test.run=^TestFileStoreProcessWriter$")
		cmd.Env = append(os.Environ(),
			"DATABRICKS_TOKEN_STORE_HELPER=1",
			"DATABRICKS_TOKEN_STORE_KEY="+key,
		)
		require.NoError(t, cmd.Start())
		commands = append(commands, cmd)
	}
	for _, cmd := range commands {
		require.NoError(t, cmd.Wait())
	}

	for _, key := range []string{"process-a", "process-b"} {
		entry, err := store.Lookup(key)
		require.NoError(t, err)
		assert.Equal(t, key, entry.Token.AccessToken)
	}
}

func TestNewFileStoreTightensExistingPermissions(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}
	path := setup(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), ownerExecReadWrite))
	require.NoError(t, os.WriteFile(path, []byte(`{"version":1,"tokens":{}}`), 0o644))

	_, err := NewFileStore(t.Context(), WithFileLocation(path))
	require.NoError(t, err)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(ownerReadWrite), info.Mode().Perm())
}
