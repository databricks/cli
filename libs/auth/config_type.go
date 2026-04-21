package auth

import (
	"strings"

	"github.com/databricks/databricks-sdk-go/config"
)

// HasUnifiedHostSignal reports whether a host has been identified as unified,
// either by a resolved DiscoveryURL pointing at an account-scoped OIDC endpoint
// or by the caller-provided legacy fallback. Extracted so callers that don't
// (yet) have an account ID can check the signal without tripping IsSPOG's guard.
//
// fallback replaces a former read of cfg.Experimental_IsUnifiedHost. Callers
// thread the CLI-side signal in (e.g. AuthArguments.IsUnifiedHost,
// Profile.IsUnifiedHost) because the SDK field is being removed.
func HasUnifiedHostSignal(discoveryURL string, fallback bool) bool {
	if discoveryURL != "" && strings.Contains(discoveryURL, "/oidc/accounts/") {
		return true
	}
	return fallback
}

// IsSPOG returns true if the config represents a SPOG (Single Pane of Glass)
// host with account-scoped OIDC. Detection layers HasUnifiedHostSignal on top
// of an accountID guard: SPOG routing requires an account ID to construct the
// OAuth URL, so a nil or empty accountID always returns false.
//
// The accountID parameter is separate from cfg.AccountID so that callers can
// control the source: ResolveConfigType passes cfg.AccountID (from config file),
// while ToOAuthArgument passes the caller-provided value to avoid env var
// contamination (DATABRICKS_ACCOUNT_ID or .well-known back-fill).
func IsSPOG(cfg *config.Config, accountID string, unifiedHostFallback bool) bool {
	if accountID == "" {
		return false
	}
	return HasUnifiedHostSignal(cfg.DiscoveryURL, unifiedHostFallback)
}

// ResolveConfigType determines the effective ConfigType for a resolved config.
// The SDK's ConfigType() classifies based on the host URL prefix alone, which
// misclassifies SPOG hosts (they don't match the accounts.* prefix). This
// function additionally uses IsSPOG to detect SPOG hosts.
//
// The cfg must already be resolved (via EnsureResolved) before calling this.
// unifiedHostFallback is threaded through to IsSPOG; see its docstring.
func ResolveConfigType(cfg *config.Config, unifiedHostFallback bool) config.ConfigType {
	configType := cfg.ConfigType()
	if configType == config.AccountConfig {
		return configType
	}

	if !IsSPOG(cfg, cfg.AccountID, unifiedHostFallback) {
		return configType
	}

	// The WorkspaceConfig return is a no-op when configType is already
	// WorkspaceConfig, but is needed for InvalidConfig (legacy unified-host
	// profiles where the SDK dropped the UnifiedHost case in v0.126.0).
	if cfg.WorkspaceID != "" && cfg.WorkspaceID != WorkspaceIDNone {
		return config.WorkspaceConfig
	}
	return config.AccountConfig
}
