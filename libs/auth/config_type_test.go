package auth

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
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

func TestIsSPOG(t *testing.T) {
	cases := []struct {
		name      string
		cfg       *config.Config
		accountID string
		want      bool
	}{
		{
			name:      "account-scoped OIDC with account_id",
			cfg:       &config.Config{DiscoveryURL: "https://spog.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server"},
			accountID: "acct-123",
			want:      true,
		},
		{
			name:      "account-scoped OIDC without account_id",
			cfg:       &config.Config{DiscoveryURL: "https://spog.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server"},
			accountID: "",
			want:      false,
		},
		{
			name:      "workspace-scoped OIDC with account_id back-filled",
			cfg:       &config.Config{DiscoveryURL: "https://workspace.databricks.com/oidc/.well-known/oauth-authorization-server"},
			accountID: "acct-123",
			want:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsSPOG(tc.cfg, tc.accountID))
		})
	}
}

// Configs used across the host-classification tests below. The three host
// shapes are mutually exclusive: exactly one helper returns true per cfg.
// accounts-dod.* is a second classic accounts variant — same OIDC shape
// and classification as accounts.*, different URL prefix.
var (
	classicAccountCfg = &config.Config{
		Host:         "https://accounts.cloud.databricks.com",
		AccountID:    "acct-123",
		DiscoveryURL: "https://accounts.cloud.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server",
	}
	classicAccountDodCfg = &config.Config{
		Host:         "https://accounts-dod.cloud.databricks.us",
		AccountID:    "acct-123",
		DiscoveryURL: "https://accounts-dod.cloud.databricks.us/oidc/accounts/acct-123/.well-known/oauth-authorization-server",
	}
	spogCfg = &config.Config{
		Host:         "https://spog.gcp.databricks.com",
		AccountID:    "acct-123",
		DiscoveryURL: "https://spog.gcp.databricks.com/oidc/accounts/acct-123/.well-known/oauth-authorization-server",
	}
	classicWorkspaceCfg = &config.Config{
		Host:         "https://dbc-xxxx.cloud.databricks.com",
		AccountID:    "acct-123",
		DiscoveryURL: "https://dbc-xxxx.cloud.databricks.com/oidc/.well-known/oauth-authorization-server",
	}
)

func TestIsSPOGHost(t *testing.T) {
	assert.False(t, IsSPOGHost(classicAccountCfg), "classic accounts.* shares the SPOG OIDC shape but is not SPOG")
	assert.False(t, IsSPOGHost(classicAccountDodCfg), "classic accounts-dod.* shares the SPOG OIDC shape but is not SPOG")
	assert.True(t, IsSPOGHost(spogCfg))
	assert.False(t, IsSPOGHost(classicWorkspaceCfg))
}

func TestIsClassicWorkspaceHost(t *testing.T) {
	assert.False(t, IsClassicWorkspaceHost(classicAccountCfg))
	assert.False(t, IsClassicWorkspaceHost(classicAccountDodCfg))
	assert.False(t, IsClassicWorkspaceHost(spogCfg))
	assert.True(t, IsClassicWorkspaceHost(classicWorkspaceCfg))
}
