package auth

import (
	"strings"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
)

// AuthArguments is a struct that contains the common arguments passed to
// `databricks auth` commands.
type AuthArguments struct {
	Host          string
	AccountID     string
	WorkspaceID   string
	IsUnifiedHost bool

	// Profile is the optional profile name. When set, the OAuth token cache
	// key is the profile name instead of the host-based key.
	Profile string
}

// ToOAuthArgument converts the AuthArguments to an OAuthArgument from the Go SDK.
// It calls EnsureResolved() to run host metadata discovery and routes based on
// the resolved DiscoveryURL rather than the Experimental_IsUnifiedHost flag.
func (a AuthArguments) ToOAuthArgument() (u2m.OAuthArgument, error) {
	cfg := &config.Config{
		Host:                       a.Host,
		AccountID:                  a.AccountID,
		WorkspaceID:                a.WorkspaceID,
		Experimental_IsUnifiedHost: a.IsUnifiedHost,
		HTTPTimeoutSeconds:         5,
		// Skip config file loading. We only want host metadata resolution
		// based on the explicit fields provided.
		Loaders: []config.Loader{config.ConfigAttributes},
	}

	// Ignore resolution errors; discovery failure is expected for non-SPOG
	// hosts and the function falls through to workspace OAuth below.
	_ = cfg.EnsureResolved()

	host := cfg.CanonicalHostName()

	// Classic accounts.* hosts always use account OAuth, even if discovery
	// returned data. This preserves backward compatibility.
	if IsAccountsHost(host) {
		return u2m.NewProfileAccountOAuthArgument(host, cfg.AccountID, a.Profile)
	}

	// Route based on discovery data: a non-accounts host with an account-scoped
	// OIDC endpoint is a SPOG/unified host. We check a.AccountID (the caller-
	// provided value) rather than cfg.AccountID to avoid env var contamination
	// (e.g. DATABRICKS_ACCOUNT_ID set in the environment). We also require the
	// DiscoveryURL to contain "/oidc/accounts/" to distinguish SPOG hosts from
	// classic workspace hosts that may also return discovery metadata.
	if a.AccountID != "" && cfg.DiscoveryURL != "" && strings.Contains(cfg.DiscoveryURL, "/oidc/accounts/") {
		return u2m.NewProfileUnifiedOAuthArgument(host, cfg.AccountID, a.Profile)
	}

	// Legacy backward compat: existing profiles with IsUnifiedHost flag.
	if a.IsUnifiedHost && a.AccountID != "" {
		return u2m.NewProfileUnifiedOAuthArgument(host, cfg.AccountID, a.Profile)
	}

	return u2m.NewProfileWorkspaceOAuthArgument(host, a.Profile)
}

// IsAccountsHost returns true if the host is a classic Databricks accounts host
// (e.g. https://accounts.cloud.databricks.com or https://accounts-dod.cloud.databricks.us).
func IsAccountsHost(host string) bool {
	h := normalizeHost(host)
	return strings.HasPrefix(h, "https://accounts.") ||
		strings.HasPrefix(h, "https://accounts-dod.")
}

// normalizeHost ensures the host has a scheme prefix.
func normalizeHost(host string) string {
	if host != "" && !strings.Contains(host, "://") {
		return "https://" + host
	}
	return host
}
