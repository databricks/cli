package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func newTestDockerTokenCommand(t *testing.T, load tokenLoader) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	require.NoError(t, databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile:  configFile,
		Profile:     "workspace",
		Host:        "https://workspace.cloud.databricks.test",
		WorkspaceID: "123456789",
		AuthType:    authTypeDatabricksCLI,
	}))

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	stdout := &bytes.Buffer{}
	cmd := newDockerTokenCommandWithTokenLoader(&auth.AuthArguments{}, load)
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))
	cmd.SetOut(stdout)
	return cmd, stdout
}

func TestDockerTokenEmitsGetResponse(t *testing.T) {
	var gotProfile string
	loadToken := func(_ context.Context, args loadTokenArgs) (*oauth2.Token, error) {
		gotProfile = args.profileName
		return &oauth2.Token{AccessToken: "access-token"}, nil
	}

	cmd, stdout := newTestDockerTokenCommand(t, loadToken)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "workspace", gotProfile)

	var got map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	assert.Equal(t, map[string]string{
		"Username": "oauthtoken",
		"Secret":   "access-token",
	}, got)
}

func TestDockerTokenForceRefresh(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "default", want: true},
		{name: "disabled", args: []string{"--no-force-refresh"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bool
			loadToken := func(_ context.Context, args loadTokenArgs) (*oauth2.Token, error) {
				got = args.forceRefresh
				return &oauth2.Token{AccessToken: "access-token"}, nil
			}
			cmd, _ := newTestDockerTokenCommand(t, loadToken)
			cmd.SetArgs(tt.args)

			require.NoError(t, cmd.Execute())
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunDockerTokenUsesConfiguredProfiler(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{
				Name:        "workspace",
				Host:        "https://workspace.cloud.databricks.test",
				WorkspaceID: "123456789",
				AuthType:    authTypeDatabricksCLI,
			},
		},
	}

	var gotProfile string
	loadToken := func(_ context.Context, args loadTokenArgs) (*oauth2.Token, error) {
		gotProfile = args.profileName
		return &oauth2.Token{AccessToken: "access-token"}, nil
	}

	cmd := &cobra.Command{Use: "token"}
	var stdout bytes.Buffer
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))
	cmd.SetOut(&stdout)

	err := runDockerToken(ctx, cmd, loadTokenArgs{
		authArguments: &auth.AuthArguments{},
		profiler:      profiler,
	}, loadToken)
	require.NoError(t, err)
	assert.Equal(t, "workspace", gotProfile)

	var got dockerGetResponse
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
}

func TestRunDockerTokenUsesMatchedProfileAccountID(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{
				Name:        "workspace",
				Host:        "https://workspace.cloud.databricks.test",
				AccountID:   "profile-account",
				WorkspaceID: "123456789",
				AuthType:    authTypeDatabricksCLI,
			},
		},
	}

	cmd := &cobra.Command{Use: "token"}
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))
	cmd.SetOut(&bytes.Buffer{})

	err := runDockerToken(ctx, cmd, loadTokenArgs{
		authArguments: &auth.AuthArguments{},
		profiler:      profiler,
	}, func(_ context.Context, args loadTokenArgs) (*oauth2.Token, error) {
		assert.Equal(t, "profile-account", args.authArguments.AccountID)
		return &oauth2.Token{AccessToken: "access-token"}, nil
	})
	require.NoError(t, err)
}

func TestDockerTokenProfileSelectsByWorkspaceID(t *testing.T) {
	registry := dockercredentials.Registry{
		WorkspaceID: "123456789",
		Host:        "123456789.container.us-west-2.cloud.databricks.test",
	}
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{{
			Name:        "workspace",
			Host:        "https://workspace.dev.cloud.databricks.test",
			WorkspaceID: registry.WorkspaceID,
			AuthType:    authTypeDatabricksCLI,
		}},
	}
	selectedProfile, err := dockerTokenProfile(t.Context(), registry, profiler)
	require.NoError(t, err)
	assert.Equal(t, "workspace", selectedProfile.Name)
}

func TestDockerTokenProfileRejectsDuplicateWorkspaceID(t *testing.T) {
	registry := dockercredentials.Registry{
		WorkspaceID: "123456789",
		Host:        "123456789.container.us-west-2.cloud.databricks.test",
	}
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{
				Name:        "prod",
				Host:        "https://workspace.cloud.databricks.test",
				WorkspaceID: registry.WorkspaceID,
				AuthType:    authTypeDatabricksCLI,
			},
			{
				Name:        "dev",
				Host:        "https://workspace.dev.cloud.databricks.test",
				WorkspaceID: registry.WorkspaceID,
				AuthType:    authTypeDatabricksCLI,
			},
		},
	}
	_, err := dockerTokenProfile(t.Context(), registry, profiler)
	assert.ErrorContains(t, err, "multiple Databricks profiles match workspace ID 123456789")
	assert.ErrorContains(t, err, "prod and dev")
}

func TestDockerTokenProfileIgnoresUnsupportedDuplicateProfile(t *testing.T) {
	registry := dockercredentials.Registry{
		WorkspaceID: "123456789",
		Host:        "123456789.container.us-west-2.cloud.databricks.test",
	}
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{
				Name:        "workspace",
				Host:        "https://workspace.cloud.databricks.test",
				WorkspaceID: registry.WorkspaceID,
				AuthType:    authTypeDatabricksCLI,
			},
			{
				Name:                 "m2m",
				Host:                 "https://workspace.cloud.databricks.test",
				WorkspaceID:          registry.WorkspaceID,
				HasClientCredentials: true,
			},
		},
	}
	selectedProfile, err := dockerTokenProfile(t.Context(), registry, profiler)
	require.NoError(t, err)
	assert.Equal(t, "workspace", selectedProfile.Name)
}

func TestDockerTokenRejectsPositionalArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	cmd := newDockerTokenCommandWithTokenLoader(&auth.AuthArguments{}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
		t.Fatal("loadToken should not be called")
		return nil, nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))
	cmd.SetArgs([]string{"DEFAULT"})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "auth docker token does not accept positional arguments")
}

func TestDockerTokenValidatesBeforeResolvingTokenStore(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	t.Setenv(storage.EnvVar, "invalid")

	cmd := newDockerTokenCommandWithTokenLoader(&auth.AuthArguments{}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
		t.Fatal("loadToken should not be called")
		return nil, nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"DEFAULT"})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "auth docker token does not accept positional arguments")
}

func TestDockerTokenRejectsAuthSelectionFlags(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	require.NoError(t, databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: configFile,
		Profile:    "DEFAULT",
		Host:       "https://profile.cloud.databricks.test",
		AuthType:   authTypeDatabricksCLI,
	}))
	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	cases := [][]string{
		{"--profile", "DEFAULT"},
		{"--host", "https://workspace.cloud.databricks.test"},
		{"--profile", "DEFAULT", "--host", "https://workspace.cloud.databricks.test"},
		{"--account-id", "abc"},
		{"--workspace-id", "123456789"},
	}

	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var authArgs auth.AuthArguments
			cmd := &cobra.Command{Use: "auth"}
			cmd.PersistentFlags().StringVar(&authArgs.Host, "host", "", "Databricks Host")
			cmd.PersistentFlags().StringVar(&authArgs.AccountID, "account-id", "", "Databricks Account ID")
			cmd.PersistentFlags().StringVar(&authArgs.WorkspaceID, "workspace-id", "", "Databricks Workspace ID")
			dockerCmd := &cobra.Command{Use: "docker"}
			dockerCmd.AddCommand(newDockerTokenCommandWithTokenLoader(&authArgs, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
				t.Fatal("loadToken should not be called")
				return nil, nil
			}))
			cmd.AddCommand(dockerCmd)
			cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
			cmd.SetContext(ctx)
			cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))
			cmd.SetArgs(append([]string{"docker", "token"}, args...))

			err := cmd.Execute()
			assert.ErrorContains(t, err, "auth docker token does not support")
		})
	}
}

func TestDockerTokenRejectsNonDARHost(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	cmd := newDockerTokenCommandWithTokenLoader(&auth.AuthArguments{}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
		t.Fatal("loadToken should not be called")
		return nil, nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("registry.example.com\n"))

	err := cmd.Execute()
	assert.ErrorContains(t, err, "is not a Databricks Artifact Registry host")
}

func TestDockerTokenErrorsWithoutMatchingProfile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	require.NoError(t, os.WriteFile(configFile, []byte(""), 0o600))

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	cmd := newDockerTokenCommandWithTokenLoader(&auth.AuthArguments{}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
		t.Fatal("loadToken should not be called")
		return nil, nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))

	err := cmd.Execute()
	assert.ErrorContains(t, err, "no Databricks profile found for workspace ID 123456789")
	assert.ErrorContains(t, err, "databricks auth login --host <workspace-url>")
	assert.ErrorContains(t, err, "workspace_id")
}

func TestDockerTokenErrorsWithMultipleMatchingProfiles(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	for _, name := range []string{"one", "two"} {
		require.NoError(t, databrickscfg.SaveToProfile(ctx, &config.Config{
			ConfigFile:  configFile,
			Profile:     name,
			Host:        "https://" + name + ".cloud.databricks.test",
			WorkspaceID: "123456789",
			AuthType:    authTypeDatabricksCLI,
		}))
	}

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	cmd := newDockerTokenCommandWithTokenLoader(&auth.AuthArguments{}, func(context.Context, loadTokenArgs) (*oauth2.Token, error) {
		t.Fatal("loadToken should not be called")
		return nil, nil
	})
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader("123456789.container.us-west-2.cloud.databricks.com\n"))

	err := cmd.Execute()
	assert.ErrorContains(t, err, "multiple Databricks profiles match workspace ID 123456789")
	assert.ErrorContains(t, err, "one and two")
	assert.ErrorContains(t, err, "Remove duplicate workspace_id entries")
}
