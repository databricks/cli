package auth

import (
	"fmt"

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
func (a AuthArguments) ToOAuthArgument() (u2m.OAuthArgument, error) {
	cfg := &config.Config{
		Host:                       a.Host,
		AccountID:                  a.AccountID,
		WorkspaceID:                a.WorkspaceID,
		Experimental_IsUnifiedHost: a.IsUnifiedHost,
	}
	host := cfg.CanonicalHostName()

	switch cfg.HostType() {
	case config.AccountHost:
		return u2m.NewProfileAccountOAuthArgument(host, cfg.AccountID, a.Profile)
	case config.WorkspaceHost:
		return u2m.NewProfileWorkspaceOAuthArgument(host, a.Profile)
	case config.UnifiedHost:
		// For unified hosts, always use the unified OAuth argument with account ID.
		// The workspace ID is stored in the config for API routing, not OAuth.
		return u2m.NewProfileUnifiedOAuthArgument(host, cfg.AccountID, a.Profile)
	default:
		return nil, fmt.Errorf("unknown host type: %v", cfg.HostType())
	}
}
