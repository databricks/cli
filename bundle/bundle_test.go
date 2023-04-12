package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadNotExists(t *testing.T) {
	b, err := Load("/doesntexist")
	assert.True(t, os.IsNotExist(err))
	assert.Nil(t, b)
}

func TestLoadExists(t *testing.T) {
	b, err := Load("./tests/basic")
	require.Nil(t, err)
	assert.Equal(t, "basic", b.Config.Bundle.Name)
}

func TestBundleCacheDir(t *testing.T) {
	projectDir := t.TempDir()
	f1, err := os.Create(filepath.Join(projectDir, "bundle.yml"))
	require.NoError(t, err)
	f1.Close()

	bundle, err := Load(projectDir)
	require.NoError(t, err)

	// Artificially set environment.
	// This is otherwise done by [mutators.SelectEnvironment].
	bundle.Config.Bundle.Environment = "default"

	cacheDir, err := bundle.CacheDir()
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(cacheDir, projectDir))
}

func TestBundleMustLoadSuccess(t *testing.T) {
	t.Setenv(envBundleRoot, "./tests/basic")
	b, err := MustLoad()
	require.NoError(t, err)
	assert.Equal(t, "tests/basic", filepath.ToSlash(b.Config.Path))
}

func TestBundleMustLoadFailureWithEnv(t *testing.T) {
	t.Setenv(envBundleRoot, "./tests/doesntexist")
	_, err := MustLoad()
	require.Error(t, err, "not a directory")
}

func TestBundleMustLoadFailureIfNotFound(t *testing.T) {
	chdir(t, t.TempDir())
	_, err := MustLoad()
	require.Error(t, err, "unable to find bundle root")
}

func TestBundleTryLoadSuccess(t *testing.T) {
	t.Setenv(envBundleRoot, "./tests/basic")
	b, err := TryLoad()
	require.NoError(t, err)
	assert.Equal(t, "tests/basic", filepath.ToSlash(b.Config.Path))
}

func TestBundleTryLoadFailureWithEnv(t *testing.T) {
	t.Setenv(envBundleRoot, "./tests/doesntexist")
	_, err := TryLoad()
	require.Error(t, err, "not a directory")
}

func TestBundleTryLoadOkIfNotFound(t *testing.T) {
	chdir(t, t.TempDir())
	b, err := TryLoad()
	assert.NoError(t, err)
	assert.Nil(t, b)
}
