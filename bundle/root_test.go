package bundle

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestRootFromEnv(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv(env.RootVariable, dir)

	// It should pull the root from the environment variable.
	root, err := mustGetRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, root, dir)
}

func TestRootFromEnvDoesntExist(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv(env.RootVariable, filepath.Join(dir, "doesntexist"))

	// It should pull the root from the environment variable.
	_, err := mustGetRoot(ctx)
	require.Errorf(t, err, "invalid bundle root")
}

func TestRootFromEnvIsFile(t *testing.T) {
	ctx := context.Background()
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
	ctx := context.Background()
	dir := ""
	t.Setenv(env.RootVariable, dir)

	// It should pull the root from the environment variable.
	_, err := mustGetRoot(ctx)
	require.Errorf(t, err, "invalid bundle root")
}

func TestRootLookup(t *testing.T) {
	ctx := context.Background()

	// Have to set then unset to allow the testing package to revert it to its original value.
	t.Setenv(env.RootVariable, "")
	os.Unsetenv(env.RootVariable)

	testutil.Chdir(t, t.TempDir())

	// Create databricks.yml file.
	f, err := os.Create(config.FileNames[0])
	require.NoError(t, err)
	defer f.Close()

	// Create directory tree.
	err = os.MkdirAll("./a/b/c", 0o755)
	require.NoError(t, err)

	// It should find the project root from $PWD.
	wd := testutil.Chdir(t, "./a/b/c")
	root, err := mustGetRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, wd, root)
}

func TestRootLookupError(t *testing.T) {
	ctx := context.Background()

	// Have to set then unset to allow the testing package to revert it to its original value.
	t.Setenv(env.RootVariable, "")
	os.Unsetenv(env.RootVariable)

	// It can't find a project root from a temporary directory.
	_ = testutil.Chdir(t, t.TempDir())
	_, err := mustGetRoot(ctx)
	require.ErrorContains(t, err, "unable to locate bundle root")
}
