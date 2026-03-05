package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
	"github.com/stretchr/testify/require"
)

func TestRootFromEnv(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	t.Setenv(env.RootVariable, dir)

	// It should pull the root from the environment variable.
	root, err := mustGetRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, root, dir)
}

func TestRootFromEnvDoesntExist(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	t.Setenv(env.RootVariable, filepath.Join(dir, "doesntexist"))

	// It should pull the root from the environment variable.
	_, err := mustGetRoot(ctx)
	require.Errorf(t, err, "invalid bundle root")
}

func TestRootFromEnvIsFile(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "invalid"))
	require.NoError(t, err)
	f.Close()
	t.Setenv(env.RootVariable, f.Name())

	// It should pull the root from the environment variable.
	_, err = mustGetRoot(ctx)
	require.Errorf(t, err, "invalid bundle root")
}

func TestRootIfEnvIsEmpty(t *testing.T) {
	ctx := t.Context()
	dir := ""
	t.Setenv(env.RootVariable, dir)

	// It should pull the root from the environment variable.
	_, err := mustGetRoot(ctx)
	require.Errorf(t, err, "invalid bundle root")
}

func TestRootLookup(t *testing.T) {
	ctx := t.Context()

	// Have to set then unset to allow the testing package to revert it to its original value.
	t.Setenv(env.RootVariable, "")
	os.Unsetenv(env.RootVariable)

	t.Chdir(t.TempDir())

	// Resolve to canonical path for comparison below. This is needed because
	// os.Getwd may return a path with symlinks (macOS) or 8.3 short names
	// (Windows) after a relative chdir.
	root, err := os.Getwd()
	require.NoError(t, err)
	root, err = filepath.EvalSymlinks(root)
	require.NoError(t, err)

	// Create databricks.yml file.
	f, err := os.Create(config.FileNames[0])
	require.NoError(t, err)
	defer f.Close()

	// Create directory tree.
	err = os.MkdirAll("./a/b/c", 0o755)
	require.NoError(t, err)

	// It should find the project root from $PWD.
	t.Chdir("./a/b/c")
	foundRoot, err := mustGetRoot(ctx)
	require.NoError(t, err)
	foundRoot, err = filepath.EvalSymlinks(foundRoot)
	require.NoError(t, err)
	require.Equal(t, root, foundRoot)
}

func TestRootLookupError(t *testing.T) {
	ctx := t.Context()

	// Have to set then unset to allow the testing package to revert it to its original value.
	t.Setenv(env.RootVariable, "")
	os.Unsetenv(env.RootVariable)

	// It can't find a project root from a temporary directory.
	t.Chdir(t.TempDir())
	_, err := mustGetRoot(ctx)
	require.ErrorContains(t, err, "unable to locate bundle root")
}
