package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newHostProfileCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("host", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.SetContext(t.Context())
	return cmd
}

func TestResolveProfileFromHostFlag(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte(""), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", cfgPath)

	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "dev", Host: "https://dev.cloud.databricks.com", AuthType: "databricks-cli"},
		},
	}

	t.Run("no flags set is a no-op", func(t *testing.T) {
		cmd := newHostProfileCmd(t)
		require.NoError(t, resolveProfileFromHostFlag(cmd, profiler))
		assert.Empty(t, cmd.Flag("profile").Value.String())
	})

	t.Run("--profile already set wins; --host is ignored", func(t *testing.T) {
		cmd := newHostProfileCmd(t)
		require.NoError(t, cmd.Flags().Set("host", "https://dev.cloud.databricks.com"))
		require.NoError(t, cmd.Flags().Set("profile", "explicit"))
		require.NoError(t, resolveProfileFromHostFlag(cmd, profiler))
		assert.Equal(t, "explicit", cmd.Flag("profile").Value.String())
	})

	t.Run("--host with a single match wires --profile", func(t *testing.T) {
		cmd := newHostProfileCmd(t)
		require.NoError(t, cmd.Flags().Set("host", "https://dev.cloud.databricks.com"))
		require.NoError(t, resolveProfileFromHostFlag(cmd, profiler))
		assert.Equal(t, "dev", cmd.Flag("profile").Value.String())
	})

	t.Run("--host with no match surfaces a clear error", func(t *testing.T) {
		cmd := newHostProfileCmd(t)
		require.NoError(t, cmd.Flags().Set("host", "https://nope.cloud.databricks.com"))
		err := resolveProfileFromHostFlag(cmd, profiler)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no profile found matching host")
		assert.Empty(t, cmd.Flag("profile").Value.String())
	})

	t.Run("DATABRICKS_CONFIG_PROFILE is left alone", func(t *testing.T) {
		t.Setenv("DATABRICKS_CONFIG_PROFILE", "from-env")
		cmd := newHostProfileCmd(t)
		require.NoError(t, cmd.Flags().Set("host", "https://dev.cloud.databricks.com"))
		require.NoError(t, resolveProfileFromHostFlag(cmd, profiler))
		// We don't overwrite --profile when the user signalled an explicit
		// choice via the env var.
		assert.Empty(t, cmd.Flag("profile").Value.String())
	})
}

func TestGetWorkspaceAuthStatus(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	showSensitive := false

	currentUserApi := m.GetMockCurrentUserAPI()
	currentUserApi.EXPECT().Me(mock.Anything, mock.Anything).Return(&iam.User{
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
			name:     "databricks-cli with default secure",
			authType: authTypeDatabricksCLI,
			want: &tokenStorageInfo{
				Mode:     "secure",
				Location: secureLocation,
				Source:   "default",
			},
		},
		{
			name:     "databricks-cli with plaintext from env",
			authType: authTypeDatabricksCLI,
			envValue: "plaintext",
			want: &tokenStorageInfo{
				Mode:     "plaintext",
				Location: plaintextLocation,
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

func TestStorageSourceLabel_ConfigUsesResolvedPath(t *testing.T) {
	ctx := t.Context()
	t.Setenv("DATABRICKS_CONFIG_FILE", "/custom/path/.databrickscfg")
	got := storageSourceLabel(ctx, storage.StorageSourceConfig)
	assert.Equal(t, "auth_storage in [__settings__] section of /custom/path/.databrickscfg", got)
}

func TestStorageSourceLabel_ConfigDefaultsToHome(t *testing.T) {
	ctx := t.Context()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("DATABRICKS_CONFIG_FILE", "")
	got := storageSourceLabel(ctx, storage.StorageSourceConfig)
	expected := "auth_storage in [__settings__] section of " + filepath.ToSlash(filepath.Join(home, ".databrickscfg"))
	assert.Equal(t, expected, got)
}

func TestStorageSourceLabel_NonConfigDelegatesToSource(t *testing.T) {
	ctx := t.Context()
	t.Setenv("DATABRICKS_CONFIG_FILE", "/should/not/appear")
	assert.Equal(t, "default", storageSourceLabel(ctx, storage.StorageSourceDefault))
	assert.Equal(t, "DATABRICKS_AUTH_STORAGE environment variable", storageSourceLabel(ctx, storage.StorageSourceEnvVar))
	assert.Equal(t, "command-line override", storageSourceLabel(ctx, storage.StorageSourceOverride))
}

func TestGetWorkspaceAuthStatus_U2M_PopulatesTokenStorage(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	currentUserApi := m.GetMockCurrentUserAPI()
	currentUserApi.EXPECT().Me(mock.Anything, mock.Anything).Return(&iam.User{UserName: "u2m-user"}, nil)

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

// describeVerifyServer simulates the host that describe's secondary
// verification calls hit: CurrentUser.Me and account Workspaces.List.
// Statuses other than 200 return an error body with that status code; a zero
// status marks an endpoint that must not be called at all.
func describeVerifyServer(t *testing.T, accountID string, meStatus, listStatus int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respond := func(status int, okBody any) {
			if status == 0 {
				t.Errorf("unexpected request to %s", r.URL.Path)
				status = http.StatusNotFound
			}
			w.Header().Set("Content-Type", "application/json")
			if status != http.StatusOK {
				w.WriteHeader(status)
				okBody = map[string]any{"error_code": "TEST_ERROR", "message": "secondary check failed"}
			}
			require.NoError(t, json.NewEncoder(w).Encode(okBody))
		}
		switch r.URL.Path {
		case "/api/2.0/preview/scim/v2/Me":
			respond(meStatus, map[string]any{"userName": "fallback-user"})
		case "/api/2.0/accounts/" + accountID + "/workspaces":
			respond(listStatus, []map[string]any{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// newDescribeWorkspaceCmd wires a mocked workspace client whose Me call fails
// with meErr, with cfg as its resolved configuration.
func newDescribeWorkspaceCmd(t *testing.T, cfg *config.Config, meErr error) *cobra.Command {
	t.Helper()
	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = cfg
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything, mock.Anything).Return(nil, meErr)
	cmd := newHostProfileCmd(t)
	cmd.SetContext(cmdctx.SetWorkspaceClient(t.Context(), m.WorkspaceClient))
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), ".databrickscfg"))
	return cmd
}

// newDescribeAccountCmd wires a mocked account client whose Workspaces.List
// call fails with listErr, with cfg as its resolved configuration.
func newDescribeAccountCmd(t *testing.T, cfg *config.Config, listErr error) *cobra.Command {
	t.Helper()
	m := mocks.NewMockAccountClient(t)
	m.AccountClient.Config = cfg
	m.GetMockWorkspacesAPI().EXPECT().List(mock.Anything).Return(nil, listErr)
	cmd := newHostProfileCmd(t)
	cmd.SetContext(cmdctx.SetAccountClient(t.Context(), m.AccountClient))
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), ".databrickscfg"))
	return cmd
}

// resolveCfg returns a tryAuth that resolves cfg from attrs and reports the
// given client type, mirroring what root.MustAnyClient leaves behind.
func resolveCfg(t *testing.T, cfg *config.Config, attrs map[string]string, isAccount bool) tryAuth {
	t.Helper()
	return func(cmd *cobra.Command, args []string) (*config.Config, bool, error) {
		require.NoError(t, config.ConfigAttributes.ResolveFromStringMap(cfg, attrs))
		return cfg, isAccount, nil
	}
}

// TestGetAuthStatusVerificationFallback covers the fallback when the primary
// verification call (mocked client) fails: describe tries the other endpoint
// over HTTP against describeVerifyServer, and only when both fail does it
// report the primary error. Error rows leave wantUsername/wantAccountID empty
// because errorAuthStatus never sets them.
func TestGetAuthStatusVerificationFallback(t *testing.T) {
	tests := []struct {
		name          string
		isAccount     bool
		primaryErr    *apierr.APIError
		meStatus      int
		listStatus    int
		accountID     string
		wantStatus    string
		wantUsername  string
		wantAccountID string
	}{
		{
			name:          "workspace check fails, account check succeeds",
			primaryErr:    &apierr.APIError{StatusCode: http.StatusBadRequest, Message: "Unable to load OAuth Config"},
			meStatus:      http.StatusNotFound,
			listStatus:    http.StatusOK,
			accountID:     "test-acct",
			wantStatus:    "success",
			wantAccountID: "test-acct",
		},
		{
			name:       "no second call without an account id",
			primaryErr: &apierr.APIError{StatusCode: http.StatusBadRequest, Message: "Unable to load OAuth Config"},
			wantStatus: "error",
		},
		{
			name:         "account check fails, workspace check succeeds",
			isAccount:    true,
			primaryErr:   &apierr.APIError{StatusCode: http.StatusForbidden, Message: "This API is disabled for users without account admin status"},
			meStatus:     http.StatusOK,
			listStatus:   http.StatusNotFound,
			accountID:    "test-acct",
			wantStatus:   "success",
			wantUsername: "fallback-user",
		},
		{
			name:       "workspace branch, both checks fail",
			primaryErr: &apierr.APIError{StatusCode: http.StatusBadRequest, Message: "Unable to load OAuth Config"},
			meStatus:   http.StatusNotFound,
			listStatus: http.StatusUnauthorized,
			accountID:  "test-acct",
			wantStatus: "error",
		},
		{
			name:       "account branch, both checks fail",
			isAccount:  true,
			primaryErr: &apierr.APIError{StatusCode: http.StatusUnauthorized, Message: "credentials expired"},
			meStatus:   http.StatusUnauthorized,
			listStatus: http.StatusNotFound,
			accountID:  "test-acct",
			wantStatus: "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABRICKS_ACCOUNT_ID", "")
			server := describeVerifyServer(t, tc.accountID, tc.meStatus, tc.listStatus)
			cfg := &config.Config{}
			var cmd *cobra.Command
			if tc.isAccount {
				cmd = newDescribeAccountCmd(t, cfg, tc.primaryErr)
			} else {
				cmd = newDescribeWorkspaceCmd(t, cfg, tc.primaryErr)
			}
			attrs := map[string]string{
				"host":      server.URL,
				"token":     "test-token",
				"auth_type": "pat",
			}
			if tc.accountID != "" {
				attrs["account_id"] = tc.accountID
			}

			status, err := getAuthStatus(cmd, []string{}, false, resolveCfg(t, cfg, attrs, tc.isAccount))
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, status.Status)
			assert.Equal(t, tc.wantUsername, status.Username)
			assert.Equal(t, tc.wantAccountID, status.AccountID)
			if tc.wantStatus == "error" {
				assert.ErrorIs(t, status.Error, tc.primaryErr)
			}
		})
	}
}

func TestGetWorkspaceAuthStatus_NonU2M_OmitsTokenStorage(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	currentUserApi := m.GetMockCurrentUserAPI()
	currentUserApi.EXPECT().Me(mock.Anything, mock.Anything).Return(&iam.User{UserName: "pat-user"}, nil)

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
