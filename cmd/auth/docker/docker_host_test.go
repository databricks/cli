package docker

import (
	"bytes"
	"context"
	"errors"
	"io"
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
)

const (
	dockerHostTestWorkspaceID   = "123456789"
	dockerHostTestRegion        = "us-west-2"
	dockerHostTestRegistryHost  = dockerHostTestWorkspaceID + ".container." + dockerHostTestRegion + ".cloud.databricks.test"
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
		return dockerHostTestRegion, nil
	}
	deps.registryHost = func(workspaceID, region, workspaceHost string) (string, error) {
		assert.Equal(t, dockerHostTestWorkspaceID, workspaceID)
		assert.Equal(t, dockerHostTestRegion, region)
		assert.Equal(t, dockerHostTestWorkspaceHost, workspaceHost)
		return dockerHostTestRegistryHost, nil
	}
	return deps
}

func dockerHostTestProfile() profile.Profile {
	return profile.Profile{
		Name:        "workspace",
		Host:        dockerHostTestWorkspaceHost,
		WorkspaceID: dockerHostTestWorkspaceID,
		AuthType:    authlib.AuthTypeDatabricksCli,
	}
}

func TestDockerHostCommandReturnsCredentialHelperLookupError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("read Docker config")
	deps := dockerHostTestDeps(t, dockerHostTestProfile())
	deps.credentialHelperConfigured = func(_, host string) (bool, error) {
		assert.Equal(t, dockerHostTestRegistryHost, host)
		return false, wantErr
	}
	ctx := env.Set(t.Context(), "DOCKER_CONFIG", t.TempDir())

	stdout, err := executeDockerHostCommand(ctx, deps, "--profile", "workspace")

	assert.ErrorIs(t, err, wantErr)
	assert.Empty(t, stdout)
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

func TestDockerHostCommandRewritesInvalidRefreshTokenErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		workspaceID string
		setup       func(*dockerHostDeps, error)
	}{
		{
			name:        "load workspace profile",
			workspaceID: dockerHostTestWorkspaceID,
			setup: func(deps *dockerHostDeps, authErr error) {
				deps.newWorkspaceClient = func(*databricks.Config) (*databricks.WorkspaceClient, error) {
					return nil, authErr
				}
			},
		},
		{
			name:        "resolve workspace ID",
			workspaceID: authlib.WorkspaceIDNone,
			setup: func(deps *dockerHostDeps, authErr error) {
				deps.resolveWorkspaceID = func(context.Context, *databricks.WorkspaceClient) (string, error) {
					return "", authErr
				}
			},
		},
		{
			name:        "resolve workspace region",
			workspaceID: dockerHostTestWorkspaceID,
			setup: func(deps *dockerHostDeps, authErr error) {
				deps.resolveWorkspaceRegion = func(context.Context, *databricks.WorkspaceClient) (string, error) {
					return "", authErr
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := dockerHostTestProfile()
			p.WorkspaceID = tt.workspaceID
			authErr := &invalidRefreshTokenTestError{error: errors.New("raw token refresh error")}
			deps := dockerHostTestDeps(t, p)
			tt.setup(&deps, authErr)

			_, err := executeDockerHostCommand(t.Context(), deps, "--profile", "workspace")
			assert.EqualError(t, err, `A new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run the following command:
  $ databricks auth login --profile workspace`)
			assert.ErrorIs(t, err, authErr)
		})
	}
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
