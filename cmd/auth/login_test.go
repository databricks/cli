package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	// Test setting host from flag with trailing slash is stripped
	authArguments.Host = "https://www.host1.com/"
	err = setHostAndAccountId(ctx, profile1, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host1.com", authArguments.Host)

	// Test setting host from argument
	authArguments.Host = ""
	err = setHostAndAccountId(ctx, profile1, &authArguments, []string{"val from [HOST]"})
	assert.NoError(t, err)
	assert.Equal(t, "val from [HOST]", authArguments.Host)

	// Test setting host from argument with trailing slash is stripped
	authArguments.Host = ""
	err = setHostAndAccountId(ctx, profile1, &authArguments, []string{"https://www.host1.com/"})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host1.com", authArguments.Host)

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

func TestExtractHostQueryParams(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		existingAcctID  string
		existingWsID    string
		wantHost        string
		wantAccountID   string
		wantWorkspaceID string
	}{
		{
			name:            "extract workspace_id from ?o=",
			host:            "https://spog.example.com/?o=12345",
			wantHost:        "https://spog.example.com",
			wantWorkspaceID: "12345",
		},
		{
			name:            "extract both account_id and workspace_id",
			host:            "https://spog.example.com/?o=12345&a=abc",
			wantHost:        "https://spog.example.com",
			wantAccountID:   "abc",
			wantWorkspaceID: "12345",
		},
		{
			name:          "extract account_id from ?account_id=",
			host:          "https://spog.example.com/?account_id=abc",
			wantHost:      "https://spog.example.com",
			wantAccountID: "abc",
		},
		{
			name:            "extract workspace_id from ?workspace_id=",
			host:            "https://spog.example.com/?workspace_id=99999",
			wantHost:        "https://spog.example.com",
			wantWorkspaceID: "99999",
		},
		{
			name:     "no query params leaves host unchanged",
			host:     "https://spog.example.com",
			wantHost: "https://spog.example.com",
		},
		{
			name:            "explicit flags take precedence over query params",
			host:            "https://spog.example.com/?o=12345&a=abc",
			existingAcctID:  "explicit-account",
			existingWsID:    "explicit-ws",
			wantHost:        "https://spog.example.com",
			wantAccountID:   "explicit-account",
			wantWorkspaceID: "explicit-ws",
		},
		{
			name:     "invalid URL is left unchanged",
			host:     "not a valid url ://???",
			wantHost: "not a valid url ://???",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &auth.AuthArguments{
				Host:        tt.host,
				AccountID:   tt.existingAcctID,
				WorkspaceID: tt.existingWsID,
			}
			extractHostQueryParams(args)
			assert.Equal(t, tt.wantHost, args.Host)
			assert.Equal(t, tt.wantAccountID, args.AccountID)
			assert.Equal(t, tt.wantWorkspaceID, args.WorkspaceID)
		})
	}
}

func TestRunHostDiscovery_NoHost(t *testing.T) {
	ctx := t.Context()
	args := &auth.AuthArguments{}
	runHostDiscovery(ctx, args)
	assert.Equal(t, "", args.AccountID)
	assert.Equal(t, "", args.WorkspaceID)
}

func TestRunHostDiscovery_ExplicitFieldsNotOverridden(t *testing.T) {
	ctx := t.Context()
	args := &auth.AuthArguments{
		Host:        "https://nonexistent.example.com",
		AccountID:   "explicit-account",
		WorkspaceID: "explicit-ws",
	}
	runHostDiscovery(ctx, args)
	// Explicit fields should not be overridden even if discovery would return values
	assert.Equal(t, "explicit-account", args.AccountID)
	assert.Equal(t, "explicit-ws", args.WorkspaceID)
}

// newDiscoveryServer creates a test HTTP server that responds to
// .well-known/databricks-config with the given metadata.
func newDiscoveryServer(t *testing.T, metadata map[string]any) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(metadata); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestRunHostDiscovery_SPOGHost(t *testing.T) {
	server := newDiscoveryServer(t, map[string]any{
		"account_id":    "discovered-account",
		"workspace_id":  "discovered-ws",
		"oidc_endpoint": "https://spog.example.com/oidc/accounts/discovered-account",
	})

	ctx := t.Context()
	args := &auth.AuthArguments{Host: server.URL}
	runHostDiscovery(ctx, args)

	assert.Equal(t, "discovered-account", args.AccountID)
	assert.Equal(t, "discovered-ws", args.WorkspaceID)
}

func TestRunHostDiscovery_ClassicWorkspaceDoesNotSetAccountID(t *testing.T) {
	// Classic workspace discovery returns workspace-scoped OIDC (no account in path).
	server := newDiscoveryServer(t, map[string]any{
		"workspace_id":  "12345",
		"oidc_endpoint": "https://ws.example.com/oidc",
	})

	ctx := t.Context()
	args := &auth.AuthArguments{Host: server.URL}
	runHostDiscovery(ctx, args)

	// Only workspace_id is set; account_id stays empty since discovery didn't return it.
	assert.Equal(t, "", args.AccountID)
	assert.Equal(t, "12345", args.WorkspaceID)
}

func TestExtractHostQueryParams_OverridesProfileWorkspaceID(t *testing.T) {
	// Simulates the fix: profile loads workspace_id="old-ws", then the user
	// provides --host https://spog.example.com?o=new-ws. After Fix 1, profile
	// inheritance is deferred, so authArguments.WorkspaceID is empty when
	// extractHostQueryParams runs, and URL param wins.
	args := &auth.AuthArguments{
		Host: "https://spog.example.com/?o=new-ws",
		// WorkspaceID is empty because profile inheritance was deferred.
	}
	extractHostQueryParams(args)
	assert.Equal(t, "https://spog.example.com", args.Host)
	assert.Equal(t, "new-ws", args.WorkspaceID)
}

func TestSetHostAndAccountId_URLParamsOverrideProfile(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})

	unifiedWorkspaceProfile := loadTestProfile(t, ctx, "unified-workspace")

	// The profile has workspace_id=123456789, but the URL has ?o=new-ws.
	// After Fix 1, URL params should win over profile values.
	args := auth.AuthArguments{
		Host:          "https://unified.databricks.com?o=new-ws",
		AccountID:     "test-unified-account",
		IsUnifiedHost: true,
	}
	err := setHostAndAccountId(ctx, unifiedWorkspaceProfile, &args, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", args.Host)
	assert.Equal(t, "new-ws", args.WorkspaceID)
}
