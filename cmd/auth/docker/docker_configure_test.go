package docker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	authlib "github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDockerConfigureTestCommand(ctx context.Context, args ...string) *cobra.Command {
	return newDockerConfigureTestCommandWithDeps(ctx, defaultConfigureDockerDeps(), args...)
}

func newDockerConfigureTestCommandWithDeps(ctx context.Context, deps configureDockerDeps, args ...string) *cobra.Command {
	cmd := &cobra.Command{Use: "auth"}
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.PersistentFlags().String("host", "", "Databricks Host")
	cmd.PersistentFlags().String("account-id", "", "Databricks Account ID")
	cmd.PersistentFlags().String("workspace-id", "", "Databricks Workspace ID")
	dockerCmd := &cobra.Command{Use: "docker"}
	dockerCmd.AddCommand(newDockerConfigureCommandWithDeps(deps))
	cmd.AddCommand(dockerCmd)
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
		assert.Equal(t, wantWorkspaceID, workspaceID)
		assert.Equal(t, wantRegion, region)
		assert.Equal(t, wantWorkspaceHost, workspaceHost)
		return registryHost, nil
	}
}

func configureDockerRegionStub(region string) func(context.Context, *databricks.WorkspaceClient) (string, error) {
	return func(context.Context, *databricks.WorkspaceClient) (string, error) {
		return region, nil
	}
}

func allowConfigureDockerWorkspaceHost(string) error {
	return nil
}

func writeConfigureDockerExecutable(t *testing.T, dir string) string {
	t.Helper()
	name := "databricks"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(path, []byte("databricks executable"), 0o755))
	return path
}

func TestConfigureDockerCommandUsesExplicitRegionWhileResolvingWorkspaceID(t *testing.T) {
	t.Parallel()

	ctx := env.Set(cmdio.MockDiscard(t.Context()), "DOCKER_CONFIG", t.TempDir())
	workspaceHost := "https://workspace.cloud.databricks.test"
	stopErr := errors.New("stop after registry host")
	deps := defaultConfigureDockerDeps()
	deps.profiler = profile.InMemoryProfiler{Profiles: profile.Profiles{{
		Name:     "DEFAULT",
		Host:     workspaceHost,
		AuthType: authlib.AuthTypeDatabricksCli,
	}}}
	deps.validateWorkspaceHost = allowConfigureDockerWorkspaceHost
	deps.executable = func() (string, error) { return "/usr/local/bin/databricks", nil }
	deps.newWorkspaceClient = func(cfg *databricks.Config) (*databricks.WorkspaceClient, error) {
		return &databricks.WorkspaceClient{Config: (*config.Config)(cfg)}, nil
	}
	deps.resolveWorkspaceID = func(context.Context, *databricks.WorkspaceClient) (string, error) {
		return "123456789", nil
	}
	deps.resolveWorkspaceRegion = func(context.Context, *databricks.WorkspaceClient) (string, error) {
		return "", errors.New("metastore summary called")
	}
	deps.registryHost = func(workspaceID, region, host string) (string, error) {
		assert.Equal(t, "123456789", workspaceID)
		assert.Equal(t, "us-west-2", region)
		assert.Equal(t, workspaceHost, host)
		return "", stopErr
	}

	cmd := newDockerConfigureTestCommandWithDeps(ctx, deps, "docker", "configure", "DEFAULT", "--region", "us-west-2")
	err := cmd.Execute()
	assert.ErrorIs(t, err, stopErr)
}

func TestConfigureDockerCommandRejectsInvalidExplicitRegionBeforeWorkspaceClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		region string
		want   string
	}{
		{name: "empty", region: "", want: "region is required"},
		{name: "whitespace", region: "   ", want: "region is required"},
		{name: "invalid DNS label", region: "-us-west-2", want: `invalid region "-us-west-2"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := env.Set(cmdio.MockDiscard(t.Context()), "DOCKER_CONFIG", t.TempDir())
			workspaceClientErr := errors.New("workspace client created")
			deps := defaultConfigureDockerDeps()
			deps.profiler = profile.InMemoryProfiler{Profiles: profile.Profiles{{
				Name:     "DEFAULT",
				Host:     "https://workspace.cloud.databricks.test",
				AuthType: authlib.AuthTypeDatabricksCli,
			}}}
			deps.validateWorkspaceHost = allowConfigureDockerWorkspaceHost
			deps.executable = func() (string, error) { return "/usr/local/bin/databricks", nil }
			deps.newWorkspaceClient = func(*databricks.Config) (*databricks.WorkspaceClient, error) {
				return nil, workspaceClientErr
			}

			cmd := newDockerConfigureTestCommandWithDeps(ctx, deps, "docker", "configure", "DEFAULT", "--region", tt.region)
			err := cmd.Execute()
			assert.ErrorContains(t, err, tt.want)
			assert.NotErrorIs(t, err, workspaceClientErr)
		})
	}
}

func TestConfigureDockerCommandReturnsMetastoreSummaryErrorBeforeMutation(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dockerDir := t.TempDir()
	server := testserver.New(t)
	server.Handle("GET", "/api/2.1/unity-catalog/metastore_summary", func(testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusInternalServerError,
			Body: map[string]any{
				"error_code": "INTERNAL_ERROR",
				"message":    "summary failed",
			},
		}
	})

	t.Setenv("DOCKER_CONFIG", dockerDir)
	deps := defaultConfigureDockerDeps()
	deps.validateWorkspaceHost = allowConfigureDockerWorkspaceHost
	deps.profiler = profile.InMemoryProfiler{Profiles: profile.Profiles{{
		Name:        "DEFAULT",
		Host:        "https://workspace.cloud.databricks.test",
		WorkspaceID: "123456789",
		AuthType:    authlib.AuthTypeDatabricksCli,
	}}}
	deps.executable = func() (string, error) { return "/usr/local/bin/databricks", nil }
	deps.newWorkspaceClient = func(cfg *databricks.Config) (*databricks.WorkspaceClient, error) {
		cfg.Host = server.URL
		cfg.Token = "test-token"
		cfg.AuthType = "pat"
		cfg.Profile = ""
		return databricks.NewWorkspaceClient(cfg)
	}
	deps.installShim = func(string) (dockercredentials.ShimInstallResult, error) {
		t.Fatal("installShim should not be called")
		return dockercredentials.ShimInstallResult{}, nil
	}

	cmd := newDockerConfigureTestCommandWithDeps(ctx, deps, "docker", "configure", "DEFAULT")
	err := cmd.Execute()
	assert.ErrorContains(t, err, `resolve workspace region for profile "DEFAULT": summary failed`)
	assert.NoFileExists(t, filepath.Join(dockerDir, "config.json"))
}

func TestConfigureDockerCommandRejectsEmptyMetastoreRegionBeforeMutation(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dockerDir := t.TempDir()
	server := testserver.New(t)
	server.Handle("GET", "/api/2.1/unity-catalog/metastore_summary", func(testserver.Request) any {
		return map[string]any{"metastore_id": "metastore-id"}
	})

	t.Setenv("DOCKER_CONFIG", dockerDir)
	deps := defaultConfigureDockerDeps()
	deps.validateWorkspaceHost = allowConfigureDockerWorkspaceHost
	deps.profiler = profile.InMemoryProfiler{Profiles: profile.Profiles{{
		Name:        "DEFAULT",
		Host:        "https://workspace.cloud.databricks.test",
		WorkspaceID: "123456789",
		AuthType:    authlib.AuthTypeDatabricksCli,
	}}}
	deps.executable = func() (string, error) { return "/usr/local/bin/databricks", nil }
	deps.newWorkspaceClient = func(cfg *databricks.Config) (*databricks.WorkspaceClient, error) {
		cfg.Host = server.URL
		cfg.Token = "test-token"
		cfg.AuthType = "pat"
		cfg.Profile = ""
		return databricks.NewWorkspaceClient(cfg)
	}
	deps.installShim = func(string) (dockercredentials.ShimInstallResult, error) {
		t.Fatal("installShim should not be called")
		return dockercredentials.ShimInstallResult{}, nil
	}

	cmd := newDockerConfigureTestCommandWithDeps(ctx, deps, "docker", "configure", "DEFAULT")
	err := cmd.Execute()
	assert.ErrorContains(t, err, `resolve workspace region for profile "DEFAULT": metastore summary did not include a region`)
	assert.NoFileExists(t, filepath.Join(dockerDir, "config.json"))
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
		AuthType:  authlib.AuthTypeDatabricksCli,
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", dockerDir)

	cmd := newDockerConfigureTestCommand(ctx, "docker", "configure", "account")
	err := cmd.Execute()
	assert.ErrorContains(t, err, "databricks auth login --host <workspace-url>")
	assert.NoFileExists(t, filepath.Join(dockerDir, "config.json"))
}

func TestConfigureDockerCommandPersistsResolvedWorkspaceID(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	dockerDir := filepath.Join(dir, "docker")
	homeDir := filepath.Join(dir, "home")
	workspaceHost := "https://workspace.gcp.databricks.test"

	writeConfigureDockerProfile(t, ctx, configFile, &config.Config{
		Profile:     "workspace",
		Host:        workspaceHost,
		WorkspaceID: authlib.WorkspaceIDNone,
		AuthType:    authlib.AuthTypeDatabricksCli,
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DATABRICKS_WORKSPACE_ID", "ambient-workspace")
	t.Setenv("DOCKER_CONFIG", dockerDir)
	t.Setenv("HOME", homeDir)

	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
		assert.Empty(t, req.Headers.Get(authlib.WorkspaceIDHeader))
		return testserver.Response{
			Headers: http.Header{"X-Databricks-Org-Id": {"999999"}},
			Body:    map[string]any{},
		}
	})
	testserver.AddDefaultHandlers(server)
	server.Handle("GET", "/api/2.1/unity-catalog/metastore_summary", func(req testserver.Request) any {
		assert.Equal(t, "999999", req.Headers.Get(authlib.WorkspaceIDHeader))
		return map[string]any{"region": "us-west-2"}
	})

	deps := defaultConfigureDockerDeps()
	deps.validateWorkspaceHost = allowConfigureDockerWorkspaceHost
	databricksPath := writeConfigureDockerExecutable(t, filepath.Join(dir, "bin"))
	deps.executable = func() (string, error) {
		return databricksPath, nil
	}
	deps.newWorkspaceClient = func(cfg *databricks.Config) (*databricks.WorkspaceClient, error) {
		assert.Equal(t, databricksPath, cfg.DatabricksCliPath)
		require.NoError(t, (*config.Config)(cfg).EnsureResolved())
		assert.Equal(t, authlib.WorkspaceIDNone, cfg.WorkspaceID)
		cfg.Host = server.URL
		cfg.Token = "test-token"
		cfg.AuthType = "pat"
		cfg.Profile = ""
		return databricks.NewWorkspaceClient(cfg)
	}
	deps.registryHost = configureDockerRegistryHostStub(t, "999999", "us-west-2", workspaceHost, "999999.container.us-west-2.gcp.databricks.test")

	cmd := newDockerConfigureTestCommandWithDeps(ctx, deps, "docker", "configure", "workspace")
	require.NoError(t, cmd.Execute())

	raw, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "workspace_id = 999999")

	helpers := readCredentialHelpers(t, filepath.Join(dockerDir, "config.json"))
	assert.Equal(t, dockercredentials.HelperName, helpers["999999.container.us-west-2.gcp.databricks.test"])
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
			cmd := newDockerConfigureTestCommand(ctx, "docker", "configure", profileName)
			err := cmd.Execute()
			assert.ErrorContains(t, err, "requires a profile created by databricks auth login")
			assert.NoFileExists(t, filepath.Join(dockerDir, "config.json"))
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
		AuthType:    authlib.AuthTypeDatabricksCli,
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", filepath.Join(dir, "docker"))
	t.Setenv("HOME", filepath.Join(dir, "home"))

	cases := [][]string{
		{"docker", "configure", "DEFAULT", "--host", "https://other.cloud.databricks.test"},
		{"docker", "configure", "DEFAULT", "--account-id", "abc"},
		{"docker", "configure", "DEFAULT", "--workspace-id", "987654321"},
	}

	for _, args := range cases {
		t.Run(args[len(args)-2], func(t *testing.T) {
			cmd := newDockerConfigureTestCommand(ctx, args...)
			err := cmd.Execute()
			assert.ErrorContains(t, err, "is not supported for auth docker configure")
		})
	}
}

func TestConfigureDockerRejectsSameWorkspaceIDInDifferentEnvironment(t *testing.T) {
	p := profile.Profile{
		Name:        "prod",
		Host:        "https://workspace.cloud.databricks.test",
		WorkspaceID: "123456789",
		AuthType:    authlib.AuthTypeDatabricksCli,
	}
	profiler := profile.InMemoryProfiler{Profiles: profile.Profiles{
		p,
		{
			Name:        "dev",
			Host:        "https://workspace.dev.cloud.databricks.test",
			WorkspaceID: p.WorkspaceID,
			AuthType:    authlib.AuthTypeDatabricksCli,
		},
	}}
	err := ensureConfigureDockerUniqueProfile(t.Context(), profiler, p, p.WorkspaceID)
	assert.ErrorContains(t, err, "multiple Databricks profiles match workspace ID 123456789: prod and dev")
}

func TestConfigureDockerAllowsUnsupportedDuplicateProfile(t *testing.T) {
	p := profile.Profile{
		Name:        "workspace",
		Host:        "https://workspace.cloud.databricks.test",
		WorkspaceID: "123456789",
		AuthType:    authlib.AuthTypeDatabricksCli,
	}
	profiler := profile.InMemoryProfiler{Profiles: profile.Profiles{
		p,
		{
			Name:                 "m2m",
			Host:                 p.Host,
			WorkspaceID:          p.WorkspaceID,
			HasClientCredentials: true,
		},
	}}
	err := ensureConfigureDockerUniqueProfile(t.Context(), profiler, p, p.WorkspaceID)
	require.NoError(t, err)
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
		AuthType:    authlib.AuthTypeDatabricksCli,
	})

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("DOCKER_CONFIG", dockerDir)
	t.Setenv("HOME", filepath.Join(dir, "home"))

	deps := defaultConfigureDockerDeps()
	deps.validateWorkspaceHost = allowConfigureDockerWorkspaceHost
	deps.resolveWorkspaceRegion = configureDockerRegionStub("us-west-2")
	deps.executable = func() (string, error) {
		return "/usr/local/bin/databricks", nil
	}
	deps.registryHost = configureDockerRegistryHostStub(t, "123456789", "us-west-2", workspaceHost, "123456789.container.us-west-2.cloud.databricks.test")
	deps.installShim = func(string) (dockercredentials.ShimInstallResult, error) {
		return dockercredentials.ShimInstallResult{}, errors.New("install failed")
	}
	deps.setCredentialHelper = func(string, string) error {
		t.Fatal("setCredentialHelper should not be called after install failure")
		return nil
	}

	cmd := newDockerConfigureTestCommandWithDeps(ctx, deps, "docker", "configure", "DEFAULT")
	err := cmd.Execute()
	assert.ErrorContains(t, err, "install failed")
	assert.NoFileExists(t, filepath.Join(dockerDir, "config.json"))
}

func TestConfigureDockerCommandWarnsAboutPATHAndPATHEXT(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	workspaceHost := "https://workspace.cloud.databricks.test"
	registryHost := "123456789.container.us-west-2.cloud.databricks.test"
	t.Setenv("DOCKER_CONFIG", t.TempDir())

	deps := defaultConfigureDockerDeps()
	deps.validateWorkspaceHost = allowConfigureDockerWorkspaceHost
	deps.resolveWorkspaceRegion = configureDockerRegionStub("us-west-2")
	deps.profiler = profile.InMemoryProfiler{Profiles: profile.Profiles{
		{
			Name:        "DEFAULT",
			Host:        workspaceHost,
			WorkspaceID: "123456789",
			AuthType:    authlib.AuthTypeDatabricksCli,
		},
	}}
	deps.executable = func() (string, error) {
		return "/usr/local/bin/databricks", nil
	}
	deps.newWorkspaceClient = func(cfg *databricks.Config) (*databricks.WorkspaceClient, error) {
		return &databricks.WorkspaceClient{Config: (*config.Config)(cfg)}, nil
	}
	deps.registryHost = configureDockerRegistryHostStub(t, "123456789", "us-west-2", workspaceHost, registryHost)
	deps.installShim = func(string) (dockercredentials.ShimInstallResult, error) {
		return dockercredentials.ShimInstallResult{
			Path:   "/usr/local/bin/docker-credential-databricks",
			OnPath: false,
		}, nil
	}
	deps.setCredentialHelper = func(string, string) error {
		return nil
	}

	cmd := newDockerConfigureTestCommandWithDeps(ctx, deps, "docker", "configure", "DEFAULT")
	require.NoError(t, cmd.Execute())
	assert.Contains(t, stderr.String(), "PATH")
	assert.Contains(t, stderr.String(), ".CMD is in PATHEXT on Windows")
}
