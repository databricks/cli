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
}

// ToOAuthArgument converts the AuthArguments to an OAuthArgument from the Go SDK.
func (a AuthArguments) ToOAuthArgument() (u2m.OAuthArgument, error) {
	cfg := &config.Config{
		Host:                       a.Host,
		AccountID:                  a.AccountID,
		WorkspaceId:                a.WorkspaceID,
		Experimental_IsUnifiedHost: a.IsUnifiedHost,
	}
	host := cfg.CanonicalHostName()

	switch cfg.HostType() {
	case config.AccountHost:
		return u2m.NewBasicAccountOAuthArgument(host, cfg.AccountID)
	case config.WorkspaceHost:
		return u2m.NewBasicWorkspaceOAuthArgument(host)
	case config.UnifiedHost:
		return u2m.NewBasicUnifiedOAuthArgument(host, cfg.AccountID)
	default:
		return nil, fmt.Errorf("unknown host type: %v", cfg.HostType())
	}
}
