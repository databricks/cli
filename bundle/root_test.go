package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestRootFromEnv(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	t.Setenv(env.RootVariable, dir)

	// The bundle root must contain a configuration file.
	f, err := os.Create(filepath.Join(dir, config.FileNames[0]))
	require.NoError(t, err)
	f.Close()

	// It should pull the root from the environment variable.
	file, diags := findConfigFile(ctx, true)
	require.NoError(t, diags.Error())
	require.Equal(t, f.Name(), file)
}

func TestRootFromEnvDoesntExist(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	t.Setenv(env.RootVariable, filepath.Join(dir, "doesntexist"))

	// It should pull the root from the environment variable.
	_, diags := findConfigFile(ctx, true)
	require.True(t, diags.HasError())
	require.Contains(t, diags[0].Summary, "Invalid bundle root")
}

func TestRootFromEnvCustomFile(t *testing.T) {
	t.Chdir(t.TempDir())
	dir, err := os.Getwd()
	require.NoError(t, err)
	testutil.Touch(t, dir, "databricks.yml")
	testutil.Touch(t, dir, "bundle.yaml")
	for _, name := range []string{"custom.yml", "custom.yaml", "custom.json"} {
		t.Run(name, func(t *testing.T) {
			testutil.Touch(t, dir, name)
			t.Setenv(env.RootVariable, name)
			file, err := ConfigFileFromEnv(t.Context())
			require.NoError(t, err)
			require.Equal(t, filepath.Join(dir, name), file)
		})
	}
}

func TestRootFromEnvUnsupportedFile(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "invalid"))
	require.NoError(t, err)
	f.Close()
	t.Setenv(env.RootVariable, f.Name())

	// It should pull the root from the environment variable.
	_, diags := findConfigFile(ctx, true)
	require.True(t, diags.HasError())
	require.Contains(t, diags[0].Summary, "Invalid bundle root")
}

func TestRootIfEnvIsEmpty(t *testing.T) {
	ctx := t.Context()
	dir := ""
	t.Setenv(env.RootVariable, dir)

	// It should pull the root from the environment variable.
	_, diags := findConfigFile(ctx, true)
	require.True(t, diags.HasError())
	require.Contains(t, diags[0].Summary, "Invalid bundle root")
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
	file, diags := findConfigFile(ctx, true)
	require.NoError(t, diags.Error())
	file, err = filepath.EvalSymlinks(file)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, config.FileNames[0]), file)
}

func TestRootLookupError(t *testing.T) {
	ctx := t.Context()

	// Have to set then unset to allow the testing package to revert it to its original value.
	t.Setenv(env.RootVariable, "")
	os.Unsetenv(env.RootVariable)

	// It can't find a project root from a temporary directory.
	t.Chdir(t.TempDir())
	_, diags := findConfigFile(ctx, true)
	require.True(t, diags.HasError())
	require.Contains(t, diags[0].Summary, "Unable to locate the bundle root")
}
