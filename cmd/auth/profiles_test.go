package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfiles(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	// Create a config file with a profile
	err := databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: configFile,
		Profile:    "profile1",
		Host:       "abc.cloud.databricks.com",
		Token:      "token1",
		AuthType:   "pat",
	})
	require.NoError(t, err)

	// Let the environment think we're using another profile
	t.Setenv("DATABRICKS_HOST", "https://def.cloud.databricks.com")
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}

	// Load the profile
	profile := &profileMetadata{Name: "profile1"}
	profile.Load(ctx, configFile, true)

	// Check the profile
	assert.Equal(t, "profile1", profile.Name)
	assert.Equal(t, "https://abc.cloud.databricks.com", profile.Host)
	assert.Equal(t, "aws", profile.Cloud)
	assert.Equal(t, "pat", profile.AuthType)
	assert.Equal(t, profileStatusUnvalidated, profile.Status)
	assert.Nil(t, profile.Valid, "Valid should be unset for unvalidated")
}

func TestProfilesDefaultMarker(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	// Create two profiles.
	for _, name := range []string{"profile-a", "profile-b"} {
		err := databrickscfg.SaveToProfile(ctx, &config.Config{
			ConfigFile: configFile,
			Profile:    name,
			Host:       "https://" + name + ".cloud.databricks.com",
			Token:      "token",
		})
		require.NoError(t, err)
	}

	// Set profile-a as the default.
	err := databrickscfg.SetDefaultProfile(ctx, "profile-a", configFile)
	require.NoError(t, err)

	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}

	// Read back the default profile and verify.
	defaultProfile, err := databrickscfg.GetDefaultProfile(ctx, configFile)
	require.NoError(t, err)
	assert.Equal(t, "profile-a", defaultProfile)
}

// newSPOGServer creates a mock SPOG server that returns account-scoped OIDC.
// It serves both validation endpoints since SPOG workspace profiles (with a
// real workspace_id) need CurrentUser.Me, while account profiles need
// Workspaces.List. The workspace-only newWorkspaceServer omits the account
// endpoint to prove routing correctness for non-SPOG hosts.
func newSPOGServer(t *testing.T, accountID string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/databricks-config":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"account_id":    accountID,
				"oidc_endpoint": r.Host + "/oidc/accounts/" + accountID,
			})
		case "/api/2.0/accounts/" + accountID + "/workspaces":
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case "/api/2.0/preview/scim/v2/Me":
			// SPOG workspace profiles also need CurrentUser.Me to succeed.
			_ = json.NewEncoder(w).Encode(map[string]any{"userName": "test-user"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// newWorkspaceServer creates a mock workspace server that returns workspace-scoped
// OIDC and only serves the workspace validation endpoint. The account validation
// endpoint returns 404 to prove the workspace path was taken.
func newWorkspaceServer(t *testing.T, accountID string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/databricks-config":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"account_id":    accountID,
				"oidc_endpoint": r.Host + "/oidc",
			})
		case "/api/2.0/preview/scim/v2/Me":
			_ = json.NewEncoder(w).Encode(map[string]any{"userName": "test-user"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestProfileLoadSPOGConfigType(t *testing.T) {
	spogServer := newSPOGServer(t, "spog-acct")
	wsServer := newWorkspaceServer(t, "ws-acct")

	cases := []struct {
		name        string
		host        string
		accountID   string
		workspaceID string
	}{
		{
			name:      "SPOG account profile validated as account",
			host:      spogServer.URL,
			accountID: "spog-acct",
		},
		{
			name:        "SPOG workspace profile validated as workspace",
			host:        spogServer.URL,
			accountID:   "spog-acct",
			workspaceID: "ws-123",
		},
		{
			name:        "SPOG profile with workspace_id=none validated as account",
			host:        spogServer.URL,
			accountID:   "spog-acct",
			workspaceID: "none",
		},
		{
			name:      "classic workspace with account_id from discovery stays workspace",
			host:      wsServer.URL,
			accountID: "ws-acct",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			configFile := filepath.Join(dir, ".databrickscfg")
			t.Setenv("HOME", dir)
			if runtime.GOOS == "windows" {
				t.Setenv("USERPROFILE", dir)
			}

			content := "[test-profile]\nhost = " + tc.host + "\ntoken = test-token\n"
			if tc.accountID != "" {
				content += "account_id = " + tc.accountID + "\n"
			}
			if tc.workspaceID != "" {
				content += "workspace_id = " + tc.workspaceID + "\n"
			}
			require.NoError(t, os.WriteFile(configFile, []byte(content), 0o600))

			p := &profileMetadata{
				Name:      "test-profile",
				Host:      tc.host,
				AccountID: tc.accountID,
			}
			p.Load(t.Context(), configFile, false)

			assert.Equal(t, profileStatusValid, p.Status, "status mismatch")
			require.NotNil(t, p.Valid)
			assert.True(t, *p.Valid)
			assert.NotEmpty(t, p.Host, "Host should be set")
			assert.NotEmpty(t, p.AuthType, "AuthType should be set")
		})
	}
}

func TestClassicAccountsHostConfigType(t *testing.T) {
	// Classic accounts.* hosts can't be tested through Load() because httptest
	// generates 127.0.0.1 URLs. Verify directly that ConfigType() classifies
	// them as AccountConfig, so the SPOG override is never needed.
	cfg := &config.Config{
		Host:      "https://accounts.cloud.databricks.com",
		AccountID: "acct-123",
	}
	assert.Equal(t, config.AccountConfig, cfg.ConfigType())

	// Even with SPOG-like discovery data, accounts.* stays AccountConfig.
	cfg.DiscoveryURL = "https://accounts.cloud.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server"
	assert.Equal(t, config.AccountConfig, cfg.ConfigType())
}

func TestProfileLoadNoDiscoveryStaysWorkspace(t *testing.T) {
	// When .well-known returns 404 and the unified-host fallback is false,
	// the SPOG override should NOT trigger even if account_id is set. The
	// profile should stay WorkspaceConfig and validate via CurrentUser.Me.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/databricks-config":
			w.WriteHeader(http.StatusNotFound)
		case "/api/2.0/preview/scim/v2/Me":
			_ = json.NewEncoder(w).Encode(map[string]any{"userName": "test-user"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}

	content := "[ws-profile]\nhost = " + server.URL + "\ntoken = test-token\naccount_id = some-acct\n"
	require.NoError(t, os.WriteFile(configFile, []byte(content), 0o600))

	p := &profileMetadata{
		Name:      "ws-profile",
		Host:      server.URL,
		AccountID: "some-acct",
	}
	p.Load(t.Context(), configFile, false)

	assert.Equal(t, profileStatusValid, p.Status, "should validate as workspace when discovery is unavailable")
	assert.NotEmpty(t, p.Host)
	assert.Equal(t, "pat", p.AuthType)
}

func TestClassifyValidationError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus profileStatus
		wantMsgSub string
	}{
		{
			name:       "nil error -> valid",
			err:        nil,
			wantStatus: profileStatusValid,
		},
		{
			name:       "deadline exceeded -> unknown timeout",
			err:        context.DeadlineExceeded,
			wantStatus: profileStatusUnknown,
			wantMsgSub: "validation timed out",
		},
		{
			name:       "url.Error wrapping deadline -> unknown timeout",
			err:        &url.Error{Op: "Get", URL: "https://x.test/", Err: context.DeadlineExceeded},
			wantStatus: profileStatusUnknown,
			wantMsgSub: "validation timed out",
		},
		{
			name:       "401 -> invalid with auth remediation",
			err:        &apierr.APIError{StatusCode: 401, Message: "unauthorized"},
			wantStatus: profileStatusInvalid,
			wantMsgSub: "databricks auth login -p test-profile",
		},
		{
			name:       "403 -> invalid with permission message",
			err:        &apierr.APIError{StatusCode: 403, Message: "forbidden"},
			wantStatus: profileStatusInvalid,
			wantMsgSub: "credentials lack permission",
		},
		{
			name:       "500 -> unknown server error",
			err:        &apierr.APIError{StatusCode: 500, Message: "internal"},
			wantStatus: profileStatusUnknown,
			wantMsgSub: "server error: 500",
		},
		{
			name:       "503 -> unknown server error",
			err:        &apierr.APIError{StatusCode: 503, Message: "unavailable"},
			wantStatus: profileStatusUnknown,
			wantMsgSub: "server error: 503",
		},
		{
			name:       "network error -> unknown could-not-reach",
			err:        &url.Error{Op: "Get", URL: "https://x.test/", Err: errors.New("dial tcp: lookup x.test: no such host")},
			wantStatus: profileStatusUnknown,
			wantMsgSub: "could not reach host",
		},
		{
			name:       "fallthrough -> unknown with raw message",
			err:        errors.New("strange unknown failure"),
			wantStatus: profileStatusUnknown,
			wantMsgSub: "strange unknown failure",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, msg := classifyValidationError("test-profile", tc.err)
			assert.Equal(t, tc.wantStatus, status)
			if tc.wantMsgSub == "" {
				assert.Empty(t, msg)
			} else {
				assert.Contains(t, msg, tc.wantMsgSub)
			}
		})
	}
}

func TestProfileLoadStatusMatrix(t *testing.T) {
	// statusServer returns a configurable HTTP status for the validation
	// endpoint. .well-known returns 404 so we land on WorkspaceConfig and
	// CurrentUser.Me is the validation API call.
	statusServer := func(t *testing.T, code int) *httptest.Server {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/databricks-config":
				w.WriteHeader(http.StatusNotFound)
			case "/api/2.0/preview/scim/v2/Me":
				w.WriteHeader(code)
				_, _ = w.Write([]byte(`{"error_code":"X","message":"x"}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		t.Cleanup(server.Close)
		return server
	}

	t.Run("401 -> invalid", func(t *testing.T) {
		s := statusServer(t, http.StatusUnauthorized)
		p := loadFromHost(t, s.URL)
		assert.Equal(t, profileStatusInvalid, p.Status)
		require.NotNil(t, p.Valid)
		assert.False(t, *p.Valid)
		assert.Contains(t, p.Error, "databricks auth login")
	})

	t.Run("403 -> invalid", func(t *testing.T) {
		s := statusServer(t, http.StatusForbidden)
		p := loadFromHost(t, s.URL)
		assert.Equal(t, profileStatusInvalid, p.Status)
		require.NotNil(t, p.Valid)
		assert.False(t, *p.Valid)
		assert.Contains(t, p.Error, "permission")
	})

	t.Run("500 -> unknown", func(t *testing.T) {
		s := statusServer(t, http.StatusInternalServerError)
		p := loadFromHost(t, s.URL)
		assert.Equal(t, profileStatusUnknown, p.Status)
		assert.Nil(t, p.Valid, "Valid is omitted for unknown")
		assert.Contains(t, p.Error, "server error")
	})

	t.Run("network down -> unknown", func(t *testing.T) {
		// Start and immediately close the server to simulate a dead host.
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		s.Close()
		p := loadFromHost(t, s.URL)
		assert.Equal(t, profileStatusUnknown, p.Status)
		assert.Nil(t, p.Valid)
		assert.Contains(t, p.Error, "could not reach host")
	})

	t.Run("InvalidConfig -> invalid", func(t *testing.T) {
		// experimental_is_unified_host=true forces HostType=UnifiedHost.
		// Without an account_id (or a SPOG-shaped DiscoveryURL), ResolveConfigType
		// can't pick a side and falls through to InvalidConfig.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(server.Close)

		dir := t.TempDir()
		configFile := filepath.Join(dir, ".databrickscfg")
		t.Setenv("HOME", dir)
		if runtime.GOOS == "windows" {
			t.Setenv("USERPROFILE", dir)
		}
		content := "[bad]\nhost = " + server.URL + "\nexperimental_is_unified_host = true\ntoken = test-token\n"
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0o600))

		p := &profileMetadata{Name: "bad", Host: server.URL}
		p.Load(t.Context(), configFile, false)
		assert.Equal(t, profileStatusInvalid, p.Status)
		require.NotNil(t, p.Valid)
		assert.False(t, *p.Valid)
		assert.Contains(t, p.Error, "fields conflict")
	})

	t.Run("skip-validate -> unvalidated", func(t *testing.T) {
		s := statusServer(t, http.StatusOK)
		p := loadFromHost(t, s.URL, withSkipValidate())
		assert.Equal(t, profileStatusUnvalidated, p.Status)
		assert.Nil(t, p.Valid)
		assert.Empty(t, p.Error)
	})
}

type loadOpts struct {
	skipValidate bool
}

type loadOpt func(*loadOpts)

func withSkipValidate() loadOpt { return func(o *loadOpts) { o.skipValidate = true } }

// loadFromHost writes a single PAT profile pointing at host into a temp
// .databrickscfg, runs Load, and returns the populated profileMetadata.
func loadFromHost(t *testing.T, host string, opts ...loadOpt) *profileMetadata {
	t.Helper()
	o := loadOpts{}
	for _, opt := range opts {
		opt(&o)
	}
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
	content := "[test-profile]\nhost = " + host + "\ntoken = test-token\n"
	require.NoError(t, os.WriteFile(configFile, []byte(content), 0o600))

	p := &profileMetadata{Name: "test-profile", Host: host}
	p.Load(t.Context(), configFile, o.skipValidate)
	return p
}
