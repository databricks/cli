package bundle

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
	"github.com/stretchr/testify/assert"
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

	chdir(t, t.TempDir())

	// Create databricks.yml file.
	f, err := os.Create(config.FileNames[0])
	require.NoError(t, err)
	defer f.Close()

	// Create directory tree.
	err = os.MkdirAll("./a/b/c", 0755)
	require.NoError(t, err)

	// It should find the project root from $PWD.
	wd := chdir(t, "./a/b/c")
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
	_ = chdir(t, t.TempDir())
	_, err := mustGetRoot(ctx)
	require.ErrorContains(t, err, "unable to locate bundle root")
}

func TestLoadYamlWhenIncludesEnvPresent(t *testing.T) {
	ctx := context.Background()
	chdir(t, filepath.Join(".", "tests", "basic"))
	t.Setenv(env.IncludesVariable, "test")

	bundle, err := MustLoad(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "basic", bundle.Config.Bundle.Name)

	cwd, err := os.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, cwd, bundle.Config.Path)
}

func TestLoadDefautlBundleWhenNoYamlAndRootAndIncludesEnvPresent(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	chdir(t, dir)
	t.Setenv(env.RootVariable, dir)
	t.Setenv(env.IncludesVariable, "test")

	bundle, err := MustLoad(ctx)
	assert.NoError(t, err)
	assert.Equal(t, dir, bundle.Config.Path)
}

func TestErrorIfNoYamlNoRootEnvAndIncludesEnvPresent(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	chdir(t, dir)
	t.Setenv(env.IncludesVariable, "test")

	_, err := MustLoad(ctx)
	assert.Error(t, err)
}

func TestErrorIfNoYamlNoIncludesEnvAndRootEnvPresent(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	chdir(t, dir)
	t.Setenv(env.RootVariable, dir)

	_, err := MustLoad(ctx)
	assert.Error(t, err)
}
