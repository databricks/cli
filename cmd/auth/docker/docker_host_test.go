package docker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	cmdroot "github.com/databricks/cli/cmd/root"
	authlib "github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dockerHostTestRegistryHost  = "123456789.container.us-west-2.cloud.databricks.test"
	dockerHostTestWorkspaceHost = "https://workspace.cloud.databricks.test"
)

func dockerHostTestDeps(t *testing.T, p profile.Profile) dockerHostDeps {
	t.Helper()

	deps := defaultDockerHostDeps()
	deps.profiler = profile.InMemoryProfiler{Profiles: profile.Profiles{p}}
	deps.validateWorkspaceHost = func(string) error { return nil }
	deps.executable = func() (string, error) { return "/usr/local/bin/databricks", nil }
	deps.newWorkspaceClient = func(cfg *databricks.Config) (*databricks.WorkspaceClient, error) {
		return &databricks.WorkspaceClient{Config: (*config.Config)(cfg)}, nil
	}
	deps.resolveWorkspaceRegion = func(context.Context, *databricks.WorkspaceClient) (string, error) {
		return "us-west-2", nil
	}
	deps.registryHost = func(workspaceID, region, workspaceHost string) (string, error) {
		assert.Equal(t, "123456789", workspaceID)
		assert.Equal(t, "us-west-2", region)
		assert.Equal(t, dockerHostTestWorkspaceHost, workspaceHost)
		return dockerHostTestRegistryHost, nil
	}
	return deps
}

func dockerHostTestProfile() profile.Profile {
	return profile.Profile{
		Name:        "workspace",
		Host:        dockerHostTestWorkspaceHost,
		WorkspaceID: "123456789",
		AuthType:    authlib.AuthTypeDatabricksCli,
	}
}

func TestDockerHostCommandReportsCredentialHelperStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		helper     string
		configured string
	}{
		{name: "configured", helper: "databricks", configured: "YES"},
		{name: "other helper", helper: "desktop", configured: "NO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dockerDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte(`{
  "credHelpers": {
    "`+dockerHostTestRegistryHost+`": "`+tt.helper+`"
  }
}`), 0o600))

			ctx := env.Set(t.Context(), "DOCKER_CONFIG", dockerDir)
			stdout, err := executeDockerHostCommand(ctx, dockerHostTestDeps(t, dockerHostTestProfile()), "--profile", "workspace")

			require.NoError(t, err)
			assert.Equal(t, "Registry host: "+dockerHostTestRegistryHost+"\nCredential helper configured: "+tt.configured+"\n", stdout)
		})
	}
}

func TestDockerHostCommandRendersJSON(t *testing.T) {
	t.Parallel()

	ctx := env.Set(t.Context(), "DOCKER_CONFIG", t.TempDir())
	stdout, err := executeDockerHostCommand(ctx, dockerHostTestDeps(t, dockerHostTestProfile()), "--profile", "workspace", "--output", "json")

	require.NoError(t, err)
	assert.JSONEq(t, `{
  "host": "123456789.container.us-west-2.cloud.databricks.test",
  "configured": false
}`, stdout)
}

func TestDockerHostCommandRequiresProfileFlag(t *testing.T) {
	t.Parallel()

	_, err := executeDockerHostCommand(t.Context(), defaultDockerHostDeps())
	assert.ErrorContains(t, err, "--profile is required for auth docker host")
}

func TestDockerHostCommandRejectsOtherAuthSelectionFlags(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"host", "account-id", "workspace-id"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := executeDockerHostCommand(t.Context(), defaultDockerHostDeps(), "--profile", "workspace", "--"+name, "value")
			assert.ErrorContains(t, err, "--"+name+" is not supported for auth docker host")
		})
	}
}

func TestDockerHostCommandResolvesMissingWorkspaceIDWithoutPersistingIt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	profileContents := []byte(`[workspace]
host = ` + dockerHostTestWorkspaceHost + `
auth_type = databricks-cli
workspace_id = none
`)
	require.NoError(t, os.WriteFile(configFile, profileContents, 0o600))

	p := dockerHostTestProfile()
	p.WorkspaceID = authlib.WorkspaceIDNone
	deps := dockerHostTestDeps(t, p)
	deps.resolveWorkspaceID = func(context.Context, *databricks.WorkspaceClient) (string, error) {
		return "123456789", nil
	}
	deps.resolveWorkspaceRegion = func(_ context.Context, w *databricks.WorkspaceClient) (string, error) {
		assert.Equal(t, "123456789", w.Config.WorkspaceID)
		return "us-west-2", nil
	}

	ctx := env.Set(t.Context(), "DATABRICKS_CONFIG_FILE", configFile)
	ctx = env.Set(ctx, "DOCKER_CONFIG", filepath.Join(dir, "docker"))
	_, err := executeDockerHostCommand(ctx, deps, "--profile", "workspace")
	require.NoError(t, err)

	got, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.Equal(t, string(profileContents), string(got))
}

func TestDockerHostCommandReturnsMetastoreError(t *testing.T) {
	t.Parallel()

	metastoreErr := errors.New("summary failed")
	deps := dockerHostTestDeps(t, dockerHostTestProfile())
	deps.resolveWorkspaceRegion = func(context.Context, *databricks.WorkspaceClient) (string, error) {
		return "", metastoreErr
	}

	_, err := executeDockerHostCommand(t.Context(), deps, "--profile", "workspace")
	assert.ErrorContains(t, err, `resolve workspace region for profile "workspace"`)
	assert.ErrorIs(t, err, metastoreErr)
}

func newDockerHostTestRoot(ctx context.Context, deps dockerHostDeps) *cobra.Command {
	ctx = cmdctx.GenerateExecId(ctx)
	cmd := cmdroot.New(ctx)
	authCmd := &cobra.Command{Use: "auth"}
	authCmd.PersistentFlags().String("host", "", "Databricks Host")
	authCmd.PersistentFlags().String("account-id", "", "Databricks Account ID")
	authCmd.PersistentFlags().String("workspace-id", "", "Databricks Workspace ID")
	dockerCmd := &cobra.Command{Use: "docker"}
	dockerCmd.AddCommand(newDockerHostCommandWithDeps(deps))
	authCmd.AddCommand(dockerCmd)
	cmd.AddCommand(authCmd)
	return cmd
}

func executeDockerHostCommand(ctx context.Context, deps dockerHostDeps, args ...string) (string, error) {
	cmd := newDockerHostTestRoot(ctx, deps)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(append([]string{"auth", "docker", "host"}, args...))
	err := cmd.Execute()
	return stdout.String(), err
}
