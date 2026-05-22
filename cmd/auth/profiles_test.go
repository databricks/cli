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
// The workspace endpoint deliberately returns 500 to mirror real SPOG hosts
// where account-audience OAuth tokens can't load workspace OAuth config.
// auth profiles probes both surfaces and accepts either success, so the test
// passes when Workspaces.List succeeds even though CurrentUser.Me fails —
// and a regression that drops the account probe surfaces as Valid=false.
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
			http.Error(w, "SPOG profiles must validate via Workspaces.List, not CurrentUser.Me", http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// newWorkspaceServer creates a mock workspace server that returns workspace-scoped
// OIDC and a workspace_id in discovery (mirroring real workspace hosts since
// PR #4809). It serves CurrentUser.Me; the account endpoint returns 404 so a
// workspace probe is the only path that produces Valid=true.
func newWorkspaceServer(t *testing.T, accountID string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/databricks-config":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"account_id":    accountID,
				"workspace_id":  "ws-from-discovery",
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
			// SPOG with a real workspace_id: workspace probe (CurrentUser.Me)
			// fails on the mock, account probe (Workspaces.List) succeeds —
			// the OR makes the profile valid.
			name:        "SPOG workspace profile valid when account probe succeeds",
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

// TestProfileLoadSPOGWorkspaceCredential covers the inverse of the
// account-OAuth case: a workspace-scoped credential (e.g. a PAT) against a
// SPOG host. CurrentUser.Me succeeds, Workspaces.List fails (no account-level
// access). The OR of the two probes must still mark the profile Valid=true.
func TestProfileLoadSPOGWorkspaceCredential(t *testing.T) {
	const accountID = "spog-acct"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/databricks-config":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"account_id":    accountID,
				"oidc_endpoint": r.Host + "/oidc/accounts/" + accountID,
			})
		case "/api/2.0/preview/scim/v2/Me":
			_ = json.NewEncoder(w).Encode(map[string]any{"userName": "test-user"})
		case "/api/2.0/accounts/" + accountID + "/workspaces":
			http.Error(w, "user lacks account-level access", http.StatusForbidden)
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

	content := "[ws-cred-on-spog]\nhost = " + server.URL + "\ntoken = test-token\naccount_id = " + accountID + "\nworkspace_id = ws-123\n"
	require.NoError(t, os.WriteFile(configFile, []byte(content), 0o600))

	p := &profileMetadata{
		Name:      "ws-cred-on-spog",
		Host:      server.URL,
		AccountID: accountID,
	}
	p.Load(t.Context(), configFile, false)

	assert.True(t, p.Valid, "workspace probe alone should make the profile valid")
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
	// account_id can linger in a profile from a prior account login on the
	// same profile name (e.g. user logged into accounts.cloud.databricks.com,
	// logged out, then re-used the profile name for a workspace login). A
	// stale account_id must not promote the profile to account validation
	// when the host itself isn't an account/SPOG surface — the workspace
	// probe is still the right signal.
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
