package auth

import (
	"strings"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
)

// WorkspaceIDNone is a sentinel value persisted to .databrickscfg when the
// user explicitly skips workspace selection for SPOG account-level access.
const WorkspaceIDNone = "none"

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

	// DiscoveryURL is cached from host metadata discovery to avoid duplicate
	// network calls when both runHostDiscovery and ToOAuthArgument need it.
	DiscoveryURL string
}

// ToOAuthArgument converts the AuthArguments to an OAuthArgument from the Go SDK.
// It calls EnsureResolved() to run host metadata discovery and routes based on
// the resolved DiscoveryURL rather than the Experimental_IsUnifiedHost flag.
func (a AuthArguments) ToOAuthArgument() (u2m.OAuthArgument, error) {
	// Strip the "none" sentinel so it is never passed to the SDK.
	workspaceID := a.WorkspaceID
	if workspaceID == WorkspaceIDNone {
		workspaceID = ""
	}

	cfg := &config.Config{
		Host:                       a.Host,
		AccountID:                  a.AccountID,
		WorkspaceID:                workspaceID,
		Experimental_IsUnifiedHost: a.IsUnifiedHost,
		HTTPTimeoutSeconds:         5,
		// Skip config file loading. We only want host metadata resolution
		// based on the explicit fields provided.
		Loaders: []config.Loader{config.ConfigAttributes},
	}

	discoveryURL := a.DiscoveryURL
	if discoveryURL == "" {
		// No cached discovery, resolve fresh.
		if err := cfg.EnsureResolved(); err == nil {
			discoveryURL = cfg.DiscoveryURL
		}
	}

	host := cfg.CanonicalHostName()

	// Classic accounts.* hosts always use account OAuth, even if discovery
	// returned data. SPOG/unified hosts are handled below via discovery or
	// the IsUnifiedHost flag.
	if strings.HasPrefix(host, "https://accounts.") || strings.HasPrefix(host, "https://accounts-dod.") {
		return u2m.NewProfileAccountOAuthArgument(host, cfg.AccountID, a.Profile)
	}

	// Route based on discovery data: a non-accounts host with an account-scoped
	// OIDC endpoint is a SPOG/unified host. We check a.AccountID (the caller-
	// provided value) rather than cfg.AccountID to avoid env var contamination
	// (e.g. DATABRICKS_ACCOUNT_ID set in the environment). We also require the
	// DiscoveryURL to contain "/oidc/accounts/" to distinguish SPOG hosts from
	// classic workspace hosts that may also return discovery metadata.
	if a.AccountID != "" && discoveryURL != "" && strings.Contains(discoveryURL, "/oidc/accounts/") {
		return u2m.NewProfileUnifiedOAuthArgument(host, cfg.AccountID, a.Profile)
	}

	// Legacy backward compat: existing profiles with IsUnifiedHost flag.
	if a.IsUnifiedHost && a.AccountID != "" {
		return u2m.NewProfileUnifiedOAuthArgument(host, cfg.AccountID, a.Profile)
	}

	return u2m.NewProfileWorkspaceOAuthArgument(host, a.Profile)
}
