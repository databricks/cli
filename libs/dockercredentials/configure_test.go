package dockercredentials_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigure(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "docker", "config.json")
	ctx := env.Set(t.Context(), "DOCKER_CONFIG", filepath.Dir(configPath))
	result, err := dockercredentials.Configure(ctx, filepath.Join(dir, "bin", "databricks"), "registry.test")
	require.NoError(t, err)
	assert.Equal(t, configPath, result.ConfigPath)
	assert.FileExists(t, result.Shim.Path)
	status, err := dockercredentials.Inspect(ctx, "registry.test")
	require.NoError(t, err)
	assert.True(t, status.Configured)
	assert.Equal(t, "registry.test", status.Host)
}

func TestConfigureInstallsShimBeforeUpdatingConfig(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(blocked, nil, 0o600))
	configPath := filepath.Join(dir, "docker", "config.json")
	ctx := env.Set(t.Context(), "DOCKER_CONFIG", filepath.Dir(configPath))
	_, err := dockercredentials.Configure(ctx, filepath.Join(blocked, "databricks"), "registry.test")
	assert.ErrorContains(t, err, "install Docker credential helper")
	assert.NoFileExists(t, configPath)
}

func TestConfigPath(t *testing.T) {
	dir := t.TempDir()
	for _, dockerDir := range []string{"", filepath.Join(dir, "custom")} {
		ctx := env.Set(t.Context(), "HOME", dir)
		ctx = env.Set(ctx, "USERPROFILE", dir)
		ctx = env.Set(ctx, "DOCKER_CONFIG", dockerDir)
		got, err := dockercredentials.ConfigPath(ctx)
		require.NoError(t, err)
		wantDir := filepath.Join(dir, ".docker")
		if dockerDir != "" {
			wantDir = dockerDir
		}
		assert.Equal(t, filepath.Join(wantDir, "config.json"), got)
	}
}
