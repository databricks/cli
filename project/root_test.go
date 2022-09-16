package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Changes into specified directory for the duration of the test.
// Returns the current working directory.
func chdir(t *testing.T, dir string) string {
	wd, err := os.Getwd()
	require.NoError(t, err)

	abs, err := filepath.Abs(dir)
	require.NoError(t, err)

	err = os.Chdir(abs)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	})

	return wd
}

func TestRootFromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(bricksRoot, dir)

	// It should pull the root from the environment variable.
	root, err := getRoot()
	require.NoError(t, err)
	require.Equal(t, root, dir)
}

func TestRootFromEnvDoesntExist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(bricksRoot, filepath.Join(dir, "doesntexist"))

	// It should pull the root from the environment variable.
	_, err := getRoot()
	require.Errorf(t, err, "invalid project root")
}

func TestRootFromEnvIsFile(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "invalid"))
	require.NoError(t, err)
	f.Close()
	t.Setenv(bricksRoot, f.Name())

	// It should pull the root from the environment variable.
	_, err = getRoot()
	require.Errorf(t, err, "invalid project root")
}

func TestRootIfEnvIsEmpty(t *testing.T) {
	dir := ""
	t.Setenv(bricksRoot, dir)

	// It should pull the root from the environment variable.
	_, err := getRoot()
	require.Errorf(t, err, "invalid project root")
}

func TestRootLookup(t *testing.T) {
	// Have to set then unset to allow the testing package to revert it to its original value.
	t.Setenv(bricksRoot, "")
	os.Unsetenv(bricksRoot)

	// It should find the project root from $PWD.
	wd := chdir(t, "./testdata/a/b/c")
	root, err := getRoot()
	require.NoError(t, err)
	require.Equal(t, root, filepath.Join(wd, "testdata"))
}

func TestRootLookupError(t *testing.T) {
	// Have to set then unset to allow the testing package to revert it to its original value.
	t.Setenv(bricksRoot, "")
	os.Unsetenv(bricksRoot)

	// It can't find a project root from a temporary directory.
	_ = chdir(t, t.TempDir())
	_, err := getRoot()
	require.ErrorContains(t, err, "unable to locate project root")
}
