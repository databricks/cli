package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestProfile(t *testing.T, ctx context.Context, profileName string) *profile.Profile {
	profile, err := loadProfileByName(ctx, profileName, profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, profile)
	return profile
}

func TestSetHostDoesNotFailWithNoDatabrickscfg(t *testing.T) {
	ctx := t.Context()
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./imaginary-file/databrickscfg")

	existingProfile, err := loadProfileByName(ctx, "foo", profile.DefaultProfiler)
	assert.NoError(t, err)

	err = setHostAndAccountId(ctx, existingProfile, &auth.AuthArguments{Host: "test"}, []string{})
	assert.NoError(t, err)
}

func TestSetHost(t *testing.T) {
	var authArguments auth.AuthArguments
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})

	profile1 := loadTestProfile(t, ctx, "profile-1")
	profile2 := loadTestProfile(t, ctx, "profile-2")

	// Test error when both flag and argument are provided
	authArguments.Host = "val from --host"
	err := setHostAndAccountId(ctx, profile1, &authArguments, []string{"val from [HOST]"})
	assert.EqualError(t, err, "please only provide a host as an argument or a flag, not both")

	// Test setting host from flag
	authArguments.Host = "val from --host"
	err = setHostAndAccountId(ctx, profile1, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "val from --host", authArguments.Host)

	// Test setting host from argument
	authArguments.Host = ""
	err = setHostAndAccountId(ctx, profile1, &authArguments, []string{"val from [HOST]"})
	assert.NoError(t, err)
	assert.Equal(t, "val from [HOST]", authArguments.Host)

	// Test setting host from profile
	authArguments.Host = ""
	err = setHostAndAccountId(ctx, profile1, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host1.com", authArguments.Host)

	// Test setting host from profile
	authArguments.Host = ""
	err = setHostAndAccountId(ctx, profile2, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host2.com", authArguments.Host)

	// Test host is not set. Should prompt.
	authArguments.Host = ""
	err = setHostAndAccountId(ctx, nil, &authArguments, []string{})
	assert.EqualError(t, err, "the command is being run in a non-interactive environment, please specify a host using --host")
}

func TestSetAccountId(t *testing.T) {
	var authArguments auth.AuthArguments
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})

	accountProfile := loadTestProfile(t, ctx, "account-profile")

	// Test setting account-id from flag
	authArguments.AccountID = "val from --account-id"
	err := setHostAndAccountId(ctx, accountProfile, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://accounts.cloud.databricks.com", authArguments.Host)
	assert.Equal(t, "val from --account-id", authArguments.AccountID)

	// Test setting account_id from profile
	authArguments.AccountID = ""
	err = setHostAndAccountId(ctx, accountProfile, &authArguments, []string{})
	require.NoError(t, err)
	assert.Equal(t, "https://accounts.cloud.databricks.com", authArguments.Host)
	assert.Equal(t, "id-from-profile", authArguments.AccountID)

	// Neither flag nor profile account-id is set, should prompt
	authArguments.AccountID = ""
	authArguments.Host = "https://accounts.cloud.databricks.com"
	err = setHostAndAccountId(ctx, nil, &authArguments, []string{})
	assert.EqualError(t, err, "the command is being run in a non-interactive environment, please specify an account ID using --account-id")
}

func TestSetWorkspaceIDForUnifiedHost(t *testing.T) {
	var authArguments auth.AuthArguments
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})

	unifiedWorkspaceProfile := loadTestProfile(t, ctx, "unified-workspace")
	unifiedAccountProfile := loadTestProfile(t, ctx, "unified-account")

	// Test setting workspace-id from flag for unified host
	authArguments = auth.AuthArguments{
		Host:          "https://unified.databricks.com",
		AccountID:     "test-unified-account",
		WorkspaceID:   "val from --workspace-id",
		IsUnifiedHost: true,
	}
	err := setHostAndAccountId(ctx, unifiedWorkspaceProfile, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", authArguments.Host)
	assert.Equal(t, "test-unified-account", authArguments.AccountID)
	assert.Equal(t, "val from --workspace-id", authArguments.WorkspaceID)

	// Test setting workspace_id from profile for unified host
	authArguments = auth.AuthArguments{
		Host:          "https://unified.databricks.com",
		AccountID:     "test-unified-account",
		IsUnifiedHost: true,
	}
	err = setHostAndAccountId(ctx, unifiedWorkspaceProfile, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", authArguments.Host)
	assert.Equal(t, "test-unified-account", authArguments.AccountID)
	assert.Equal(t, "123456789", authArguments.WorkspaceID)

	// Test workspace_id is optional - should default to empty in non-interactive mode
	authArguments = auth.AuthArguments{
		Host:          "https://unified.databricks.com",
		AccountID:     "test-unified-account",
		IsUnifiedHost: true,
	}
	err = setHostAndAccountId(ctx, unifiedAccountProfile, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", authArguments.Host)
	assert.Equal(t, "test-unified-account", authArguments.AccountID)
	assert.Equal(t, "", authArguments.WorkspaceID) // Empty is valid for account-level access

	// Test workspace_id is optional - should default to empty when no profile exists
	authArguments = auth.AuthArguments{
		Host:          "https://unified.databricks.com",
		AccountID:     "test-unified-account",
		IsUnifiedHost: true,
	}
	err = setHostAndAccountId(ctx, nil, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", authArguments.Host)
	assert.Equal(t, "test-unified-account", authArguments.AccountID)
	assert.Equal(t, "", authArguments.WorkspaceID) // Empty is valid for account-level access
}

func TestPromptForWorkspaceIDInNonInteractiveMode(t *testing.T) {
	// Setup non-interactive context
	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})

	// Test that promptForWorkspaceID returns empty string (no error) in non-interactive mode
	workspaceID, err := promptForWorkspaceID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "", workspaceID)
}

func TestLoadProfileByNameAndClusterID(t *testing.T) {
	testCases := []struct {
		name              string
		profile           string
		configFileEnv     string
		homeDirOverride   string
		expectedHost      string
		expectedClusterID string
	}{
		{
			name:              "cluster profile",
			profile:           "cluster-profile",
			configFileEnv:     "./testdata/.databrickscfg",
			expectedHost:      "https://www.host2.com",
			expectedClusterID: "cluster-from-config",
		},
		{
			name:              "profile from home directory (existing)",
			profile:           "cluster-profile",
			homeDirOverride:   "testdata",
			expectedHost:      "https://www.host2.com",
			expectedClusterID: "cluster-from-config",
		},
		{
			name:              "profile does not exist",
			profile:           "no-profile",
			configFileEnv:     "./testdata/.databrickscfg",
			expectedHost:      "",
			expectedClusterID: "",
		},
		{
			name:              "account profile",
			profile:           "account-profile",
			configFileEnv:     "./testdata/.databrickscfg",
			expectedHost:      "https://accounts.cloud.databricks.com",
			expectedClusterID: "",
		},
		{
			name:              "config doesn't exist",
			profile:           "any-profile",
			configFileEnv:     "./nonexistent/.databrickscfg",
			expectedHost:      "",
			expectedClusterID: "",
		},
		{
			name:              "profile from home directory (non-existent)",
			profile:           "any-profile",
			homeDirOverride:   "nonexistent",
			expectedHost:      "",
			expectedClusterID: "",
		},
		{
			name:              "invalid profile (missing host)",
			profile:           "invalid-profile",
			configFileEnv:     "./testdata/.databrickscfg",
			expectedHost:      "",
			expectedClusterID: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()

			if tc.configFileEnv != "" {
				t.Setenv("DATABRICKS_CONFIG_FILE", tc.configFileEnv)
			} else if tc.homeDirOverride != "" {
				// Use default ~/.databrickscfg
				ctx = env.WithUserHomeDir(ctx, tc.homeDirOverride)
			}

			profile, err := loadProfileByName(ctx, tc.profile, profile.DefaultProfiler)
			require.NoError(t, err)

			if tc.expectedHost == "" {
				assert.Nil(t, profile, "Test case '%s' failed: expected nil profile but got non-nil profile", tc.name)
			} else {
				assert.NotNil(t, profile, "Test case '%s' failed: expected profile but got nil", tc.name)
				assert.Equal(t, tc.expectedHost, profile.Host,
					"Test case '%s' failed: expected host '%s', but got '%s'", tc.name, tc.expectedHost, profile.Host)
				assert.Equal(t, tc.expectedClusterID, profile.ClusterID,
					"Test case '%s' failed: expected cluster ID '%s', but got '%s'", tc.name, tc.expectedClusterID, profile.ClusterID)
			}
		})
	}
}

func TestShouldUseDiscovery(t *testing.T) {
	tests := []struct {
		name            string
		hostFlag        string
		args            []string
		existingProfile *profile.Profile
		want            bool
	}{
		{
			name: "no host from any source",
			want: true,
		},
		{
			name:     "host from flag",
			hostFlag: "https://example.com",
			want:     false,
		},
		{
			name: "host from positional arg",
			args: []string{"https://example.com"},
			want: false,
		},
		{
			name:            "host from existing profile",
			existingProfile: &profile.Profile{Host: "https://example.com"},
			want:            false,
		},
		{
			name:            "existing profile without host",
			existingProfile: &profile.Profile{Name: "test"},
			want:            true,
		},
		{
			name:            "nil profile",
			existingProfile: nil,
			want:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldUseDiscovery(tt.hostFlag, tt.args, tt.existingProfile)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDiscoveryLogin_IntrospectionFailureStillSavesProfile(t *testing.T) {
	// Create a temporary config file for this test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	// Create a mock introspection server that returns an error
	introspectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer introspectServer.Close()

	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})

	// Test the discoveryLogin function's introspection error handling by
	// calling IntrospectToken directly (since discoveryLogin requires a
	// real PersistentAuth flow we can't easily mock).
	result, err := auth.IntrospectToken(ctx, introspectServer.URL, "test-token")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "403")
}

func TestDiscoveryLogin_IntrospectionSuccessExtractsMetadata(t *testing.T) {
	introspectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]any{
			"principal_context": map[string]any{
				"authentication_scope": map[string]any{
					"account_id":   "acc-12345",
					"workspace_id": 2548836972759138,
				},
			},
		})
		assert.NoError(t, err)
	}))
	defer introspectServer.Close()

	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})
	result, err := auth.IntrospectToken(ctx, introspectServer.URL, "test-token")
	require.NoError(t, err)
	assert.Equal(t, "acc-12345", result.AccountID)
	assert.Equal(t, fmt.Sprintf("%d", int64(2548836972759138)), result.WorkspaceID)
}
