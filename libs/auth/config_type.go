package auth

import (
	"strings"

	"github.com/databricks/databricks-sdk-go/config"
)

// IsSPOG returns true if the config represents a SPOG (Single Pane of Glass)
// host with account-scoped OIDC. Detection is based on:
//  1. The resolved DiscoveryURL containing /oidc/accounts/ (from .well-known).
//  2. The Experimental_IsUnifiedHost flag as a legacy fallback.
//
// The accountID parameter is separate from cfg.AccountID so that callers can
// control the source: ResolveConfigType passes cfg.AccountID (from config file),
// while ToOAuthArgument passes the caller-provided value to avoid env var
// contamination (DATABRICKS_ACCOUNT_ID or .well-known back-fill).
func IsSPOG(cfg *config.Config, accountID string) bool {
	if accountID == "" {
		return false
	}
	if cfg.DiscoveryURL != "" && strings.Contains(cfg.DiscoveryURL, "/oidc/accounts/") {
		return true
	}
	return cfg.Experimental_IsUnifiedHost
}

// ResolveConfigType determines the effective ConfigType for a resolved config.
// The SDK's ConfigType() classifies based on the host URL prefix alone, which
// misclassifies SPOG hosts (they don't match the accounts.* prefix). This
// function additionally uses IsSPOG to detect SPOG hosts.
//
// The cfg must already be resolved (via EnsureResolved) before calling this.
func ResolveConfigType(cfg *config.Config) config.ConfigType {
	configType := cfg.ConfigType()
	if configType == config.AccountConfig {
		return configType
	}

	if !IsSPOG(cfg, cfg.AccountID) {
		return configType
	}

	if cfg.WorkspaceID != "" && cfg.WorkspaceID != WorkspaceIDNone {
		return config.WorkspaceConfig
	}
	return config.AccountConfig
}
