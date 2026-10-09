package auth

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasUnifiedHostSignal(t *testing.T) {
	cases := []struct {
		name         string
		discoveryURL string
		want         bool
	}{
		{name: "no signal", want: false},
		{name: "account-scoped OIDC", discoveryURL: "https://spog.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server", want: true},
		{name: "workspace-scoped OIDC", discoveryURL: "https://workspace.databricks.com/oidc/.well-known/oauth-authorization-server", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, HasUnifiedHostSignal(tc.discoveryURL))
		})
	}
}

func TestResolveConfigType(t *testing.T) {
	cases := []struct {
		name string
		cfg  *config.Config
		want config.ConfigType
	}{
		{
			name: "classic accounts host stays AccountConfig",
			cfg: &config.Config{
				Host:      "https://accounts.cloud.databricks.com",
				AccountID: "acct-123",
			},
			want: config.AccountConfig,
		},
		{
			name: "SPOG account-scoped OIDC without workspace routes to AccountConfig",
			cfg: &config.Config{
				Host:         "https://spog.databricks.com",
				AccountID:    "acct-123",
				DiscoveryURL: "https://spog.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server",
			},
			want: config.AccountConfig,
		},
		{
			name: "SPOG account-scoped OIDC with workspace routes to WorkspaceConfig",
			cfg: &config.Config{
				Host:         "https://spog.databricks.com",
				AccountID:    "acct-123",
				WorkspaceID:  "ws-456",
				DiscoveryURL: "https://spog.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server",
			},
			want: config.WorkspaceConfig,
		},
		{
			name: "SPOG account-scoped OIDC with workspace_id=none routes to AccountConfig",
			cfg: &config.Config{
				Host:         "https://spog.databricks.com",
				AccountID:    "acct-123",
				WorkspaceID:  "none",
				DiscoveryURL: "https://spog.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server",
			},
			want: config.AccountConfig,
		},
		{
			name: "SPOG workspace-scoped OIDC routes to WorkspaceConfig",
			cfg: &config.Config{
				Host:         "https://spog.databricks.com",
				AccountID:    "acct-123",
				WorkspaceID:  "ws-456",
				DiscoveryURL: "https://spog.databricks.com/oidc/.well-known/oauth-authorization-server?o=ws-456",
			},
			want: config.WorkspaceConfig,
		},
		{
			name: "workspace-scoped OIDC with account_id stays WorkspaceConfig",
			cfg: &config.Config{
				Host:         "https://workspace.databricks.com",
				AccountID:    "acct-123",
				DiscoveryURL: "https://workspace.databricks.com/oidc/.well-known/oauth-authorization-server",
			},
			want: config.WorkspaceConfig,
		},
		{
			name: "no discovery stays WorkspaceConfig",
			cfg: &config.Config{
				Host:      "https://workspace.databricks.com",
				AccountID: "acct-123",
			},
			want: config.WorkspaceConfig,
		},
		{
			name: "plain workspace without account_id",
			cfg: &config.Config{
				Host: "https://workspace.databricks.com",
			},
			want: config.WorkspaceConfig,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveConfigType(tc.cfg)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveConfigType_UnifiedHostMetadata(t *testing.T) {
	cases := []struct {
		name         string
		discoveryURL string
	}{
		{"workspace-scoped SPOG profile", "https://spog.databricks.com/oidc/.well-known/oauth-authorization-server?o=ws-456"},
		{"account-level SPOG profile with a workspace", "https://spog.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				Host:         "https://spog.databricks.com",
				AccountID:    "acct-123",
				WorkspaceID:  "ws-456",
				Token:        "token",
				DiscoveryURL: tc.discoveryURL,
				Loaders:      []config.Loader{config.ConfigAttributes},
				HostMetadataResolver: func(context.Context, string) (*config.HostMetadata, error) {
					return &config.HostMetadata{
						AccountID:    "acct-123",
						HostType:     config.UnifiedHost,
						OIDCEndpoint: "https://spog.databricks.com/oidc/accounts/{account_id}",
					}, nil
				},
			}
			require.NoError(t, cfg.EnsureResolved())
			assert.Equal(t, config.WorkspaceConfig, ResolveConfigType(cfg))
		})
	}
}

func TestSpogWorkspaceDiscoveryURL(t *testing.T) {
	assert.Equal(t,
		"https://acme.databricks.test/oidc/.well-known/oauth-authorization-server?o=123",
		SpogWorkspaceDiscoveryURL("https://acme.databricks.test/", "123"))
}

func TestIsSpogWorkspaceDiscoveryURL(t *testing.T) {
	tests := []struct {
		name         string
		discoveryURL string
		want         bool
	}{
		{"spog workspace", "https://acme.databricks.test/oidc/.well-known/oauth-authorization-server?o=123", true},
		{"canonical workspace", "https://dbc-123.cloud.databricks.test/oidc/.well-known/oauth-authorization-server", false},
		{"account scoped", "https://acme.databricks.test/oidc/accounts/abc/.well-known/oauth-authorization-server", false},
		{"other path with o", "https://acme.databricks.test/oidc/accounts/abc/.well-known/oauth-authorization-server?o=123", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsSpogWorkspaceDiscoveryURL(tt.discoveryURL))
		})
	}
}

func TestSpogWorkspaceIDFromDiscoveryURL(t *testing.T) {
	assert.Equal(t, "123", SpogWorkspaceIDFromDiscoveryURL(SpogWorkspaceDiscoveryURL("https://acme.databricks.test", "123")))
	assert.Empty(t, SpogWorkspaceIDFromDiscoveryURL("https://acme.databricks.test/oidc/accounts/abc/.well-known/oauth-authorization-server"))
	assert.Empty(t, SpogWorkspaceIDFromDiscoveryURL(""))
}
