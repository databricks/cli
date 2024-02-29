package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/databricks/cli/cmd/root"
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
	ctx = root.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	showSensitive := false

	currentUserApi := m.GetMockCurrentUserAPI()
	currentUserApi.EXPECT().Me(mock.Anything).Return(&iam.User{
		UserName: "test-user",
	}, nil)

	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.Flag("profile").Value.Set("my-profile")
	cmd.Flag("profile").Changed = true

	cfg := &config.Config{
		Profile: "my-profile",
	}
	m.WorkspaceClient.Config = cfg
	t.Setenv("DATABRICKS_AUTH_TYPE", "azure-cli")

	status, err := getWorkspaceAuthStatus(cmd, []string{}, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, error) {
		config.ConfigAttributes.ResolveFromStringMap(cfg, map[string]string{
			"host":      "https://test.com",
			"token":     "test-token",
			"auth_type": "azure-cli",
		})
		return cfg, nil
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, "success", status.Status)
	require.Equal(t, "test-user", status.Details.Username)
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
	ctx = root.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	showSensitive := false

	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.Flag("profile").Value.Set("my-profile")
	cmd.Flag("profile").Changed = true

	cfg := &config.Config{
		Profile: "my-profile",
	}
	m.WorkspaceClient.Config = cfg
	t.Setenv("DATABRICKS_AUTH_TYPE", "azure-cli")

	status, err := getWorkspaceAuthStatus(cmd, []string{}, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, error) {
		config.ConfigAttributes.ResolveFromStringMap(cfg, map[string]string{
			"host":      "https://test.com",
			"token":     "test-token",
			"auth_type": "azure-cli",
		})
		return cfg, fmt.Errorf("auth error")
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
	ctx = root.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	showSensitive := true

	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.Flag("profile").Value.Set("my-profile")
	cmd.Flag("profile").Changed = true

	cfg := &config.Config{
		Profile: "my-profile",
	}
	m.WorkspaceClient.Config = cfg
	t.Setenv("DATABRICKS_AUTH_TYPE", "azure-cli")

	status, err := getWorkspaceAuthStatus(cmd, []string{}, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, error) {
		config.ConfigAttributes.ResolveFromStringMap(cfg, map[string]string{
			"host":      "https://test.com",
			"token":     "test-token",
			"auth_type": "azure-cli",
		})
		return cfg, fmt.Errorf("auth error")
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
	ctx = root.SetAccountClient(ctx, m.AccountClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	showSensitive := false

	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.Flag("profile").Value.Set("my-profile")
	cmd.Flag("profile").Changed = true

	cfg := &config.Config{
		Profile: "my-profile",
	}
	m.AccountClient.Config = cfg
	t.Setenv("DATABRICKS_AUTH_TYPE", "azure-cli")

	status, err := getAccountAuthStatus(cmd, []string{}, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, error) {
		config.ConfigAttributes.ResolveFromStringMap(cfg, map[string]string{
			"account_id": "test-account-id",
			"username":   "test-user",
			"host":       "https://test.com",
			"token":      "test-token",
			"auth_type":  "azure-cli",
		})
		return cfg, nil
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, "success", status.Status)

	require.Equal(t, "test-user", status.Details.Username)
	require.Equal(t, "https://test.com", status.Details.Host)
	require.Equal(t, "azure-cli", status.Details.AuthType)
	require.Equal(t, "test-account-id", status.Details.AccountID)

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
