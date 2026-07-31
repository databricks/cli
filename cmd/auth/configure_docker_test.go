package auth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newConfigureDockerTestCommand(ctx context.Context, args ...string) *cobra.Command {
	cmd := New()
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetArgs(args)
	return cmd
}

func newConfigureDockerTestCommandWithDeps(ctx context.Context, deps configureDockerDeps, args ...string) *cobra.Command {
	cmd := &cobra.Command{Use: "auth"}
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.PersistentFlags().String("host", "", "Databricks Host")
	cmd.PersistentFlags().String("account-id", "", "Databricks Account ID")
	cmd.PersistentFlags().String("workspace-id", "", "Databricks Workspace ID")
	cmd.AddCommand(newConfigureDockerCommandWithDeps(deps))
	cmd.SetContext(ctx)
	cmd.SetArgs(args)
	return cmd
}

func writeConfigureDockerProfile(t *testing.T, ctx context.Context, configFile string, cfg *config.Config) {
	t.Helper()
	cfg.ConfigFile = configFile
	require.NoError(t, databrickscfg.SaveToProfile(ctx, cfg))
}

func readCredentialHelpers(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var cfg struct {
		CredHelpers map[string]string `json:"credHelpers"`
	}
	require.NoError(t, json.Unmarshal(raw, &cfg))
	return cfg.CredHelpers
}

func configureDockerRegistryHostStub(t *testing.T, wantWorkspaceID, wantRegion, wantWorkspaceHost, registryHost string) func(string, string, string) (string, error) {
	t.Helper()
	return func(workspaceID, region, workspaceHost string) (string, error) {
		require.Equal(t, wantWorkspaceID, workspaceID)
		require.Equal(t, wantRegion, region)
		require.Equal(t, wantWorkspaceHost, workspaceHost)
		return registryHost, nil
	}
}

func TestConfigureDockerCommandWritesDockerConfigAndShim(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	dockerDir := filepath.Join(dir, "docker")
	homeDir := filepath.Join(dir, "home")
	binDir := filepath.Join(homeDir, ".databricks", "bin")
	workspaceHost := "https://workspace.staging.cloud.databricks.test"

	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:     "DEFAULT",
		Host:        workspaceHost,
		WorkspaceID: "123456789",
		AuthType:    authTypeDatabricksCLI,
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", dockerDir)
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", binDir)

	registryHost := "123456789.container.us-west-2.staging.cloud.databricks.test"
	deps := defaultConfigureDockerDeps()
	deps.registryHost = configureDockerRegistryHostStub(t, "123456789", "us-west-2", workspaceHost, registryHost)

	cmd := newConfigureDockerTestCommandWithDeps(ctx, deps, "configure-docker", "DEFAULT", "--region", "us-west-2")
	require.NoError(t, cmd.Execute())

	helpers := readCredentialHelpers(t, filepath.Join(dockerDir, "config.json"))
	require.Equal(t, dockercredentials.HelperName, helpers[registryHost])

	_, err := os.Stat(filepath.Join(binDir, "docker-credential-databricks"))
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), registryHost)
	assert.Contains(t, stderr.String(), filepath.Join(dockerDir, "config.json"))
}

func TestConfigureDockerCommandRequiresRegion(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:     "DEFAULT",
		Host:        "https://workspace.cloud.databricks.test",
		WorkspaceID: "123456789",
		AuthType:    authTypeDatabricksCLI,
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	cmd := newConfigureDockerTestCommand(ctx, "configure-docker", "DEFAULT")
	err := cmd.Execute()
	require.ErrorContains(t, err, "--region is required because workspace region cannot be inferred from this profile")
}

func TestConfigureDockerCommandRejectsAccountOnlyProfile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	dockerDir := filepath.Join(dir, "docker")

	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:   "account",
		Host:      "https://accounts.cloud.databricks.test",
		AccountID: "acc",
		AuthType:  authTypeDatabricksCLI,
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", dockerDir)

	cmd := newConfigureDockerTestCommand(ctx, "configure-docker", "account", "--region", "us-west-2")
	err := cmd.Execute()
	require.ErrorContains(t, err, "databricks auth login --host <workspace-url>")
	require.NoFileExists(t, filepath.Join(dockerDir, "config.json"))
}

func TestConfigureDockerCommandPersistsResolvedWorkspaceID(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	dockerDir := filepath.Join(dir, "docker")
	homeDir := filepath.Join(dir, "home")
	workspaceHost := "https://workspace.gcp.databricks.test"

	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:  "workspace",
		Host:     workspaceHost,
		AuthType: authTypeDatabricksCLI,
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", dockerDir)
	t.Setenv("HOME", homeDir)

	deps := defaultConfigureDockerDeps()
	deps.newWorkspaceClient = func(cfg *databricks.Config) (*databricks.WorkspaceClient, error) {
		return &databricks.WorkspaceClient{Config: (*config.Config)(cfg)}, nil
	}
	deps.resolveWorkspaceID = func(context.Context, *databricks.WorkspaceClient) (string, error) {
		return "999999", nil
	}
	deps.registryHost = configureDockerRegistryHostStub(t, "999999", "us-west-2", workspaceHost, "999999.container.us-west-2.gcp.databricks.test")

	cmd := newConfigureDockerTestCommandWithDeps(ctx, deps, "configure-docker", "workspace", "--region", "us-west-2")
	require.NoError(t, cmd.Execute())

	raw, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "workspace_id = 999999")

	helpers := readCredentialHelpers(t, filepath.Join(dockerDir, "config.json"))
	require.Equal(t, dockercredentials.HelperName, helpers["999999.container.us-west-2.gcp.databricks.test"])
}

func TestConfigureDockerCommandRejectsUnsupportedWorkspaceHostBeforeProfileAndDockerConfigMutation(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	dockerDir := filepath.Join(dir, "docker")

	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:  "DEFAULT",
		Host:     "https://workspace.example.test",
		AuthType: authTypeDatabricksCLI,
	})
	before, err := os.ReadFile(configFile)
	require.NoError(t, err)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", dockerDir)
	t.Setenv("HOME", filepath.Join(dir, "home"))

	deps := defaultConfigureDockerDeps()
	deps.newWorkspaceClient = func(cfg *databricks.Config) (*databricks.WorkspaceClient, error) {
		return &databricks.WorkspaceClient{Config: (*config.Config)(cfg)}, nil
	}
	deps.resolveWorkspaceID = func(context.Context, *databricks.WorkspaceClient) (string, error) {
		return "123456789", nil
	}
	deps.installShim = func(context.Context, string, string) (dockercredentials.ShimInstallResult, error) {
		t.Fatal("installShim should not be called")
		return dockercredentials.ShimInstallResult{}, nil
	}
	deps.setCredentialHelper = func(string, string) error {
		t.Fatal("setCredentialHelper should not be called")
		return nil
	}

	cmd := newConfigureDockerTestCommandWithDeps(ctx, deps, "configure-docker", "DEFAULT", "--region", "us-west-2")
	err = cmd.Execute()
	require.ErrorContains(t, err, `"workspace.example.test" is not a supported Databricks workspace host`)
	after, err := os.ReadFile(configFile)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after))
	require.NoFileExists(t, filepath.Join(dockerDir, "config.json"))
}

func TestConfigureDockerCommandRejectsUnsupportedAuthProfiles(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	dockerDir := filepath.Join(dir, "docker")
	homeDir := filepath.Join(dir, "home")

	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:     "pat",
		Host:        "https://workspace.cloud.databricks.test",
		WorkspaceID: "123456789",
		AuthType:    "pat",
	})
	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:      "m2m",
		Host:         "https://m2m.cloud.databricks.test",
		WorkspaceID:  "987654321",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	})
	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:     "blank-auth",
		Host:        "https://blank-auth.cloud.databricks.test",
		WorkspaceID: "111222333",
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", dockerDir)
	t.Setenv("HOME", homeDir)

	for _, profileName := range []string{"pat", "m2m", "blank-auth"} {
		t.Run(profileName, func(t *testing.T) {
			cmd := newConfigureDockerTestCommand(ctx, "configure-docker", profileName, "--region", "us-west-2")
			err := cmd.Execute()
			require.ErrorContains(t, err, "requires a profile created by databricks auth login")
			require.NoFileExists(t, filepath.Join(dockerDir, "config.json"))
		})
	}
}

func TestConfigureDockerCommandRejectsExplicitInheritedFlags(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:     "DEFAULT",
		Host:        "https://workspace.cloud.databricks.test",
		WorkspaceID: "123456789",
		AuthType:    authTypeDatabricksCLI,
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", filepath.Join(dir, "docker"))
	t.Setenv("HOME", filepath.Join(dir, "home"))

	cases := [][]string{
		{"configure-docker", "DEFAULT", "--region", "us-west-2", "--host", "https://other.cloud.databricks.test"},
		{"configure-docker", "DEFAULT", "--region", "us-west-2", "--account-id", "abc"},
		{"configure-docker", "DEFAULT", "--region", "us-west-2", "--workspace-id", "987654321"},
	}

	for _, args := range cases {
		t.Run(args[len(args)-2], func(t *testing.T) {
			cmd := newConfigureDockerTestCommand(ctx, args...)
			err := cmd.Execute()
			require.ErrorContains(t, err, "is not supported for configure-docker")
		})
	}
}

func TestConfigureDockerCommandRejectsAmbiguousWorkspaceIDBeforeDockerConfig(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	dockerDir := filepath.Join(dir, "docker")

	for _, name := range []string{"one", "two"} {
		writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
			Profile:     name,
			Host:        "https://" + name + ".cloud.databricks.test",
			WorkspaceID: "123456789",
			AuthType:    authTypeDatabricksCLI,
		})
	}

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", dockerDir)
	t.Setenv("HOME", filepath.Join(dir, "home"))

	deps := defaultConfigureDockerDeps()
	deps.installShim = func(context.Context, string, string) (dockercredentials.ShimInstallResult, error) {
		t.Fatal("installShim should not be called")
		return dockercredentials.ShimInstallResult{}, nil
	}
	deps.setCredentialHelper = func(string, string) error {
		t.Fatal("setCredentialHelper should not be called")
		return nil
	}

	cmd := newConfigureDockerTestCommandWithDeps(ctx, deps, "configure-docker", "one", "--region", "us-west-2")
	err := cmd.Execute()
	require.ErrorContains(t, err, "multiple Databricks profiles match workspace ID 123456789")
	require.NoFileExists(t, filepath.Join(dockerDir, "config.json"))
}

func TestConfigureDockerCommandInstallsShimBeforeDockerConfig(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	dockerDir := filepath.Join(dir, "docker")
	workspaceHost := "https://workspace.cloud.databricks.test"

	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:     "DEFAULT",
		Host:        workspaceHost,
		WorkspaceID: "123456789",
		AuthType:    authTypeDatabricksCLI,
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", dockerDir)
	t.Setenv("HOME", filepath.Join(dir, "home"))

	deps := defaultConfigureDockerDeps()
	deps.executable = func() (string, error) {
		return "/usr/local/bin/databricks", nil
	}
	deps.registryHost = configureDockerRegistryHostStub(t, "123456789", "us-west-2", workspaceHost, "123456789.container.us-west-2.cloud.databricks.test")
	deps.installShim = func(context.Context, string, string) (dockercredentials.ShimInstallResult, error) {
		return dockercredentials.ShimInstallResult{}, errors.New("install failed")
	}
	deps.setCredentialHelper = func(string, string) error {
		t.Fatal("setCredentialHelper should not be called after install failure")
		return nil
	}

	cmd := newConfigureDockerTestCommandWithDeps(ctx, deps, "configure-docker", "DEFAULT", "--region", "us-west-2")
	err := cmd.Execute()
	require.ErrorContains(t, err, "install failed")
	require.NoFileExists(t, filepath.Join(dockerDir, "config.json"))
}
