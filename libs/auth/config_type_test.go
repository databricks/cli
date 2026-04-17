package auth

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

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
			name: "workspace-scoped OIDC with account_id stays WorkspaceConfig",
			cfg: &config.Config{
				Host:         "https://workspace.databricks.com",
				AccountID:    "acct-123",
				DiscoveryURL: "https://workspace.databricks.com/oidc/.well-known/oauth-authorization-server",
			},
			want: config.WorkspaceConfig,
		},
		{
			name: "IsUnifiedHost fallback without discovery routes to AccountConfig",
			cfg: &config.Config{
				Host:                       "https://spog.databricks.com",
				AccountID:                  "acct-123",
				Experimental_IsUnifiedHost: true,
			},
			want: config.AccountConfig,
		},
		{
			name: "IsUnifiedHost fallback with workspace routes to WorkspaceConfig",
			cfg: &config.Config{
				Host:                       "https://spog.databricks.com",
				AccountID:                  "acct-123",
				WorkspaceID:                "ws-456",
				Experimental_IsUnifiedHost: true,
			},
			want: config.WorkspaceConfig,
		},
		{
			name: "no discovery and no IsUnifiedHost stays WorkspaceConfig",
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
