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
