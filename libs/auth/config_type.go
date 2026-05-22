package auth

import (
	"strings"

	"github.com/databricks/databricks-sdk-go/config"
)

// IsClassicAccountHost reports whether a host is a classic accounts.* host
// (account-level API access). Must be called with a canonicalized host; see
// config.Config.CanonicalHostName.
func IsClassicAccountHost(canonicalHost string) bool {
	return strings.HasPrefix(canonicalHost, "https://accounts.") ||
		strings.HasPrefix(canonicalHost, "https://accounts-dod.")
}

// HasUnifiedHostSignal reports whether a host has been identified as unified,
// based on a resolved DiscoveryURL pointing at an account-scoped OIDC endpoint.
// Extracted so callers that don't (yet) have an account ID can check the signal
// without tripping IsSPOG's guard.
func HasUnifiedHostSignal(discoveryURL string) bool {
	return discoveryURL != "" && strings.Contains(discoveryURL, "/oidc/accounts/")
}

// IsSPOG returns true if the config represents a SPOG (Single Pane of Glass)
// host with account-scoped OIDC. Detection layers HasUnifiedHostSignal on top
// of an accountID guard: SPOG routing requires an account ID to construct the
// OAuth URL, so a nil or empty accountID always returns false.
//
// The accountID parameter is separate from cfg.AccountID so that callers
// (currently ToOAuthArgument) can pass the caller-provided value to avoid
// env-var contamination (DATABRICKS_ACCOUNT_ID or .well-known back-fill)
// that would otherwise misroute plain workspace hosts as SPOG.
func IsSPOG(cfg *config.Config, accountID string) bool {
	if accountID == "" {
		return false
	}
	return HasUnifiedHostSignal(cfg.DiscoveryURL)
}

// IsSPOGHost reports whether cfg points at a unified SPOG host: account-scoped
// OIDC discovery and NOT a classic accounts.* host. Classic accounts.* hosts
// share the same OIDC shape, so IsSPOG alone can't tell them apart; layer
// IsClassicAccountHost on top to disambiguate.
func IsSPOGHost(cfg *config.Config) bool {
	if IsClassicAccountHost(cfg.CanonicalHostName()) {
		return false
	}
	return IsSPOG(cfg, cfg.AccountID)
}

// IsClassicWorkspaceHost reports whether cfg points at a classic workspace
// host: neither a classic accounts.* host nor a SPOG host.
func IsClassicWorkspaceHost(cfg *config.Config) bool {
	return !IsClassicAccountHost(cfg.CanonicalHostName()) && !IsSPOG(cfg, cfg.AccountID)
}
