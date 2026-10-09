package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// newTestStore returns an in-memory token cache for tests so that
// discoveryLogin and other login helpers don't touch ~/.databricks/token-cache.json.
func newTestStore() storage.Store {
	return &inMemoryStore{Tokens: map[string]*oauth2.Token{}}
}

type putErrorStore struct {
	storage.Store
	err error
}

func (s *putErrorStore) Put(string, storage.Entry) error {
	return s.err
}

type countingStore struct {
	storage.Store
	putCalls int
}

func (s *countingStore) Put(key string, entry storage.Entry) error {
	s.putCalls++
	return s.Store.Put(key, entry)
}

func TestStoreLoginTokenDeletesStaleTokenOnFailure(t *testing.T) {
	const profileName = "TEST"
	inner := storage.NewMemoryStore()
	require.NoError(t, inner.Put(profileName, storage.Entry{
		Token: &oauth2.Token{AccessToken: "old-token"},
	}))
	storeErr := errors.New("put failed")
	store := &putErrorStore{Store: inner, err: storeErr}
	arg, err := u2m.NewProfileWorkspaceOAuthArgument("https://workspace.example.test", profileName)
	require.NoError(t, err)

	err = storeLoginToken(t.Context(), store, storage.StorageModeSecure, arg, &oauth2.Token{AccessToken: "new-token"})

	assert.ErrorIs(t, err, storeErr)
	_, err = inner.Lookup(profileName)
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

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
}

func (f *fakeDiscoveryPersistentAuth) Challenge() (*oauth2.Token, error) {
	return f.token, f.challengeErr
}

func (f *fakeDiscoveryPersistentAuth) Close() error {
	return nil
}

type fakeDiscoveryClient struct {
	oauthArg          *u2m.BasicDiscoveryOAuthArgument
	oauthArgErr       error
	persistentAuth    discoveryPersistentAuth
	persistentAuthErr error
	// followUpAuths, when set, serve NewPersistentAuth calls after the first
	// (e.g. the second login at a workspace's primary URL), in order.
	followUpAuths []discoveryPersistentAuth
	// newFollowUpAuth, when set, serves NewPersistentAuth calls after the
	// first and receives their options.
	newFollowUpAuth  func(ctx context.Context, opts ...u2m.PersistentAuthOption) (discoveryPersistentAuth, error)
	introspection    *auth.IntrospectionResult
	introspectionErr error
	// For assertions
	introspectHost        string
	introspectToken       string
	newPersistentAuthCall int
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
	f.newPersistentAuthCall++
	if f.newPersistentAuthCall > 1 && f.newFollowUpAuth != nil {
		return f.newFollowUpAuth(ctx, opts...)
	}
	if f.newPersistentAuthCall > 1 && len(f.followUpAuths) > 0 {
		next := f.followUpAuths[0]
		f.followUpAuths = f.followUpAuths[1:]
		return next, nil
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
	authArguments.Host = "https://www.host1.test/"
	err = setHostAndAccountId(ctx, profile1, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host1.test", authArguments.Host)

	// Test setting host from argument
	authArguments.Host = ""
	err = setHostAndAccountId(ctx, profile1, &authArguments, []string{"val from [HOST]"})
	assert.NoError(t, err)
	assert.Equal(t, "val from [HOST]", authArguments.Host)

	// Test setting host from argument with trailing slash is stripped
	authArguments.Host = ""
	err = setHostAndAccountId(ctx, profile1, &authArguments, []string{"https://www.host1.test/"})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host1.test", authArguments.Host)

	// Test setting host from profile
	authArguments.Host = ""
	err = setHostAndAccountId(ctx, profile1, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host1.test", authArguments.Host)

	// Test setting host from profile
	authArguments.Host = ""
	err = setHostAndAccountId(ctx, profile2, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.host2.test", authArguments.Host)

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
		Host:        "https://unified.databricks.com",
		AccountID:   "test-unified-account",
		WorkspaceID: "val from --workspace-id",
	}
	err := setHostAndAccountId(ctx, unifiedWorkspaceProfile, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", authArguments.Host)
	assert.Equal(t, "test-unified-account", authArguments.AccountID)
	assert.Equal(t, "val from --workspace-id", authArguments.WorkspaceID)

	// Test setting workspace_id from profile for unified host
	authArguments = auth.AuthArguments{
		Host:      "https://unified.databricks.com",
		AccountID: "test-unified-account",
	}
	err = setHostAndAccountId(ctx, unifiedWorkspaceProfile, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", authArguments.Host)
	assert.Equal(t, "test-unified-account", authArguments.AccountID)
	assert.Equal(t, "123456789", authArguments.WorkspaceID)

	// Test workspace_id is optional - should default to empty in non-interactive mode
	authArguments = auth.AuthArguments{
		Host:      "https://unified.databricks.com",
		AccountID: "test-unified-account",
	}
	err = setHostAndAccountId(ctx, unifiedAccountProfile, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", authArguments.Host)
	assert.Equal(t, "test-unified-account", authArguments.AccountID)
	assert.Empty(t, authArguments.WorkspaceID) // Empty is valid for account-level access

	// Test workspace_id is optional - should default to empty when no profile exists
	authArguments = auth.AuthArguments{
		Host:      "https://unified.databricks.com",
		AccountID: "test-unified-account",
	}
	err = setHostAndAccountId(ctx, nil, &authArguments, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", authArguments.Host)
	assert.Equal(t, "test-unified-account", authArguments.AccountID)
	assert.Empty(t, authArguments.WorkspaceID) // Empty is valid for account-level access
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
			expectedHost:      "https://www.host2.test",
			expectedClusterID: "cluster-from-config",
		},
		{
			name:              "profile from home directory (existing)",
			profile:           "cluster-profile",
			homeDirOverride:   "testdata",
			expectedHost:      "https://www.host2.test",
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

func TestNeedsAccountIDPrompt(t *testing.T) {
	cases := []struct {
		name         string
		host         string
		discoveryURL string
		want         bool
	}{
		{name: "classic accounts host", host: "https://accounts.cloud.databricks.com", want: true},
		{name: "accounts-dod host", host: "https://accounts-dod.databricks.com", want: true},
		{name: "accounts host with path", host: "https://accounts.cloud.databricks.com/some/path", want: true},
		{name: "plain workspace host", host: "https://workspace.cloud.databricks.com"},
		{name: "account-scoped DiscoveryURL", host: "https://spog.cloud.databricks.com", discoveryURL: "https://spog.cloud.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server", want: true},
		{name: "workspace-scoped DiscoveryURL", host: "https://workspace.cloud.databricks.com", discoveryURL: "https://workspace.cloud.databricks.com/oidc/.well-known/oauth-authorization-server"},
		{name: "workspace host no signals", host: "https://workspace.cloud.databricks.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := needsAccountIDPrompt(tc.host, tc.discoveryURL)
			assert.Equal(t, tc.want, got)
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

func TestU2MClientIDFromProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile *profile.Profile
		want    string
	}{
		{name: "no profile"},
		{
			name:    "implicit auth type",
			profile: &profile.Profile{ClientID: "custom-client-id"},
		},
		{
			name:    "M2M auth type",
			profile: &profile.Profile{AuthType: "oauth-m2m", ClientID: "custom-client-id"},
		},
		{
			name:    "U2M auth type",
			profile: &profile.Profile{AuthType: authTypeDatabricksCLI, ClientID: "custom-client-id"},
			want:    "custom-client-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, u2mClientIDFromProfile(tt.profile))
		})
	}
}

func TestU2MResourcesFromProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile *profile.Profile
		want    []string
	}{
		{name: "no profile"},
		{
			name:    "implicit auth type",
			profile: &profile.Profile{Resources: "https://workspace.test/ai-gateway/mcp/system.ai.github"},
		},
		{
			name:    "M2M auth type",
			profile: &profile.Profile{AuthType: "oauth-m2m", Resources: "https://workspace.test/ai-gateway/mcp/system.ai.github"},
		},
		{
			name:    "U2M auth type single resource",
			profile: &profile.Profile{AuthType: authTypeDatabricksCLI, Resources: "https://workspace.test/ai-gateway/mcp/system.ai.github"},
			want:    []string{"https://workspace.test/ai-gateway/mcp/system.ai.github"},
		},
		{
			name:    "U2M auth type multiple resources",
			profile: &profile.Profile{AuthType: authTypeDatabricksCLI, Resources: "https://workspace.test/ai-gateway/mcp/system.ai.github, https://workspace.test/ai-gateway/mcp/system.ai.slack"},
			want:    []string{"https://workspace.test/ai-gateway/mcp/system.ai.github", "https://workspace.test/ai-gateway/mcp/system.ai.slack"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, u2mResourcesFromProfile(tt.profile))
		})
	}
}

func TestRunHostDiscovery_NoHost(t *testing.T) {
	ctx := t.Context()
	args := &auth.AuthArguments{}
	runHostDiscovery(ctx, args)
	assert.Empty(t, args.AccountID)
	assert.Empty(t, args.WorkspaceID)
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
	assert.Empty(t, args.AccountID)
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

func TestShouldPromptWorkspace(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})

	legacyAccountProfile := loadTestProfile(t, ctx, "spog-skip-workspace")
	newAccountProfile := loadTestProfile(t, ctx, "spog-skip-workspace-new")
	workspaceProfile := loadTestProfile(t, ctx, "unified-workspace")

	tests := []struct {
		name            string
		authArguments   auth.AuthArguments
		existingProfile *profile.Profile
		skipWorkspace   bool
		want            bool
	}{
		{
			name:          "no profile, account_id set, no workspace_id",
			authArguments: auth.AuthArguments{AccountID: "acc"},
			want:          true,
		},
		{
			name:          "classic account console host never prompts",
			authArguments: auth.AuthArguments{Host: "https://accounts.test", AccountID: "acc"},
			want:          false,
		},
		{
			name:          "classic account console host without scheme never prompts",
			authArguments: auth.AuthArguments{Host: "accounts.test", AccountID: "acc"},
			want:          false,
		},
		{
			name:          "accounts-dod host never prompts",
			authArguments: auth.AuthArguments{Host: "https://accounts-dod.test", AccountID: "acc"},
			want:          false,
		},
		{
			name:          "workspace host with account_id prompts",
			authArguments: auth.AuthArguments{Host: "https://myworkspace.test", AccountID: "acc"},
			want:          true,
		},
		{
			name:          "classic account console host with workspace_id set never prompts",
			authArguments: auth.AuthArguments{Host: "https://accounts.test", AccountID: "acc", WorkspaceID: "12345"},
			want:          false,
		},
		{
			name:            "re-login into legacy account-only profile (workspace_id = none)",
			authArguments:   auth.AuthArguments{AccountID: "spog-account"},
			existingProfile: legacyAccountProfile,
			want:            false,
		},
		{
			name:            "re-login into new account-only profile (no workspace_id key)",
			authArguments:   auth.AuthArguments{AccountID: "spog-account"},
			existingProfile: newAccountProfile,
			want:            false,
		},
		{
			name:            "re-login into workspace profile prompts when workspace_id missing from args",
			authArguments:   auth.AuthArguments{AccountID: "test-unified-account"},
			existingProfile: workspaceProfile,
			want:            true,
		},
		{
			name:            "account-only profile for a different account still prompts",
			authArguments:   auth.AuthArguments{AccountID: "different-account"},
			existingProfile: legacyAccountProfile,
			want:            true,
		},
		{
			name:          "skipWorkspace suppresses the prompt",
			authArguments: auth.AuthArguments{AccountID: "acc"},
			skipWorkspace: true,
			want:          false,
		},
		{
			name:          "no account_id means no prompt",
			authArguments: auth.AuthArguments{},
			want:          false,
		},
		{
			name:          "workspace_id already known means no prompt",
			authArguments: auth.AuthArguments{AccountID: "acc", WorkspaceID: "12345"},
			want:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldPromptWorkspace(&tt.authArguments, tt.existingProfile, tt.skipWorkspace)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSetHostAndAccountId_URLParamsOverrideProfile(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_FILE", "./testdata/.databrickscfg")
	ctx, _ := cmdio.SetupTest(t.Context(), cmdio.TestOptions{})

	unifiedWorkspaceProfile := loadTestProfile(t, ctx, "unified-workspace")

	// The profile has workspace_id=123456789, but the URL has ?o=99999.
	// URL params should win over profile values.
	args := auth.AuthArguments{
		Host:      "https://unified.databricks.com?o=99999",
		AccountID: "test-unified-account",
	}
	err := setHostAndAccountId(ctx, unifiedWorkspaceProfile, &args, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "https://unified.databricks.com", args.Host)
	assert.Equal(t, "99999", args.WorkspaceID)
}

func TestGetProfileName(t *testing.T) {
	tests := []struct {
		name string
		args *auth.AuthArguments
		want string
	}{
		{
			name: "account id set",
			args: &auth.AuthArguments{Host: "https://db-deco-test.databricks.com", AccountID: "abc-123"},
			want: "ACCOUNT-abc-123",
		},
		{
			name: "no account id falls back to host",
			args: &auth.AuthArguments{Host: "https://db-deco-test.databricks.com"},
			want: "db-deco-test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getProfileName(tt.args))
		})
	}
}

// TestSkipWorkspaceProfileNameUsesDiscoveredAccountID verifies that the
// pre-naming discovery block populates AccountID from .well-known so the
// profile-name prompt suggests ACCOUNT-<account-id> instead of the host-based
// default.
func TestSkipWorkspaceProfileNameUsesDiscoveredAccountID(t *testing.T) {
	server := newDiscoveryServer(t, map[string]any{
		"account_id":    "abc-123",
		"oidc_endpoint": "https://spog.example.com/oidc/accounts/abc-123",
	})

	ctx := t.Context()
	args := &auth.AuthArguments{Host: server.URL}

	// Mirrors the pre-naming block in newLoginCommand's RunE for --skip-workspace.
	params := auth.ExtractHostQueryParams(args.Host)
	args.Host = params.Host
	if args.AccountID == "" {
		args.AccountID = params.AccountID
	}
	runHostDiscovery(ctx, args)

	assert.Equal(t, "abc-123", args.AccountID)
	assert.Equal(t, "ACCOUNT-abc-123", getProfileName(args))
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
			name:    "resource is incompatible",
			setFlag: "resource",
			flagVal: "https://workspace.test/ai-gateway/mcp-services/system.ai.github",
			wantErr: "--resource requires --host to be specified",
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
			cmd.Flags().Bool("configure-cluster", false, "")
			cmd.Flags().Bool("configure-serverless", false, "")
			cmd.Flags().StringArray("resource", nil, "")

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
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     time.Second,
		scopes:      "all-apis, ,sql,",
		browserFunc: func(string) error { return nil },
		tokenStore:  newTestStore(),
	})
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

func TestDiscoveryLogin_ProfileSaveFailureDoesNotStoreToken(t *testing.T) {
	tmpDir := t.TempDir()
	parentFile := filepath.Join(tmpDir, "not-a-directory")
	require.NoError(t, os.WriteFile(parentFile, nil, 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(parentFile, ".databrickscfg"))

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://workspace.example.test")

	dc := &fakeDiscoveryClient{
		oauthArg: oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{
			token: &oauth2.Token{AccessToken: "test-token"},
		},
		introspectionErr: errors.New("introspection failed"),
	}
	store := &countingStore{Store: storage.NewMemoryStore()}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     time.Second,
		browserFunc: func(string) error { return nil },
		tokenStore:  store,
	})

	require.ErrorContains(t, err, "saving profile")
	assert.Zero(t, store.putCalls)
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

	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:              dc,
		profileName:     "DISCOVERY",
		timeout:         time.Second,
		existingProfile: existingProfile,
		browserFunc:     func(string) error { return nil },
		tokenStore:      newTestStore(),
	})
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

	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:              dc,
		profileName:     "DISCOVERY",
		timeout:         time.Second,
		existingProfile: existingProfile,
		browserFunc:     func(string) error { return nil },
		tokenStore:      newTestStore(),
	})
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
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     time.Second,
		browserFunc: func(string) error { return nil },
		tokenStore:  newTestStore(),
	})
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
		Name:     "DISCOVERY",
		Host:     "https://old-workspace.example.com",
		Scopes:   "sql,clusters",
		AuthType: authTypeDatabricksCLI,
		ClientID: "custom-client-id",
	}

	// No --scopes flag (empty string), should fall back to existing profile scopes.
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:              dc,
		profileName:     "DISCOVERY",
		timeout:         time.Second,
		clientID:        existingProfile.ClientID,
		existingProfile: existingProfile,
		browserFunc:     func(string) error { return nil },
		tokenStore:      newTestStore(),
	})
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://workspace.example.com", savedProfile.Host)
	assert.Equal(t, "sql,clusters", savedProfile.Scopes)
	assert.Equal(t, "custom-client-id", savedProfile.ClientID)
}

func TestDiscoveryLogin_ExplicitClientIDOverridesExistingProfile(t *testing.T) {
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
		Name:     "DISCOVERY",
		Host:     "https://old-workspace.example.com",
		AuthType: authTypeDatabricksCLI,
		ClientID: "profile-client-id",
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:              dc,
		profileName:     "DISCOVERY",
		timeout:         time.Second,
		clientID:        "flag-client-id",
		existingProfile: existingProfile,
		browserFunc:     func(string) error { return nil },
		tokenStore:      newTestStore(),
	})
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "flag-client-id", savedProfile.ClientID)
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
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:              dc,
		profileName:     "DISCOVERY",
		timeout:         time.Second,
		scopes:          "all-apis",
		existingProfile: existingProfile,
		browserFunc:     func(string) error { return nil },
		tokenStore:      newTestStore(),
	})
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
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     time.Second,
		browserFunc: func(string) error { return nil },
		tokenStore:  newTestStore(),
	})
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, server.URL, savedProfile.Host)
	assert.Equal(t, "discovered-account", savedProfile.AccountID, "account_id should come from host discovery")
	assert.Equal(t, "discovered-ws", savedProfile.WorkspaceID, "workspace_id should come from host discovery")
}

func TestShouldResolveProvisionedURL(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		account  string
		expected bool
	}{
		{"classic account host with account id", "https://accounts.cloud.databricks.com", "abc-123", true},
		{"account host without scheme", "accounts.cloud.databricks.com", "abc-123", true},
		{"classic account host without account id", "https://accounts.cloud.databricks.com", "", false},
		{"workspace host with account id", "https://dbc-abc.cloud.databricks.com", "abc-123", false},
		{"unified host with account id", "https://mycompany.databricks.com", "abc-123", false},
		{"empty host with account id", "", "abc-123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, shouldResolveProvisionedURL(tt.host, tt.account))
		})
	}
}

// rewriteHostTransport routes every request to target (a test server), keeping
// the request's path and query, so a lookup addressed to a classic account host
// can be served locally.
type rewriteHostTransport struct {
	target string
}

func (rt rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := url.Parse(rt.target)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	return http.DefaultTransport.RoundTrip(req)
}

// assertNoRequestTransport fails the test if any HTTP request is made through it.
type assertNoRequestTransport struct {
	t *testing.T
}

func (rt assertNoRequestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.t.Errorf("unexpected provisioned-URL lookup to %s", req.URL)
	return nil, errors.New("unexpected request")
}

func TestDiscoveryLogin_AccountSelectionResolvesProvisionedURL(t *testing.T) {
	// The provisioned-urls endpoint returns the account's primary SPOG host.
	spogServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/accounts/introspection-account/provisioned-urls/primary", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"url": "https://dbc-spog.cloud.databricks.com"}`))
	}))
	defer spogServer.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	require.NoError(t, os.WriteFile(configPath, []byte(""), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	// A classic account host triggers the provisioned-URL lookup. The reserved
	// .invalid TLD keeps host metadata discovery from making a real network call
	// (it fast-fails, so account_id falls back to introspection).
	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://accounts.invalid")

	dc := &fakeDiscoveryClient{
		oauthArg:       oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{token: &oauth2.Token{AccessToken: "test-token"}},
		introspection:  &auth.IntrospectionResult{AccountID: "introspection-account"},
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     5 * time.Second,
		browserFunc: func(string) error { return nil },
		tokenStore:  newTestStore(),
		httpClient:  &http.Client{Transport: rewriteHostTransport{target: spogServer.URL}},
	})
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://dbc-spog.cloud.databricks.com", savedProfile.Host, "host should be switched to the account's primary provisioned URL")
	assert.Equal(t, "introspection-account", savedProfile.AccountID)
}

func TestDiscoveryLogin_WorkspaceSelectionKeepsDiscoveredHost(t *testing.T) {
	// A workspace host is not a classic account host, so even though token
	// introspection backfills an account_id, the provisioned-URL lookup must not
	// run and the discovered workspace host must be preserved.
	server := newDiscoveryServer(t, map[string]any{
		"workspace_id": "discovered-ws",
	})

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	require.NoError(t, os.WriteFile(configPath, []byte(""), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost(server.URL)

	dc := &fakeDiscoveryClient{
		oauthArg:       oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{token: &oauth2.Token{AccessToken: "test-token"}},
		introspection:  &auth.IntrospectionResult{AccountID: "introspection-account"},
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     5 * time.Second,
		browserFunc: func(string) error { return nil },
		tokenStore:  newTestStore(),
		// Any provisioned-URL lookup here would be a bug: fail the test if attempted.
		httpClient: &http.Client{Transport: assertNoRequestTransport{t: t}},
	})
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, server.URL, savedProfile.Host, "workspace host must be preserved, not rewritten to a provisioned URL")
	assert.Equal(t, "introspection-account", savedProfile.AccountID)
}

func TestDiscoveryLogin_AccountSelectionLookupFailureKeepsHost(t *testing.T) {
	// When the provisioned-URL lookup fails, login still succeeds and the profile
	// keeps the discovered account host (best-effort enrichment).
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	require.NoError(t, os.WriteFile(configPath, []byte(""), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost("https://accounts.invalid")

	dc := &fakeDiscoveryClient{
		oauthArg:       oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{token: &oauth2.Token{AccessToken: "test-token"}},
		introspection:  &auth.IntrospectionResult{AccountID: "introspection-account"},
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     5 * time.Second,
		browserFunc: func(string) error { return nil },
		tokenStore:  newTestStore(),
		httpClient:  &http.Client{Transport: rewriteHostTransport{target: failServer.URL}},
	})
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://accounts.invalid", savedProfile.Host, "host stays the discovered account host when the lookup fails")
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
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     time.Second,
		browserFunc: func(string) error { return nil },
		tokenStore:  newTestStore(),
	})
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
		Name:        "DISCOVERY",
		Host:        "https://old-unified.databricks.com",
		AccountID:   "old-account",
		WorkspaceID: "999999",
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:              dc,
		profileName:     "DISCOVERY",
		timeout:         time.Second,
		existingProfile: existingProfile,
		browserFunc:     func(string) error { return nil },
		tokenStore:      newTestStore(),
	})
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://new-workspace.example.com", savedProfile.Host)
	// Stale routing fields must be cleared.
	assert.Empty(t, savedProfile.AccountID, "stale account_id should be cleared")
	assert.Empty(t, savedProfile.WorkspaceID, "stale workspace_id should be cleared on introspection failure")

	// Verify the experimental_is_unified_host INI key was also cleared from disk.
	raw, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "experimental_is_unified_host")
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
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:              dc,
		profileName:     "DISCOVERY",
		timeout:         time.Second,
		existingProfile: existingProfile,
		browserFunc:     func(string) error { return nil },
		tokenStore:      newTestStore(),
	})
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, "https://new-workspace.example.com", savedProfile.Host)
	assert.Equal(t, "fresh-account", savedProfile.AccountID, "account_id should be saved from introspection")
	assert.Equal(t, "222222", savedProfile.WorkspaceID, "workspace_id should be updated to fresh introspection value")
}

func TestDiscoveryLogin_OverridesHostFromEnv(t *testing.T) {
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

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.Set(ctx, "DATABRICKS_DISCOVERY_HOST", "https://login.staging.test")
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     time.Second,
		browserFunc: func(string) error { return nil },
		tokenStore:  newTestStore(),
	})
	require.NoError(t, err)

	assert.Contains(t, stderr.String(), "Opening https://login.staging.test in your browser...")
	assert.NotContains(t, stderr.String(), "Opening login.databricks.com in your browser...")
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

func TestShouldResolveWorkspacePrimaryURL(t *testing.T) {
	tests := []struct {
		name     string
		args     auth.AuthArguments
		expected bool
	}{
		{"workspace host with account and workspace id", auth.AuthArguments{Host: "https://dbc-abc.cloud.databricks.com", AccountID: "acc", WorkspaceID: "123"}, true},
		{"workspace host without workspace id", auth.AuthArguments{Host: "https://dbc-abc.cloud.databricks.com", AccountID: "acc"}, false},
		{"workspace host with none workspace id", auth.AuthArguments{Host: "https://dbc-abc.cloud.databricks.com", AccountID: "acc", WorkspaceID: auth.WorkspaceIDNone}, false},
		{"workspace host without account id", auth.AuthArguments{Host: "https://dbc-abc.cloud.databricks.com", WorkspaceID: "123"}, false},
		{"classic account host", auth.AuthArguments{Host: "https://accounts.cloud.databricks.com", AccountID: "acc", WorkspaceID: "123"}, false},
		{"unified host", auth.AuthArguments{Host: "https://acme.databricks.com", AccountID: "acc", WorkspaceID: "123", DiscoveryURL: "https://acme.databricks.com/oidc/accounts/acc/.well-known/oauth-authorization-server"}, false},
		{"empty host", auth.AuthArguments{AccountID: "acc", WorkspaceID: "123"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, shouldResolveWorkspacePrimaryURL(&tt.args))
		})
	}
}

// newUnifiedHostServer serves /.well-known/databricks-config for a unified
// (SPOG) host with an account-scoped OIDC endpoint.
func newUnifiedHostServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"account_id":    "spog-account",
				"oidc_endpoint": "http://" + r.Host + "/oidc/accounts/{account_id}",
				"host_type":     "UNIFIED_HOST",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	return server
}

// newSpogServer serves /.well-known/databricks-config for a unified (SPOG)
// host with an account-scoped OIDC endpoint, and workspace-level OAuth
// metadata (selected with ?o=) whose endpoints are on the host returned by
// workspaceHost, as the SPOG host serves them for a workspace.
func newSpogServer(t *testing.T, workspaceHost func() string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/databricks-config":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"account_id":    "spog-account",
				"oidc_endpoint": "http://" + r.Host + "/oidc/accounts/{account_id}",
				"host_type":     "UNIFIED_HOST",
			})
		case "/oidc/.well-known/oauth-authorization-server":
			if r.URL.Query().Get("o") == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"authorization_endpoint": workspaceHost() + "/oidc/v1/authorize",
				"token_endpoint":         workspaceHost() + "/oidc/v1/token",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// newSpogWorkspacePair returns a canonical workspace server whose account's
// primary URL is a SPOG server that serves the workspace's OAuth from the
// workspace server.
func newSpogWorkspacePair(t *testing.T) (workspace, spog *httptest.Server) {
	t.Helper()
	var workspaceURL string
	spog = newSpogServer(t, func() string { return workspaceURL })
	workspace = newDiscoveryServer(t, map[string]any{
		"account_id":   "spog-account",
		"workspace_id": "12345",
		"primary_url":  spog.URL,
	})
	workspaceURL = workspace.URL
	return workspace, spog
}

func TestSetHostAndAccountId_DoesNotSwitchToPrimaryURL(t *testing.T) {
	// setHostAndAccountId also serves auth token, which must keep the profile's
	// host so the cached token is found and refreshed where it was issued.
	spog := newUnifiedHostServer(t)
	workspace := newDiscoveryServer(t, map[string]any{
		"account_id":   "spog-account",
		"workspace_id": "12345",
		"primary_url":  spog.URL,
	})

	args := &auth.AuthArguments{Host: workspace.URL}
	err := setHostAndAccountId(t.Context(), nil, args, []string{})
	require.NoError(t, err)

	assert.Equal(t, workspace.URL, args.Host)
	assert.False(t, auth.HasUnifiedHostSignal(args.DiscoveryURL), "discovery URL %q", args.DiscoveryURL)
}

func TestDiscoveryLogin_WorkspaceSelectionLogsInAtPrimaryURL(t *testing.T) {
	spog := newUnifiedHostServer(t)
	workspace := newDiscoveryServer(t, map[string]any{
		"account_id":   "spog-account",
		"workspace_id": "12345",
		"primary_url":  spog.URL,
	})

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	require.NoError(t, os.WriteFile(configPath, []byte(""), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost(workspace.URL)

	dc := &fakeDiscoveryClient{
		oauthArg:       oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{token: &oauth2.Token{AccessToken: "workspace-token"}},
		followUpAuths:  []discoveryPersistentAuth{&fakeDiscoveryPersistentAuth{token: &oauth2.Token{AccessToken: "spog-token"}}},
		introspection:  &auth.IntrospectionResult{},
	}
	store := &inMemoryStore{Tokens: map[string]*oauth2.Token{}}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     5 * time.Second,
		browserFunc: func(string) error { return nil },
		tokenStore:  store,
	})
	require.NoError(t, err)

	assert.Equal(t, 2, dc.newPersistentAuthCall, "expected a second login at the primary URL")
	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, spog.URL, savedProfile.Host)
	assert.Equal(t, "spog-account", savedProfile.AccountID)
	assert.Equal(t, "12345", savedProfile.WorkspaceID)
	require.Contains(t, store.Tokens, "DISCOVERY")
	assert.Equal(t, "spog-token", store.Tokens["DISCOVERY"].AccessToken)
}

func TestDiscoveryLogin_PrimaryURLLoginFailureKeepsWorkspace(t *testing.T) {
	spog := newUnifiedHostServer(t)
	workspace := newDiscoveryServer(t, map[string]any{
		"account_id":   "spog-account",
		"workspace_id": "12345",
		"primary_url":  spog.URL,
	})

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	require.NoError(t, os.WriteFile(configPath, []byte(""), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost(workspace.URL)

	dc := &fakeDiscoveryClient{
		oauthArg:       oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{token: &oauth2.Token{AccessToken: "workspace-token"}},
		followUpAuths:  []discoveryPersistentAuth{&fakeDiscoveryPersistentAuth{challengeErr: errors.New("browser closed")}},
		introspection:  &auth.IntrospectionResult{},
	}
	store := &inMemoryStore{Tokens: map[string]*oauth2.Token{}}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     5 * time.Second,
		browserFunc: func(string) error { return nil },
		tokenStore:  store,
	})
	require.NoError(t, err)

	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, workspace.URL, savedProfile.Host)
	assert.Equal(t, "12345", savedProfile.WorkspaceID)
	require.Contains(t, store.Tokens, "DISCOVERY")
	assert.Equal(t, "workspace-token", store.Tokens["DISCOVERY"].AccessToken)
}

type unifiedPathEndpointSupplier struct {
	MockApiClient
}

func (*unifiedPathEndpointSupplier) GetUnifiedOAuthEndpoints(_ context.Context, host, accountID string) (*u2m.OAuthAuthorizationServer, error) {
	return &u2m.OAuthAuthorizationServer{
		AuthorizationEndpoint: host + "/oidc/accounts/" + accountID + "/v1/authorize",
		TokenEndpoint:         host + "/oidc/accounts/" + accountID + "/v1/token",
	}, nil
}

func declineInBrowser(t *testing.T, authorizeURL *string) func(string) error {
	return func(rawURL string) error {
		*authorizeURL = rawURL
		u, err := url.Parse(rawURL)
		require.NoError(t, err)
		q := u.Query()
		callback := q.Get("redirect_uri") + "?" + url.Values{
			"error":             {"access_denied"},
			"error_description": {"declined in test"},
			"state":             {q.Get("state")},
		}.Encode()
		go func() {
			resp, err := http.Get(callback)
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}
}

func hostOfURL(t *testing.T, rawURL string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	require.NoError(t, err)
	return u.Host
}

func TestDiscoveryLogin_PrimaryURLLoginUsesAccountLevelOAuthAndLoginOptions(t *testing.T) {
	spog := newUnifiedHostServer(t)
	workspace := newDiscoveryServer(t, map[string]any{
		"account_id":   "spog-account",
		"workspace_id": "12345",
		"primary_url":  spog.URL,
	})

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".databrickscfg")
	require.NoError(t, os.WriteFile(configPath, []byte(""), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
	require.NoError(t, err)
	oauthArg.SetDiscoveredHost(workspace.URL)

	// The second login runs a real PersistentAuth with the options discoveryLogin
	// passes, so the authorize URL shows which OAuth endpoints, client ID and
	// scopes it would use.
	var authorizeURL string
	dc := &fakeDiscoveryClient{
		oauthArg:       oauthArg,
		persistentAuth: &fakeDiscoveryPersistentAuth{token: &oauth2.Token{AccessToken: "workspace-token"}},
		newFollowUpAuth: func(ctx context.Context, opts ...u2m.PersistentAuthOption) (discoveryPersistentAuth, error) {
			return u2m.NewPersistentAuth(ctx, append(opts, u2m.WithOAuthEndpointSupplier(&unifiedPathEndpointSupplier{}))...)
		},
		introspection: &auth.IntrospectionResult{},
	}

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	err = discoveryLogin(ctx, discoveryLoginInputs{
		dc:          dc,
		profileName: "DISCOVERY",
		timeout:     10 * time.Second,
		scopes:      "sql,jobs",
		clientID:    "custom-client",
		browserFunc: declineInBrowser(t, &authorizeURL),
		tokenStore:  newTestStore(),
	})
	require.NoError(t, err)

	require.NotEmpty(t, authorizeURL, "the second login should open the browser")
	u, err := url.Parse(authorizeURL)
	require.NoError(t, err)
	assert.Equal(t, hostOfURL(t, spog.URL), u.Host)
	assert.Equal(t, "/oidc/accounts/spog-account/v1/authorize", u.Path)
	assert.Equal(t, "custom-client", u.Query().Get("client_id"))
	scopes := strings.Fields(u.Query().Get("scope"))
	assert.Contains(t, scopes, "sql")
	assert.Contains(t, scopes, "jobs")

	// Declining the second login keeps the workspace profile.
	savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, savedProfile)
	assert.Equal(t, workspace.URL, savedProfile.Host)
}

func TestDiscoveryLogin_PrimaryURLLoginSetupFailureKeepsWorkspace(t *testing.T) {
	tests := []struct {
		name string
		// primaryURL maps the SPOG server's URL to the primary_url the workspace reports.
		primaryURL      func(spogURL string) string
		followUpErr     error
		wantFollowUpRun bool
	}{
		{
			name:            "second login can't be set up",
			primaryURL:      func(spogURL string) string { return spogURL },
			followUpErr:     errors.New("setup failed"),
			wantFollowUpRun: true,
		},
		{
			// OAuth arguments only accept http for 127.0.0.1, so a localhost
			// primary URL passes discovery but fails ToOAuthArgument.
			name:       "oauth argument for the primary URL can't be built",
			primaryURL: func(spogURL string) string { return strings.Replace(spogURL, "127.0.0.1", "localhost", 1) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spog := newUnifiedHostServer(t)
			workspace := newDiscoveryServer(t, map[string]any{
				"account_id":   "spog-account",
				"workspace_id": "12345",
				"primary_url":  tt.primaryURL(spog.URL),
			})

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ".databrickscfg")
			require.NoError(t, os.WriteFile(configPath, []byte(""), 0o600))
			t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

			oauthArg, err := u2m.NewBasicDiscoveryOAuthArgument("DISCOVERY")
			require.NoError(t, err)
			oauthArg.SetDiscoveredHost(workspace.URL)

			followUpRun := false
			dc := &fakeDiscoveryClient{
				oauthArg:       oauthArg,
				persistentAuth: &fakeDiscoveryPersistentAuth{token: &oauth2.Token{AccessToken: "workspace-token"}},
				newFollowUpAuth: func(context.Context, ...u2m.PersistentAuthOption) (discoveryPersistentAuth, error) {
					followUpRun = true
					return nil, tt.followUpErr
				},
				introspection: &auth.IntrospectionResult{},
			}
			store := &inMemoryStore{Tokens: map[string]*oauth2.Token{}}

			ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
			err = discoveryLogin(ctx, discoveryLoginInputs{
				dc:          dc,
				profileName: "DISCOVERY",
				timeout:     5 * time.Second,
				browserFunc: func(string) error { return nil },
				tokenStore:  store,
			})
			require.NoError(t, err)

			assert.Equal(t, tt.wantFollowUpRun, followUpRun)
			savedProfile, err := loadProfileByName(ctx, "DISCOVERY", profile.DefaultProfiler)
			require.NoError(t, err)
			require.NotNil(t, savedProfile)
			assert.Equal(t, workspace.URL, savedProfile.Host)
			require.Contains(t, store.Tokens, "DISCOVERY")
			assert.Equal(t, "workspace-token", store.Tokens["DISCOVERY"].AccessToken)
		})
	}
}

func TestSwitchToWorkspacePrimaryURL(t *testing.T) {
	spog := newUnifiedHostServer(t)
	notUnified := newDiscoveryServer(t, map[string]any{"workspace_id": "999"})
	lookupFails := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(lookupFails.Close)

	tests := []struct {
		name            string
		host            string
		workspaceID     string
		wantSwitched    bool
		wantWorkspaceID string
	}{
		{
			name:            "switches to the primary url",
			host:            newDiscoveryServer(t, map[string]any{"account_id": "spog-account", "workspace_id": "12345", "primary_url": spog.URL + "/"}).URL,
			workspaceID:     "12345",
			wantSwitched:    true,
			wantWorkspaceID: "12345",
		},
		{
			// A workspace_id inherited from an existing profile can belong to
			// another workspace; on SPOG it decides routing.
			name:            "uses the host's workspace id over an inherited one",
			host:            newDiscoveryServer(t, map[string]any{"account_id": "spog-account", "workspace_id": "12345", "primary_url": spog.URL}).URL,
			workspaceID:     "67890",
			wantSwitched:    true,
			wantWorkspaceID: "12345",
		},
		{
			name:            "no primary url",
			host:            newDiscoveryServer(t, map[string]any{"account_id": "spog-account", "workspace_id": "12345"}).URL,
			workspaceID:     "12345",
			wantWorkspaceID: "12345",
		},
		{
			name:            "primary url is not a unified host",
			host:            newDiscoveryServer(t, map[string]any{"account_id": "spog-account", "workspace_id": "12345", "primary_url": notUnified.URL}).URL,
			workspaceID:     "12345",
			wantWorkspaceID: "12345",
		},
		{
			name:            "lookup fails",
			host:            lookupFails.URL,
			workspaceID:     "12345",
			wantWorkspaceID: "12345",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &auth.AuthArguments{Host: tt.host, AccountID: "spog-account", WorkspaceID: tt.workspaceID}
			switchToWorkspacePrimaryURL(t.Context(), args)

			if tt.wantSwitched {
				assert.Equal(t, spog.URL, args.Host)
				assert.True(t, auth.HasUnifiedHostSignal(args.DiscoveryURL), "discovery URL %q", args.DiscoveryURL)
			} else {
				assert.Equal(t, tt.host, args.Host)
			}
			assert.Equal(t, "spog-account", args.AccountID)
			assert.Equal(t, tt.wantWorkspaceID, args.WorkspaceID)
		})
	}
}

func TestSetLoginHostAndAccountId_Resources(t *testing.T) {
	workspace, spog := newSpogWorkspacePair(t)
	// A unified host that serves no workspace-level OAuth metadata.
	spogWithoutWorkspaceOAuth := newUnifiedHostServer(t)
	resources := []string{workspace.URL + "/ai-gateway/mcp-services/system.ai.github"}

	tests := []struct {
		name        string
		host        string
		workspaceID string
		resources   []string
		wantHost    string
		wantErr     string
	}{
		{name: "spog host with ?o= logs in at the workspace host", host: spog.URL + "?o=12345", resources: resources, wantHost: workspace.URL},
		{name: "spog host with --workspace-id logs in at the workspace host", host: spog.URL, workspaceID: "12345", resources: resources, wantHost: workspace.URL},
		{name: "spog host without a workspace", host: spog.URL, resources: resources, wantErr: "--resource on a unified host needs a workspace"},
		{name: "spog host that doesn't serve the workspace", host: spogWithoutWorkspaceOAuth.URL + "?o=12345", resources: resources, wantErr: "Pass the workspace URL with --host instead"},
		{name: "workspace host is kept", host: workspace.URL, resources: resources, wantHost: workspace.URL},
		{name: "spog host without resources is kept", host: spog.URL + "?o=12345", wantHost: spog.URL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &auth.AuthArguments{Host: tt.host, WorkspaceID: tt.workspaceID}
			err := setLoginHostAndAccountId(cmdio.MockDiscard(t.Context()), nil, args, []string{}, tt.resources)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, args.Host)
		})
	}
}
