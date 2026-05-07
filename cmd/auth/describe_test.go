package auth

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetWorkspaceAuthStatus(t *testing.T) {
	ctx := t.Context()
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
			"host":      "https://test.test",
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
	require.Equal(t, "https://test.test", status.Details.Host)
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
	ctx := t.Context()
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
			"host":      "https://test.test",
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
	ctx := t.Context()
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
			"host":      "https://test.test",
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
	ctx := t.Context()
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
			"host":       "https://test.test",
			"token":      "test-token",
			"auth_type":  "azure-cli",
		})
		return cfg, true, nil
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Equal(t, "success", status.Status)

	require.Equal(t, "test-user", status.Username)
	require.Equal(t, "https://test.test", status.Details.Host)
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

func TestResolveTokenStorageInfo(t *testing.T) {
	cases := []struct {
		name     string
		authType string
		envValue string
		want     *tokenStorageInfo
	}{
		{
			name:     "non-databricks-cli auth has no token storage",
			authType: "pat",
			want:     nil,
		},
		{
			name:     "databricks-cli with default plaintext",
			authType: authTypeDatabricksCLI,
			want: &tokenStorageInfo{
				Mode:     "plaintext",
				Location: plaintextLocation,
				Source:   "default",
			},
		},
		{
			name:     "databricks-cli with secure from env",
			authType: authTypeDatabricksCLI,
			envValue: "secure",
			want: &tokenStorageInfo{
				Mode:     "secure",
				Location: secureLocation,
				Source:   "DATABRICKS_AUTH_STORAGE environment variable",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(storage.EnvVar, tc.envValue)
			t.Setenv("DATABRICKS_CONFIG_FILE", t.TempDir()+"/.databrickscfg")

			got := resolveTokenStorageInfo(t.Context(), tc.authType)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetWorkspaceAuthStatus_U2M_PopulatesTokenStorage(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	currentUserApi := m.GetMockCurrentUserAPI()
	currentUserApi.EXPECT().Me(mock.Anything).Return(&iam.User{UserName: "u2m-user"}, nil)

	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")
	require.NoError(t, cmd.Flag("profile").Value.Set("u2m-profile"))
	cmd.Flag("profile").Changed = true

	cfg := &config.Config{Profile: "u2m-profile"}
	m.WorkspaceClient.Config = cfg
	t.Setenv(storage.EnvVar, "secure")
	t.Setenv("DATABRICKS_CONFIG_FILE", t.TempDir()+"/.databrickscfg")
	require.NoError(t, config.ConfigAttributes.Configure(cfg))

	status, err := getAuthStatus(cmd, []string{}, false, func(cmd *cobra.Command, args []string) (*config.Config, bool, error) {
		require.NoError(t, config.ConfigAttributes.ResolveFromStringMap(cfg, map[string]string{
			"host":      "https://test.test",
			"auth_type": authTypeDatabricksCLI,
		}))
		return cfg, false, nil
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	require.NotNil(t, status.TokenStorage)
	assert.Equal(t, "secure", status.TokenStorage.Mode)
	assert.Equal(t, secureLocation, status.TokenStorage.Location)
	assert.Equal(t, "DATABRICKS_AUTH_STORAGE environment variable", status.TokenStorage.Source)
}

func TestGetWorkspaceAuthStatus_NonU2M_OmitsTokenStorage(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	currentUserApi := m.GetMockCurrentUserAPI()
	currentUserApi.EXPECT().Me(mock.Anything).Return(&iam.User{UserName: "pat-user"}, nil)

	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")

	cfg := &config.Config{}
	m.WorkspaceClient.Config = cfg
	require.NoError(t, config.ConfigAttributes.Configure(cfg))

	status, err := getAuthStatus(cmd, []string{}, false, func(cmd *cobra.Command, args []string) (*config.Config, bool, error) {
		require.NoError(t, config.ConfigAttributes.ResolveFromStringMap(cfg, map[string]string{
			"host":      "https://test.test",
			"token":     "pat-token",
			"auth_type": "pat",
		}))
		return cfg, false, nil
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Nil(t, status.TokenStorage)
}
