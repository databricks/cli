package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetWorkspaceAuthStatus(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	showSensitive := false

	currentUserApi := m.GetMockCurrentUserAPI()
	currentUserApi.EXPECT().Me(mock.Anything).Return(&iam.User{
		UserName: "test-user",
	}, nil)

	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")
	err := cmd.Flag("profile").Value.Set("my-profile")
	require.NoError(t, err)
	cmd.Flag("profile").Changed = true

	cfg := &config.Config{
		Profile: "my-profile",
	}
	m.WorkspaceClient.Config = cfg
	t.Setenv("DATABRICKS_AUTH_TYPE", "azure-cli")
	err = config.ConfigAttributes.Configure(cfg)
	require.NoError(t, err)

	status, err := getAuthStatus(cmd, []string{}, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, bool, error) {
		err := config.ConfigAttributes.ResolveFromStringMap(cfg, map[string]string{
			"host":      "https://test.com",
			"token":     "test-token",
			"auth_type": "azure-cli",
		})
		require.NoError(t, err)
		return cfg, false, nil
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, "success", status.Status)
	require.Equal(t, "test-user", status.Username)
	require.Equal(t, "https://test.com", status.Details.Host)
	require.Equal(t, "azure-cli", status.Details.AuthType)

	require.Equal(t, "azure-cli", status.Details.Configuration["auth_type"].Value)
	require.Equal(t, "DATABRICKS_AUTH_TYPE environment variable", status.Details.Configuration["auth_type"].Source.String())
	require.False(t, status.Details.Configuration["auth_type"].AuthTypeMismatch)

	require.Equal(t, "********", status.Details.Configuration["token"].Value)
	require.Equal(t, "dynamic configuration", status.Details.Configuration["token"].Source.String())
	require.True(t, status.Details.Configuration["token"].AuthTypeMismatch)

	require.Equal(t, "my-profile", status.Details.Configuration["profile"].Value)
	require.Equal(t, "--profile flag", status.Details.Configuration["profile"].Source.String())
	require.False(t, status.Details.Configuration["profile"].AuthTypeMismatch)
}

func TestGetWorkspaceAuthStatusError(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	showSensitive := false

	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")
	err := cmd.Flag("profile").Value.Set("my-profile")
	require.NoError(t, err)
	cmd.Flag("profile").Changed = true

	cfg := &config.Config{
		Profile: "my-profile",
	}
	m.WorkspaceClient.Config = cfg
	t.Setenv("DATABRICKS_AUTH_TYPE", "azure-cli")
	err = config.ConfigAttributes.Configure(cfg)
	require.NoError(t, err)

	status, err := getAuthStatus(cmd, []string{}, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, bool, error) {
		err = config.ConfigAttributes.ResolveFromStringMap(cfg, map[string]string{
			"host":      "https://test.com",
			"token":     "test-token",
			"auth_type": "azure-cli",
		})
		return cfg, false, errors.New("auth error")
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, "error", status.Status)

	require.Equal(t, "azure-cli", status.Details.Configuration["auth_type"].Value)
	require.Equal(t, "DATABRICKS_AUTH_TYPE environment variable", status.Details.Configuration["auth_type"].Source.String())
	require.False(t, status.Details.Configuration["auth_type"].AuthTypeMismatch)

	require.Equal(t, "********", status.Details.Configuration["token"].Value)
	require.Equal(t, "dynamic configuration", status.Details.Configuration["token"].Source.String())
	require.True(t, status.Details.Configuration["token"].AuthTypeMismatch)

	require.Equal(t, "my-profile", status.Details.Configuration["profile"].Value)
	require.Equal(t, "--profile flag", status.Details.Configuration["profile"].Source.String())
	require.False(t, status.Details.Configuration["profile"].AuthTypeMismatch)
}

func TestGetWorkspaceAuthStatusSensitive(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	showSensitive := true

	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")
	err := cmd.Flag("profile").Value.Set("my-profile")
	require.NoError(t, err)
	cmd.Flag("profile").Changed = true

	cfg := &config.Config{
		Profile: "my-profile",
	}
	m.WorkspaceClient.Config = cfg
	t.Setenv("DATABRICKS_AUTH_TYPE", "azure-cli")
	err = config.ConfigAttributes.Configure(cfg)
	require.NoError(t, err)

	status, err := getAuthStatus(cmd, []string{}, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, bool, error) {
		err = config.ConfigAttributes.ResolveFromStringMap(cfg, map[string]string{
			"host":      "https://test.com",
			"token":     "test-token",
			"auth_type": "azure-cli",
		})
		return cfg, false, errors.New("auth error")
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, "error", status.Status)

	require.Equal(t, "azure-cli", status.Details.Configuration["auth_type"].Value)
	require.Equal(t, "DATABRICKS_AUTH_TYPE environment variable", status.Details.Configuration["auth_type"].Source.String())
	require.False(t, status.Details.Configuration["auth_type"].AuthTypeMismatch)

	require.Equal(t, "test-token", status.Details.Configuration["token"].Value)
	require.Equal(t, "dynamic configuration", status.Details.Configuration["token"].Source.String())
	require.True(t, status.Details.Configuration["token"].AuthTypeMismatch)
}

func TestGetAccountAuthStatus(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockAccountClient(t)
	ctx = cmdctx.SetAccountClient(ctx, m.AccountClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	showSensitive := false

	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")
	err := cmd.Flag("profile").Value.Set("my-profile")
	require.NoError(t, err)
	cmd.Flag("profile").Changed = true

	cfg := &config.Config{
		Profile: "my-profile",
	}
	m.AccountClient.Config = cfg
	t.Setenv("DATABRICKS_AUTH_TYPE", "azure-cli")
	err = config.ConfigAttributes.Configure(cfg)
	require.NoError(t, err)

	wsApi := m.GetMockWorkspacesAPI()
	wsApi.EXPECT().List(mock.Anything).Return(nil, nil)

	status, err := getAuthStatus(cmd, []string{}, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, bool, error) {
		err = config.ConfigAttributes.ResolveFromStringMap(cfg, map[string]string{
			"account_id": "test-account-id",
			"username":   "test-user",
			"host":       "https://test.com",
			"token":      "test-token",
			"auth_type":  "azure-cli",
		})
		return cfg, true, nil
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, "success", status.Status)

	require.Equal(t, "test-user", status.Username)
	require.Equal(t, "https://test.com", status.Details.Host)
	require.Equal(t, "azure-cli", status.Details.AuthType)
	require.Equal(t, "test-account-id", status.AccountID)

	require.Equal(t, "azure-cli", status.Details.Configuration["auth_type"].Value)
	require.Equal(t, "DATABRICKS_AUTH_TYPE environment variable", status.Details.Configuration["auth_type"].Source.String())
	require.False(t, status.Details.Configuration["auth_type"].AuthTypeMismatch)

	require.Equal(t, "********", status.Details.Configuration["token"].Value)
	require.Equal(t, "dynamic configuration", status.Details.Configuration["token"].Source.String())
	require.True(t, status.Details.Configuration["token"].AuthTypeMismatch)

	require.Equal(t, "my-profile", status.Details.Configuration["profile"].Value)
	require.Equal(t, "--profile flag", status.Details.Configuration["profile"].Source.String())
	require.False(t, status.Details.Configuration["profile"].AuthTypeMismatch)
}
