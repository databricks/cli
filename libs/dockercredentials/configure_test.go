package dockercredentials_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	inspectRegistry = dockercredentials.Registry{WorkspaceID: workspaceID, Host: "registry.test"}
	inspectProfiler = profile.InMemoryProfiler{Profiles: profile.Profiles{workspaceProfile("workspace")}}
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
	status, err := dockercredentials.Inspect(ctx, inspectProfiler, inspectRegistry, "workspace")
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

			status, err := dockercredentials.Inspect(ctx, inspectProfiler, inspectRegistry, "workspace")
			require.NoError(t, err)
			assert.Equal(t, "registry.test", status.Host)
			assert.Equal(t, tc.want, status.Configured)
		})
	}
}

func TestInspectRequiresDockerConfigAndProfile(t *testing.T) {
	recorded := workspaceProfile("prod")
	recorded.DockerCredentialHelper = true
	otherWorkspace := workspaceProfile("other")
	otherWorkspace.WorkspaceID = "987654321"
	tests := []struct {
		name             string
		profiles         profile.Profiles
		selected         string
		skipDockerConfig bool
		want             bool
	}{
		{name: "only profile", profiles: profile.Profiles{workspaceProfile("prod")}, selected: "prod", want: true},
		{name: "Docker config does not use the helper", profiles: profile.Profiles{recorded}, selected: "prod", skipDockerConfig: true},
		{name: "recorded profile", profiles: profile.Profiles{recorded, workspaceProfile("dev")}, selected: "prod", want: true},
		{name: "another profile is recorded", profiles: profile.Profiles{recorded, workspaceProfile("dev")}, selected: "dev"},
		{name: "no profile is recorded", profiles: profile.Profiles{workspaceProfile("prod"), workspaceProfile("dev")}, selected: "prod"},
		{name: "profile is for another workspace", profiles: profile.Profiles{workspaceProfile("prod"), otherWorkspace}, selected: "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			binDir := filepath.Join(dir, "bin")
			configPath := filepath.Join(dir, "docker", "config.json")
			ctx := env.Set(t.Context(), "DOCKER_CONFIG", filepath.Dir(configPath))
			t.Setenv("PATH", binDir)
			t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
			_, err := dockercredentials.Configure(ctx, filepath.Join(binDir, "databricks"), "registry.test")
			require.NoError(t, err)
			if tt.skipDockerConfig {
				require.NoError(t, os.Remove(configPath))
			}

			status, err := dockercredentials.Inspect(ctx, profile.InMemoryProfiler{Profiles: tt.profiles}, inspectRegistry, tt.selected)
			require.NoError(t, err)
			assert.Equal(t, tt.want, status.Configured)
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
