package ucm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/ucm/config"
	"github.com/stretchr/testify/require"
)

func TestRootFromEnv(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	t.Setenv(RootEnv, dir)

	root, err := mustGetRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, root, dir)
}

func TestRootFromCtxEnv(t *testing.T) {
	dir := t.TempDir()
	// Clear any process-level RootEnv so only the ctx override is visible.
	t.Setenv(RootEnv, "")
	os.Unsetenv(RootEnv)
	ctx := env.Set(t.Context(), RootEnv, dir)

	root, err := mustGetRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, root, dir)
}

func TestRootFromEnvDoesntExist(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	t.Setenv(RootEnv, filepath.Join(dir, "doesntexist"))

	_, err := mustGetRoot(ctx)
	require.Errorf(t, err, "invalid ucm root")
}

func TestRootFromEnvIsFile(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "invalid"))
	require.NoError(t, err)
	f.Close()
	t.Setenv(RootEnv, f.Name())

	_, err = mustGetRoot(ctx)
	require.Errorf(t, err, "invalid ucm root")
}

func TestRootIfEnvIsEmpty(t *testing.T) {
	ctx := t.Context()
	dir := ""
	t.Setenv(RootEnv, dir)

	_, err := mustGetRoot(ctx)
	require.Errorf(t, err, "invalid ucm root")
}

func TestRootLookup(t *testing.T) {
	ctx := t.Context()

	t.Setenv(RootEnv, "")
	os.Unsetenv(RootEnv)

	t.Chdir(t.TempDir())

	root, err := os.Getwd()
	require.NoError(t, err)
	root, err = filepath.EvalSymlinks(root)
	require.NoError(t, err)

	f, err := os.Create(config.FileNames[0])
	require.NoError(t, err)
	defer f.Close()

	err = os.MkdirAll("./a/b/c", 0o755)
	require.NoError(t, err)

	t.Chdir("./a/b/c")
	foundRoot, err := mustGetRoot(ctx)
	require.NoError(t, err)
	foundRoot, err = filepath.EvalSymlinks(foundRoot)
	require.NoError(t, err)
	require.Equal(t, root, foundRoot)
}

func TestRootLookupError(t *testing.T) {
	ctx := t.Context()

	t.Setenv(RootEnv, "")
	os.Unsetenv(RootEnv)

	t.Chdir(t.TempDir())
	_, err := mustGetRoot(ctx)
	require.ErrorContains(t, err, "unable to locate ucm root")
}
