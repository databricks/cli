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
	t.Setenv("PATH", filepath.Join(dir, "bin"))
	t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
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

func TestInspectRequiresHelperOnPath(t *testing.T) {
	for _, tc := range []struct {
		name      string
		installed bool
		onPath    bool
		want      bool
	}{
		{name: "missing"},
		{name: "outside PATH", installed: true},
		{name: "on PATH", installed: true, onPath: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			binDir := filepath.Join(dir, "bin")
			configPath := filepath.Join(dir, "docker", "config.json")
			ctx := env.Set(t.Context(), "DOCKER_CONFIG", filepath.Dir(configPath))
			require.NoError(t, dockercredentials.SetCredentialHelper(configPath, "registry.test"))
			if tc.onPath {
				t.Setenv("PATH", binDir)
			} else {
				t.Setenv("PATH", t.TempDir())
			}
			t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
			if tc.installed {
				_, err := dockercredentials.InstallShim(filepath.Join(binDir, "databricks"))
				require.NoError(t, err)
			}

			status, err := dockercredentials.Inspect(ctx, "registry.test")
			require.NoError(t, err)
			assert.Equal(t, "registry.test", status.Host)
			assert.Equal(t, tc.want, status.Configured)
		})
	}
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
