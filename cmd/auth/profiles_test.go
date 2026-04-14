package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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
		wantValid   bool
	}{
		{
			name:      "SPOG account profile validated as account",
			host:      spogServer.URL,
			accountID: "spog-acct",
			wantValid: true,
		},
		{
			name:        "SPOG workspace profile validated as workspace",
			host:        spogServer.URL,
			accountID:   "spog-acct",
			workspaceID: "ws-123",
			wantValid:   true,
		},
		{
			name:        "SPOG profile with workspace_id=none validated as account",
			host:        spogServer.URL,
			accountID:   "spog-acct",
			workspaceID: "none",
			wantValid:   true,
		},
		{
			name:      "classic workspace with account_id from discovery stays workspace",
			host:      wsServer.URL,
			accountID: "ws-acct",
			wantValid: true,
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

			assert.Equal(t, tc.wantValid, p.Valid, "Valid mismatch")
			assert.NotEmpty(t, p.Host, "Host should be set")
			assert.NotEmpty(t, p.AuthType, "AuthType should be set")
		})
	}
}

func TestProfileLoadUnifiedHostFallback(t *testing.T) {
	// When Experimental_IsUnifiedHost is set but .well-known is unreachable,
	// ConfigType() returns InvalidConfig. The fallback should reclassify as
	// AccountConfig so the profile is still validated.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/databricks-config":
			w.WriteHeader(http.StatusNotFound)
		case "/api/2.0/accounts/unified-acct/workspaces":
			_ = json.NewEncoder(w).Encode([]map[string]any{})
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

	content := "[unified-profile]\nhost = " + server.URL + "\ntoken = test-token\naccount_id = unified-acct\nexperimental_is_unified_host = true\n"
	require.NoError(t, os.WriteFile(configFile, []byte(content), 0o600))

	p := &profileMetadata{
		Name:      "unified-profile",
		Host:      server.URL,
		AccountID: "unified-acct",
	}
	p.Load(t.Context(), configFile, false)

	assert.True(t, p.Valid, "unified host profile should be valid via fallback")
	assert.NotEmpty(t, p.Host)
	assert.NotEmpty(t, p.AuthType)
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
	// When .well-known returns 404 and Experimental_IsUnifiedHost is false,
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
