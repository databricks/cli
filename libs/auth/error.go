package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/oauth"
)

// RewriteAuthError rewrites the error message for invalid refresh token error.
// It returns the rewritten error and a boolean indicating whether the error was rewritten.
func RewriteAuthError(ctx context.Context, host, accountId, profile string, err error) (error, bool) {
	target := &oauth.InvalidRefreshTokenError{}
	if errors.As(err, &target) {
		oauthArgument, err := AuthArguments{host, accountId}.ToOAuthArgument()
		if err != nil {
			return err, false
		}
		msg := "a new access token could not be retrieved because the refresh token is invalid."
		msg += fmt.Sprintf(" To reauthenticate, run `%s`", BuildLoginCommand(ctx, profile, oauthArgument))
		return errors.New(msg), true
	}
	return err, false
}

func BuildLoginCommand(ctx context.Context, profile string, arg oauth.OAuthArgument) string {
	cmd := []string{
		"databricks",
		"auth",
		"login",
	}
	if profile != "" {
		cmd = append(cmd, "--profile", profile)
	} else {
		switch arg := arg.(type) {
		case oauth.AccountOAuthArgument:
			cmd = append(cmd, "--host", arg.GetAccountHost(ctx), "--account-id", arg.GetAccountId(ctx))
		case oauth.WorkspaceOAuthArgument:
			cmd = append(cmd, "--host", arg.GetWorkspaceHost(ctx))
		}
	}
	return strings.Join(cmd, " ")
}

type AuthArguments struct {
	Host      string
	AccountId string
}

func (a AuthArguments) ToOAuthArgument() (oauth.OAuthArgument, error) {
	cfg := &config.Config{
		Host:      a.Host,
		AccountID: a.AccountId,
	}
	if cfg.IsAccountClient() {
		return oauth.NewBasicAccountOAuthArgument(cfg.Host, cfg.AccountID)
	}
	return oauth.NewBasicWorkspaceOAuthArgument(cfg.Host)
}
