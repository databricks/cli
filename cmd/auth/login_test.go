package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// logBuffer is a thread-safe bytes.Buffer for capturing log output in tests.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (lb *logBuffer) Write(p []byte) (int, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.buf.Write(p)
}

func (lb *logBuffer) String() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.buf.String()
}

func loadTestProfile(t *testing.T, ctx context.Context, profileName string) *profile.Profile {
	profile, err := loadProfileByName(ctx, profileName, profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, profile)
	return profile
}

type fakeDiscoveryPersistentAuth struct {
	token        *oauth2.Token
	challengeErr error
	tokenErr     error
}

func (f *fakeDiscoveryPersistentAuth) Challenge() error {
	return f.challengeErr
}

func (f *fakeDiscoveryPersistentAuth) Token() (*oauth2.Token, error) {
	if f.tokenErr != nil {
		return nil, f.tokenErr
	}
	return f.token, nil
}

func (f *fakeDiscoveryPersistentAuth) Close() error {
	return nil
}

type fakeDiscoveryClient struct {
	oauthArg          *u2m.BasicDiscoveryOAuthArgument
	oauthArgErr       error
	persistentAuth    discoveryPersistentAuth
	persistentAuthErr error
	introspection     *auth.IntrospectionResult
	introspectionErr  error
	// For assertions
	introspectHost  string
	introspectToken string
}

func (f *fakeDiscoveryClient) NewOAuthArgument(profileName string) (*u2m.BasicDiscoveryOAuthArgument, error) {
	if f.oauthArgErr != nil {
		return nil, f.oauthArgErr
	}
	return f.oauthArg, nil
}

func (f *fakeDiscoveryClient) NewPersistentAuth(ctx context.Context, opts ...u2m.PersistentAuthOption) (discoveryPersistentAuth, error) {
	if f.persistentAuthErr != nil {
		return nil, f.persistentAuthErr
	}
	return f.persistentAuth, nil
}

func (f *fakeDiscoveryClient) IntrospectToken(ctx context.Context, host, accessToken string) (*auth.IntrospectionResult, error) {
	f.introspectHost = host
	f.introspectToken = accessToken
	if f.introspectionErr != nil {
		return nil, f.introspectionErr
	}
	return f.introspection, nil
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

func TestSplitScopes(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output []string
	}{
		{
			name:   "empty input",
			input:  "",
			output: nil,
		},
		{
			name:   "single scope",
			input:  "all-apis",
			output: []string{"all-apis"},
		},
		{
			name:   "trims whitespace",
			input:  " all-apis , sql ",
			output: []string{"all-apis", "sql"},
		},
		{
			name:   "drops empty entries",
			input:  "all-apis, ,sql,,",
			output: []string{"all-apis", "sql"},
		},
		{
			name:   "only empty entries",
			input:  " , , ",
			output: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.output, splitScopes(tt.input))
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

func TestSetHostAndAccountId_WorkspaceIDNoneSentinelInherited(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})

	skipProfile := loadTestProfile(t, ctx, "spog-skip-workspace")

	// When loading from a profile with workspace_id=none, the sentinel should
	// be inherited and the workspace prompt should not fire.
	args := auth.AuthArguments{
		Host:      "https://spog.example.com",
		AccountID: "spog-account",
	}
	err := setHostAndAccountId(ctx, skipProfile, &args, []string{})
	assert.NoError(t, err)
	assert.Equal(t, auth.WorkspaceIDNone, args.WorkspaceID)
}

func TestSetHostAndAccountId_URLParamsOverrideProfile(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})

	unifiedWorkspaceProfile := loadTestProfile(t, ctx, "unified-workspace")

	// The profile has workspace_id=123456789, but the URL has ?o=99999.
	// URL params should win over profile values.
	args := auth.AuthArguments{
		Host:          "https://unified.databricks.com?o=99999",
		AccountID:     "test-unified-account",
		IsUnifiedHost: true,
	}
	err := setHostAndAccountId(ctx, unifiedWorkspaceProfile, &args, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", args.Host)
	assert.Equal(t, "99999", args.WorkspaceID)
}

func TestValidateDiscoveryFlagCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		setFlag string
		flagVal string
		wantErr string
	}{
		{
			name:    "account-id is incompatible",
			setFlag: "account-id",
			flagVal: "abc123",
			wantErr: "--account-id requires --host to be specified",
		},
		{
			name:    "workspace-id is incompatible",
			setFlag: "workspace-id",
			flagVal: "12345",
			wantErr: "--workspace-id requires --host to be specified",
		},
		{
			name:    "experimental-is-unified-host is incompatible",
			setFlag: "experimental-is-unified-host",
			flagVal: "true",
			wantErr: "--experimental-is-unified-host requires --host to be specified",
		},
		{
			name:    "configure-cluster is incompatible",
			setFlag: "configure-cluster",
			flagVal: "true",
			wantErr: "--configure-cluster requires --host to be specified",
		},
		{
			name:    "configure-serverless is incompatible",
			setFlag: "configure-serverless",
			flagVal: "true",
			wantErr: "--configure-serverless requires --host to be specified",
		},
		{
			name: "no flags set is ok",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("account-id", "", "")
			cmd.Flags().String("workspace-id", "", "")
			cmd.Flags().Bool("experimental-is-unified-host", false, "")
			cmd.Flags().Bool("configure-cluster", false, "")
			cmd.Flags().Bool("configure-serverless", false, "")

			if tt.setFlag != "" {
				require.NoError(t, cmd.Flags().Set(tt.setFlag, tt.flagVal))
			}

			err := validateDiscoveryFlagCompatibility(cmd)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDiscoveryLogin_IntrospectionFailureStillSavesProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://workspace.example.com")

	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
		introspectionErr: errors.New("introspection failed"),
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, dc, "DISCOVERY", time.Second, "all-apis, ,sql,", nil, func(string) error { return nil })
	require.NoError(t, err)

	assert.Equal(t, "https://workspace.example.com", dc.introspectHost)
	assert.Equal(t, "test-token", dc.introspectToken)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://workspace.example.com", savedProfile.Host)
	assert.Equal(t, "all-apis,sql", savedProfile.Scopes)
	assert.Empty(t, savedProfile.AccountID)
	assert.Empty(t, savedProfile.WorkspaceID)
}

func TestDiscoveryLogin_AccountIDMismatchWarning(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://workspace.example.com")

	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
		introspection: &auth.IntrospectionResult{
			AccountID:   "new-account-id",
			WorkspaceID: "12345",
		},
	}

	// Set up a logger that captures log records to verify the warning.
	var logBuf logBuffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	ctx = log.NewContext(ctx, logger)

	existingProfile := &profile.Profile{
		Name:      "DISCOVERY",
		AccountID: "old-account-id",
	}

	err = discoveryLogin(ctx, dc, "DISCOVERY", time.Second, "", existingProfile, func(string) error { return nil })
	require.NoError(t, err)

	// Verify warning about mismatched account IDs was logged.
	assert.Contains(t, logBuf.String(), "new-account-id")
	assert.Contains(t, logBuf.String(), "old-account-id")

	// Account ID from introspection is now saved to the profile.
	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://workspace.example.com", savedProfile.Host)
	assert.Equal(t, "new-account-id", savedProfile.AccountID)
	assert.Equal(t, "12345", savedProfile.WorkspaceID)
}

func TestDiscoveryLogin_NoWarningWhenAccountIDsMatch(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://workspace.example.com")

	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
		introspection: &auth.IntrospectionResult{
			AccountID:   "same-account-id",
			WorkspaceID: "12345",
		},
	}

	var logBuf logBuffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	ctx = log.NewContext(ctx, logger)

	existingProfile := &profile.Profile{
		Name:      "DISCOVERY",
		AccountID: "same-account-id",
	}

	err = discoveryLogin(ctx, dc, "DISCOVERY", time.Second, "", existingProfile, func(string) error { return nil })
	require.NoError(t, err)

	// No warning should be logged when account IDs match.
	assert.Empty(t, logBuf.String())
}

func TestDiscoveryLogin_EmptyDiscoveredHostReturnsError(t *testing.T) {
	// Return arg without calling SetDiscoveredHost, so GetDiscoveredHost returns "".
	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)

	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, dc, "DISCOVERY", time.Second, "", nil, func(string) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no workspace host was discovered")
}

func TestDiscoveryLogin_ReloginPreservesExistingProfileScopes(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://workspace.example.com")

	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
		introspectionErr: errors.New("introspection failed"),
	}

	existingProfile := &profile.Profile{
		Name:   "DISCOVERY",
		Host:   "https://old-workspace.example.com",
		Scopes: "sql,clusters",
	}

	// No --scopes flag (empty string), should fall back to existing profile scopes.
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, dc, "DISCOVERY", time.Second, "", existingProfile, func(string) error { return nil })
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://workspace.example.com", savedProfile.Host)
	assert.Equal(t, "sql,clusters", savedProfile.Scopes)
}

func TestDiscoveryLogin_ExplicitScopesOverrideExistingProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://workspace.example.com")

	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
		introspectionErr: errors.New("introspection failed"),
	}

	existingProfile := &profile.Profile{
		Name:   "DISCOVERY",
		Host:   "https://old-workspace.example.com",
		Scopes: "sql,clusters",
	}

	// Explicit --scopes flag should override existing profile scopes.
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, dc, "DISCOVERY", time.Second, "all-apis", existingProfile, func(string) error { return nil })
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "all-apis", savedProfile.Scopes)
}

func TestDiscoveryLogin_SPOGHostPopulatesAccountIDFromDiscovery(t *testing.T) {
	// Start a mock server that returns SPOG discovery metadata.
	server := newDiscoveryServer(t, map[string]any{
		"account_id":    "discovered-account",
		"workspace_id":  "discovered-ws",
		"oidc_endpoint": "https://spog.example.com/oidc/accounts/discovered-account",
	})

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost(server.URL)

	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
		// Introspection returns different values to verify discovery takes precedence.
		introspection: &auth.IntrospectionResult{
			AccountID:   "introspection-account",
			WorkspaceID: "introspection-ws",
		},
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, dc, "DISCOVERY", time.Second, "", nil, func(string) error { return nil })
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, server.URL, savedProfile.Host)
	assert.Equal(t, "discovered-account", savedProfile.AccountID, "account_id should come from host discovery")
	assert.Equal(t, "discovered-ws", savedProfile.WorkspaceID, "workspace_id should come from host discovery")
}

func TestDiscoveryLogin_IntrospectionFallsBackWhenDiscoveryFails(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	// Use a host that won't respond to .well-known/databricks-config.
	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://workspace.example.com")

	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
		introspection: &auth.IntrospectionResult{
			AccountID:   "introspection-account",
			WorkspaceID: "introspection-ws",
		},
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, dc, "DISCOVERY", time.Second, "", nil, func(string) error { return nil })
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://workspace.example.com", savedProfile.Host)
	assert.Equal(t, "introspection-account", savedProfile.AccountID, "account_id should fall back to introspection")
	assert.Equal(t, "introspection-ws", savedProfile.WorkspaceID, "workspace_id should fall back to introspection")
}

func TestDiscoveryLogin_ClearsStaleRoutingFieldsFromUnifiedProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")

	// Pre-populate a profile that looks like an older hostless/unified login.
	initialConfig := `[DISCOVERY]
host = https://old-unified.databricks.com
account_id = old-account
workspace_id = 999999
experimental_is_unified_host = true
auth_type = databricks-cli
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://new-workspace.example.com")

	// Introspection fails, so workspace_id should be cleared (not left stale).
	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
		introspectionErr: errors.New("introspection unavailable"),
	}

	existingProfile := &profile.Profile{
		Name:          "DISCOVERY",
		Host:          "https://old-unified.databricks.com",
		AccountID:     "old-account",
		WorkspaceID:   "999999",
		IsUnifiedHost: true,
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, dc, "DISCOVERY", time.Second, "", existingProfile, func(string) error { return nil })
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://new-workspace.example.com", savedProfile.Host)
	// Stale routing fields must be cleared.
	assert.Empty(t, savedProfile.AccountID, "stale account_id should be cleared")
	assert.Empty(t, savedProfile.WorkspaceID, "stale workspace_id should be cleared on introspection failure")
	assert.False(t, savedProfile.IsUnifiedHost, "stale experimental_is_unified_host should be cleared")
}

func TestDiscoveryLogin_IntrospectionWritesFreshWorkspaceID(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")

	// Pre-populate with stale workspace_id.
	initialConfig := `[DISCOVERY]
host = https://old.example.com
workspace_id = 111111
auth_type = databricks-cli
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0o600)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://new-workspace.example.com")

	// Introspection succeeds with a fresh workspace_id.
	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
		introspection: &auth.IntrospectionResult{
			AccountID:   "fresh-account",
			WorkspaceID: "222222",
		},
	}

	existingProfile := &profile.Profile{
		Name:        "DISCOVERY",
		Host:        "https://old.example.com",
		WorkspaceID: "111111",
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, dc, "DISCOVERY", time.Second, "", existingProfile, func(string) error { return nil })
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://new-workspace.example.com", savedProfile.Host)
	assert.Equal(t, "fresh-account", savedProfile.AccountID, "account_id should be saved from introspection")
	assert.Equal(t, "222222", savedProfile.WorkspaceID, "workspace_id should be updated to fresh introspection value")
}

func TestLoginRejectsPositionalArgWithHostFlag(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	authArgs := &auth.AuthArguments{Host: "https://example.com"}
	cmd := newLoginCommand(authArgs)
	cmd.Flags().String("profile", "", "")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"myprofile"})
	err := cmd.Execute()
	assert.ErrorContains(t, err, `argument "myprofile" cannot be combined with --host or --profile`)
}

func TestLoginRejectsPositionalArgWithProfileFlag(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	authArgs := &auth.AuthArguments{}
	cmd := newLoginCommand(authArgs)
	cmd.Flags().String("profile", "", "")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--profile", "myprofile", "https://example.com"})
	err := cmd.Execute()
	assert.ErrorContains(t, err, `argument "https://example.com" cannot be combined with --host or --profile`)
}
