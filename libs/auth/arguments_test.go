package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToOAuthArgument(t *testing.T) {
	tests := []struct {
		name         string
		args         AuthArguments
		wantHost     string
		wantCacheKey string
		wantError    bool
	}{
		{
			name: "workspace with no scheme",
			args: AuthArguments{
				Host: "my-workspace.cloud.databricks.com",
			},
			wantHost:     "https://my-workspace.cloud.databricks.com",
			wantCacheKey: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "workspace with https",
			args: AuthArguments{
				Host: "https://my-workspace.cloud.databricks.com",
			},
			wantHost:     "https://my-workspace.cloud.databricks.com",
			wantCacheKey: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "workspace with profile uses profile-based cache key",
			args: AuthArguments{
				Host:    "https://my-workspace.cloud.databricks.com",
				Profile: "my-profile",
			},
			wantHost:     "https://my-workspace.cloud.databricks.com",
			wantCacheKey: "my-profile",
		},
		{
			name: "account with no scheme",
			args: AuthArguments{
				Host:      "accounts.cloud.databricks.com",
				AccountID: "123456789",
			},
			wantHost:     "https://accounts.cloud.databricks.com",
			wantCacheKey: "https://accounts.cloud.databricks.com/oidc/accounts/123456789",
		},
		{
			name: "account with https",
			args: AuthArguments{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "123456789",
			},
			wantHost:     "https://accounts.cloud.databricks.com",
			wantCacheKey: "https://accounts.cloud.databricks.com/oidc/accounts/123456789",
		},
		{
			name: "account with profile uses profile-based cache key",
			args: AuthArguments{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "123456789",
				Profile:   "my-account-profile",
			},
			wantHost:     "https://accounts.cloud.databricks.com",
			wantCacheKey: "my-account-profile",
		},
		{
			name: "workspace with query parameter",
			args: AuthArguments{
				Host: "https://my-workspace.cloud.databricks.com?o=123456789",
			},
			wantHost:     "https://my-workspace.cloud.databricks.com",
			wantCacheKey: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "workspace with query parameter and path",
			args: AuthArguments{
				Host: "https://my-workspace.cloud.databricks.com/path?o=123456789",
			},
			wantHost:     "https://my-workspace.cloud.databricks.com",
			wantCacheKey: "https://my-workspace.cloud.databricks.com",
		},
		{
			name: "unified host with account ID only",
			args: AuthArguments{
				Host:          "https://unified.cloud.databricks.com",
				AccountID:     "123456789",
				IsUnifiedHost: true,
			},
			wantHost:     "https://unified.cloud.databricks.com",
			wantCacheKey: "https://unified.cloud.databricks.com/oidc/accounts/123456789",
		},
		{
			name: "unified host with both account ID and workspace ID",
			args: AuthArguments{
				Host:          "https://unified.cloud.databricks.com",
				AccountID:     "123456789",
				WorkspaceID:   "123456789",
				IsUnifiedHost: true,
			},
			wantHost:     "https://unified.cloud.databricks.com",
			wantCacheKey: "https://unified.cloud.databricks.com/oidc/accounts/123456789",
		},
		{
			name: "unified host with profile uses profile-based cache key",
			args: AuthArguments{
				Host:          "https://unified.cloud.databricks.com",
				AccountID:     "123456789",
				IsUnifiedHost: true,
				Profile:       "my-unified-profile",
			},
			wantHost:     "https://unified.cloud.databricks.com",
			wantCacheKey: "my-unified-profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.ToOAuthArgument()
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCacheKey, got.GetCacheKey())

			// Check if we got the right type of argument and verify the hostname
			if tt.args.IsUnifiedHost {
				arg, ok := got.(u2m.UnifiedOAuthArgument)
				assert.True(t, ok, "expected UnifiedOAuthArgument for unified host")
				assert.Equal(t, tt.wantHost, arg.GetHost())
			} else if tt.args.AccountID != "" {
				arg, ok := got.(u2m.AccountOAuthArgument)
				assert.True(t, ok, "expected AccountOAuthArgument for account host")
				assert.Equal(t, tt.wantHost, arg.GetAccountHost())
			} else {
				arg, ok := got.(u2m.WorkspaceOAuthArgument)
				assert.True(t, ok, "expected WorkspaceOAuthArgument for workspace host")
				assert.Equal(t, tt.wantHost, arg.GetWorkspaceHost())
			}
		})
	}
}

func TestToOAuthArgument_SPOGHostRoutesToUnified(t *testing.T) {
	// A SPOG host returns an account-scoped OIDC endpoint from discovery.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(map[string]any{
				"account_id":    "spog-account",
				"workspace_id":  "spog-ws",
				"oidc_endpoint": r.Host + "/oidc/accounts/spog-account",
			})
			require.NoError(t, err)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	args := AuthArguments{
		Host:      server.URL,
		AccountID: "spog-account",
	}
	got, err := args.ToOAuthArgument()
	require.NoError(t, err)

	// Should route to unified OAuth.
	_, ok := got.(u2m.UnifiedOAuthArgument)
	assert.True(t, ok, "expected UnifiedOAuthArgument for SPOG host, got %T", got)
}

func TestToOAuthArgument_ClassicWorkspaceNotMisrouted(t *testing.T) {
	// A classic workspace host returns workspace-scoped OIDC (no /accounts/ in path).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(map[string]any{
				"workspace_id":  "12345",
				"oidc_endpoint": r.Host + "/oidc",
			})
			require.NoError(t, err)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Even with AccountID set (from env or caller), a classic workspace host
	// should NOT be routed to unified OAuth.
	args := AuthArguments{
		Host:      server.URL,
		AccountID: "some-account",
	}
	got, err := args.ToOAuthArgument()
	require.NoError(t, err)

	// Should route to workspace OAuth, not unified.
	_, ok := got.(u2m.WorkspaceOAuthArgument)
	assert.True(t, ok, "expected WorkspaceOAuthArgument for classic workspace, got %T", got)
}

func TestToOAuthArgument_NoAccountIDSkipsUnifiedRouting(t *testing.T) {
	// Even if discovery returns an account-scoped OIDC URL, without an explicit
	// AccountID from the caller, unified routing should NOT be triggered.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(map[string]any{
				"account_id":    "discovered-account",
				"oidc_endpoint": r.Host + "/oidc/accounts/discovered-account",
			})
			require.NoError(t, err)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	args := AuthArguments{
		Host: server.URL,
		// No AccountID set by caller.
	}
	got, err := args.ToOAuthArgument()
	require.NoError(t, err)

	// Should route to workspace OAuth because caller didn't provide AccountID.
	_, ok := got.(u2m.WorkspaceOAuthArgument)
	assert.True(t, ok, "expected WorkspaceOAuthArgument when no caller AccountID, got %T", got)
}

func TestIsAccountsHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"https://accounts.cloud.databricks.com", true},
		{"https://accounts-dod.cloud.databricks.us", true},
		{"accounts.cloud.databricks.com", true},
		{"accounts-dod.cloud.databricks.us", true},
		{"https://my-workspace.cloud.databricks.com", false},
		{"https://spog.example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			assert.Equal(t, tt.want, IsAccountsHost(tt.host))
		})
	}
}
