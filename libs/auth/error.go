package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/databricks/databricks-sdk-go/credentials/u2m"
)

// RewriteAuthError rewrites the error message for invalid refresh token error.
// It returns whether the error was rewritten and the rewritten error.
func RewriteAuthError(ctx context.Context, host, accountId, profile string, err error) (bool, error) {
	target := &u2m.InvalidRefreshTokenError{}
	if errors.As(err, &target) {
		oauthArgument, err := AuthArguments{host, accountId}.ToOAuthArgument()
		if err != nil {
			return false, err
		}
		msg := `A new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run the following command:
  $ ` + BuildLoginCommand(ctx, profile, oauthArgument)
		return true, errors.New(msg)
	}
	return false, err
}

// BuildLoginCommand builds the login command for the given OAuth argument or profile.
func BuildLoginCommand(ctx context.Context, profile string, arg u2m.OAuthArgument) string {
	cmd := []string{
		"databricks",
		"auth",
		"login",
	}
	if profile != "" {
		cmd = append(cmd, "--profile", profile)
	} else {
		switch arg := arg.(type) {
		case u2m.AccountOAuthArgument:
			cmd = append(cmd, "--host", arg.GetAccountHost(), "--account-id", arg.GetAccountId())
		case u2m.WorkspaceOAuthArgument:
			cmd = append(cmd, "--host", arg.GetWorkspaceHost())
		}
	}
	return strings.Join(cmd, " ")
}
