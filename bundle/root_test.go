package bundle

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
	t.Setenv(envBundleRoot, dir)

	// It should pull the root from the environment variable.
	root, err := getRoot()
	require.NoError(t, err)
	require.Equal(t, root, dir)
}

func TestRootFromEnvDoesntExist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envBundleRoot, filepath.Join(dir, "doesntexist"))

	// It should pull the root from the environment variable.
	_, err := getRoot()
	require.Errorf(t, err, "invalid bundle root")
}

func TestRootFromEnvIsFile(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "invalid"))
	require.NoError(t, err)
	f.Close()
	t.Setenv(envBundleRoot, f.Name())

	// It should pull the root from the environment variable.
	_, err = getRoot()
	require.Errorf(t, err, "invalid bundle root")
}

func TestRootIfEnvIsEmpty(t *testing.T) {
	dir := ""
	t.Setenv(envBundleRoot, dir)

	// It should pull the root from the environment variable.
	_, err := getRoot()
	require.Errorf(t, err, "invalid bundle root")
}

func TestRootLookup(t *testing.T) {
	// Have to set then unset to allow the testing package to revert it to its original value.
	t.Setenv(envBundleRoot, "")
	os.Unsetenv(envBundleRoot)

	chdir(t, t.TempDir())

	// Create bundle.yml file.
	f, err := os.Create(ConfigFile)
	require.NoError(t, err)
	defer f.Close()

	// Create directory tree.
	err = os.MkdirAll("./a/b/c", 0755)
	require.NoError(t, err)

	// It should find the project root from $PWD.
	wd := chdir(t, "./a/b/c")
	root, err := getRoot()
	require.NoError(t, err)
	require.Equal(t, wd, root)
}

func TestRootLookupError(t *testing.T) {
	// Have to set then unset to allow the testing package to revert it to its original value.
	t.Setenv(envBundleRoot, "")
	os.Unsetenv(envBundleRoot)

	// It can't find a project root from a temporary directory.
	_ = chdir(t, t.TempDir())
	_, err := getRoot()
	require.ErrorContains(t, err, "unable to locate bundle root")
}
