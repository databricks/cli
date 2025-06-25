package auth

import (
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
)

// AuthArguments is a struct that contains the common arguments passed to
// `databricks auth` commands.
type AuthArguments struct {
	Host      string
	AccountID string
}

// ToOAuthArgument converts the AuthArguments to an OAuthArgument from the Go SDK.
func (a AuthArguments) ToOAuthArgument() (u2m.OAuthArgument, error) {
	cfg := &config.Config{
		Host:      a.Host,
		AccountID: a.AccountID,
	}
	host := cfg.CanonicalHostName()
	if cfg.IsAccountClient() {
		return u2m.NewBasicAccountOAuthArgument(host, cfg.AccountID)
	}
	return u2m.NewBasicWorkspaceOAuthArgument(host)
}
