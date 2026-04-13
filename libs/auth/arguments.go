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

	if a.DiscoveryURL != "" {
		cfg.DiscoveryURL = a.DiscoveryURL
	} else if err := cfg.EnsureResolved(); err == nil {
		// EnsureResolved populates cfg.DiscoveryURL from .well-known.
	}

	host := cfg.CanonicalHostName()

	// Classic accounts.* hosts always use account OAuth.
	if strings.HasPrefix(host, "https://accounts.") || strings.HasPrefix(host, "https://accounts-dod.") {
		return u2m.NewProfileAccountOAuthArgument(host, cfg.AccountID, a.Profile)
	}

	// Pass a.AccountID (not cfg.AccountID) to avoid env var / discovery
	// back-fill from triggering SPOG routing for plain workspace hosts.
	if IsSPOG(cfg, a.AccountID) {
		return u2m.NewProfileUnifiedOAuthArgument(host, cfg.AccountID, a.Profile)
	}

	return u2m.NewProfileWorkspaceOAuthArgument(host, a.Profile)
}
