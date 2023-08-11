package bundle

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadNotExists(t *testing.T) {
	b, err := Load(context.Background(), "/doesntexist")
	assert.True(t, os.IsNotExist(err))
	assert.Nil(t, b)
}

func TestLoadExists(t *testing.T) {
	b, err := Load(context.Background(), "./tests/basic")
	require.Nil(t, err)
	assert.Equal(t, "basic", b.Config.Bundle.Name)
}

func TestBundleCacheDir(t *testing.T) {
	projectDir := t.TempDir()
	f1, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	require.NoError(t, err)
	f1.Close()

	bundle, err := Load(context.Background(), projectDir)
	require.NoError(t, err)

	// Artificially set environment.
	// This is otherwise done by [mutators.SelectEnvironment].
	bundle.Config.Bundle.Environment = "default"

	// unset env variable in case it's set
	t.Setenv("DATABRICKS_BUNDLE_TMP", "")

	cacheDir, err := bundle.CacheDir()

	// format is <CWD>/.databricks/bundle/<environment>
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(projectDir, ".databricks", "bundle", "default"), cacheDir)
}

func TestBundleCacheDirOverride(t *testing.T) {
	projectDir := t.TempDir()
	bundleTmpDir := t.TempDir()
	f1, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	require.NoError(t, err)
	f1.Close()

	bundle, err := Load(context.Background(), projectDir)
	require.NoError(t, err)

	// Artificially set environment.
	// This is otherwise done by [mutators.SelectEnvironment].
	bundle.Config.Bundle.Environment = "default"

	// now we expect to use 'bundleTmpDir' instead of CWD/.databricks/bundle
	t.Setenv("DATABRICKS_BUNDLE_TMP", bundleTmpDir)

	cacheDir, err := bundle.CacheDir()

	// format is <DATABRICKS_BUNDLE_TMP>/<environment>
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(bundleTmpDir, "default"), cacheDir)
}

func TestBundleMustLoadSuccess(t *testing.T) {
	t.Setenv(envBundleRoot, "./tests/basic")
	b, err := MustLoad(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "tests/basic", filepath.ToSlash(b.Config.Path))
}

func TestBundleMustLoadFailureWithEnv(t *testing.T) {
	t.Setenv(envBundleRoot, "./tests/doesntexist")
	_, err := MustLoad(context.Background())
	require.Error(t, err, "not a directory")
}

func TestBundleMustLoadFailureIfNotFound(t *testing.T) {
	chdir(t, t.TempDir())
	_, err := MustLoad(context.Background())
	require.Error(t, err, "unable to find bundle root")
}

func TestBundleTryLoadSuccess(t *testing.T) {
	t.Setenv(envBundleRoot, "./tests/basic")
	b, err := TryLoad(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "tests/basic", filepath.ToSlash(b.Config.Path))
}

func TestBundleTryLoadFailureWithEnv(t *testing.T) {
	t.Setenv(envBundleRoot, "./tests/doesntexist")
	_, err := TryLoad(context.Background())
	require.Error(t, err, "not a directory")
}

func TestBundleTryLoadOkIfNotFound(t *testing.T) {
	chdir(t, t.TempDir())
	b, err := TryLoad(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, b)
}
