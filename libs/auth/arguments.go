package auth

import (
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/oauth"
)

// AuthArguments is a struct that contains the common arguments passed to
// `databricks auth` commands.
type AuthArguments struct {
	Host      string
	AccountID string
}

// ToOAuthArgument converts the AuthArguments to an OAuthArgument from the Go SDK.
func (a AuthArguments) ToOAuthArgument() (oauth.OAuthArgument, error) {
	cfg := &config.Config{
		Host:      a.Host,
		AccountID: a.AccountID,
	}
	if cfg.IsAccountClient() {
		return oauth.NewBasicAccountOAuthArgument(cfg.Host, cfg.AccountID)
	}
	return oauth.NewBasicWorkspaceOAuthArgument(cfg.Host)
}
