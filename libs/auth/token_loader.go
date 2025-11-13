package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"golang.org/x/oauth2"
)

type persistentAuth interface {
	Token() (*oauth2.Token, error)
	Close() error
}

var persistentAuthFactory = func(ctx context.Context, opts ...u2m.PersistentAuthOption) (persistentAuth, error) {
	return u2m.NewPersistentAuth(ctx, opts...)
}

// AcquireTokenRequest describes the information needed to load or refresh a token
// from the persistent auth cache.
type AcquireTokenRequest struct {
	AuthArguments      *AuthArguments
	ProfileName        string
	Timeout            time.Duration
	PersistentAuthOpts []u2m.PersistentAuthOption
}

// AcquireToken obtains an OAuth token from the persistent auth cache, refreshing it if needed.
func AcquireToken(ctx context.Context, req AcquireTokenRequest) (*oauth2.Token, error) {
	if req.AuthArguments == nil {
		return nil, errors.New("auth arguments are required")
	}

	oauthArgument, err := req.AuthArguments.ToOAuthArgument()
	if err != nil {
		return nil, err
	}

	var cancel context.CancelFunc
	if req.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	allOpts := append([]u2m.PersistentAuthOption{}, req.PersistentAuthOpts...)
	allOpts = append(allOpts, u2m.WithOAuthArgument(oauthArgument))
	persistentAuth, err := persistentAuthFactory(ctx, allOpts...)
	if err != nil {
		return nil, fmt.Errorf("%w. %s", err, buildHelpfulLoginMessage(ctx, req.ProfileName, oauthArgument))
	}
	defer persistentAuth.Close()

	token, err := persistentAuth.Token()
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			// The error returned by the SDK when the token cache doesn't exist or doesn't contain a token
			// for the given host changed in SDK v0.77.0: https://github.com/databricks/databricks-sdk-go/pull/1250.
			// This was released as part of CLI v0.264.0.
			//
			// Older SDK versions check for a particular substring to determine if
			// the OAuth authentication type can fall through or if it is a real error.
			// This means we need to keep this error message constant for backwards compatibility.
			//
			// This is captured in an acceptance test under "cmd/auth/token".
			err = errors.New("cache: databricks OAuth is not configured for this host")
		}
		if rewritten, rewrittenErr := RewriteAuthError(ctx, req.AuthArguments.Host, req.AuthArguments.AccountID, req.ProfileName, err); rewritten {
			return nil, rewrittenErr
		}
		return nil, fmt.Errorf("%w. %s", err, buildHelpfulLoginMessage(ctx, req.ProfileName, oauthArgument))
	}

	return token, nil
}

func buildHelpfulLoginMessage(ctx context.Context, profile string, arg u2m.OAuthArgument) string {
	loginMsg := BuildLoginCommand(ctx, profile, arg)
	return fmt.Sprintf("Try logging in again with `%s` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new", loginMsg)
}
