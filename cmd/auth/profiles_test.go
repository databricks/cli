package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/databrickscfg"
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
	assert.False(t, profile.Valid, "Valid should be false when validation is skipped")
	assert.Contains(t, profile.ValidReason, "skip-validate")
}

// TestProfileLoadSkipValidateMakesNoRequests guards the --skip-validate
// contract: EnsureResolved would otherwise fetch /.well-known/databricks-config
// for every profile, so the handler counts every request it receives.
func TestProfileLoadSkipValidateMakesNoRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}

	content := "[offline-profile]\nhost = " + server.URL + "\ntoken = test-token\n"
	require.NoError(t, os.WriteFile(configFile, []byte(content), 0o600))

	p := &profileMetadata{Name: "offline-profile", Host: server.URL}
	p.Load(t.Context(), configFile, true)

	assert.Zero(t, requests.Load(), "expected no network calls with skipValidate")
	assert.Equal(t, server.URL, p.Host)
	assert.False(t, p.Valid, "Valid should be false when validation is skipped")
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

			assert.True(t, p.Valid)
			assert.Empty(t, p.ValidReason)
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

	assert.True(t, p.Valid, "should validate as workspace when discovery is unavailable")
	assert.NotEmpty(t, p.Host)
	assert.Equal(t, "pat", p.AuthType)
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

	// Any validation error (auth, permission, server, network) is reported as
	// NO with the full error preserved in ValidReason for --output json.
	t.Run("401 -> invalid", func(t *testing.T) {
		s := statusServer(t, http.StatusUnauthorized)
		p := loadFromHost(t, s.URL)
		assert.False(t, p.Valid)
		assert.NotEmpty(t, p.ValidReason)
	})

	t.Run("403 -> invalid", func(t *testing.T) {
		s := statusServer(t, http.StatusForbidden)
		p := loadFromHost(t, s.URL)
		assert.False(t, p.Valid)
		assert.NotEmpty(t, p.ValidReason)
	})

	t.Run("500 -> invalid", func(t *testing.T) {
		s := statusServer(t, http.StatusInternalServerError)
		p := loadFromHost(t, s.URL)
		assert.False(t, p.Valid)
		assert.NotEmpty(t, p.ValidReason)
	})

	t.Run("InvalidConfig -> invalid", func(t *testing.T) {
		// Host metadata reporting host_type=unified forces HostType=UnifiedHost.
		// Without an account_id (or a SPOG-shaped DiscoveryURL), ResolveConfigType
		// can't pick a side and falls through to InvalidConfig.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/databricks-config" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"host_type": "unified"})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(server.Close)

		dir := t.TempDir()
		configFile := filepath.Join(dir, ".databrickscfg")
		t.Setenv("HOME", dir)
		if runtime.GOOS == "windows" {
			t.Setenv("USERPROFILE", dir)
		}
		content := "[bad]\nhost = " + server.URL + "\ntoken = test-token\n"
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0o600))

		p := &profileMetadata{Name: "bad", Host: server.URL}
		p.Load(t.Context(), configFile, false)
		assert.False(t, p.Valid)
		assert.Contains(t, p.ValidReason, "fields conflict")
	})

	t.Run("skip-validate -> skipped", func(t *testing.T) {
		s := statusServer(t, http.StatusOK)
		p := loadFromHost(t, s.URL, withSkipValidate())
		assert.False(t, p.Valid)
		assert.Contains(t, p.ValidReason, "skip-validate")
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
