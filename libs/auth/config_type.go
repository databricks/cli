package auth

import (
	"net/url"
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
// The accountID parameter is separate from cfg.AccountID so that callers can
// control the source: ResolveConfigType passes cfg.AccountID (from config file),
// while ToOAuthArgument passes the caller-provided value to avoid env var
// contamination (DATABRICKS_ACCOUNT_ID or .well-known back-fill).
func IsSPOG(cfg *config.Config, accountID string) bool {
	if accountID == "" {
		return false
	}
	return HasUnifiedHostSignal(cfg.DiscoveryURL)
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

	// A workspace-scoped SPOG token only reaches its workspace.
	if IsSpogWorkspaceDiscoveryURL(cfg.DiscoveryURL) {
		return config.WorkspaceConfig
	}

	if !IsSPOG(cfg, cfg.AccountID) {
		return configType
	}

	if cfg.WorkspaceID != "" && cfg.WorkspaceID != WorkspaceIDNone {
		return config.WorkspaceConfig
	}
	return config.AccountConfig
}

// spogWorkspaceDiscoveryPath is the authorization server metadata path for
// workspace-level OAuth. On a SPOG host the workspace is selected with the
// o=<workspace_id> query parameter, and the served endpoints are those of the
// workspace's canonical host.
const spogWorkspaceDiscoveryPath = "/oidc/.well-known/oauth-authorization-server"

// SpogWorkspaceDiscoveryURL returns the workspace-level OAuth metadata URL for
// workspaceID on the SPOG host. Saved as a profile's discovery_url, it marks
// the profile as holding a workspace-scoped token: OAuth runs against the
// workspace's own endpoints while API calls go to the SPOG host.
func SpogWorkspaceDiscoveryURL(spogHost, workspaceID string) string {
	return strings.TrimSuffix(spogHost, "/") + spogWorkspaceDiscoveryPath + "?" + url.Values{"o": {workspaceID}}.Encode()
}

// IsSpogWorkspaceDiscoveryURL reports whether discoveryURL selects one
// workspace on a SPOG host, as produced by [SpogWorkspaceDiscoveryURL].
// Canonical workspace discovery URLs never carry the o= parameter.
func IsSpogWorkspaceDiscoveryURL(discoveryURL string) bool {
	u, err := url.Parse(discoveryURL)
	if err != nil {
		return false
	}
	return u.Path == spogWorkspaceDiscoveryPath && u.Query().Get("o") != ""
}

// SpogWorkspaceIDFromDiscoveryURL returns the workspace ID selected by a
// discovery URL produced by [SpogWorkspaceDiscoveryURL], or "" for any other URL.
func SpogWorkspaceIDFromDiscoveryURL(discoveryURL string) string {
	if !IsSpogWorkspaceDiscoveryURL(discoveryURL) {
		return ""
	}
	u, _ := url.Parse(discoveryURL)
	return u.Query().Get("o")
}
