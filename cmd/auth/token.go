package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/credentials/oauth"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

func helpfulError(ctx context.Context, profile string, persistentAuth oauth.OAuthArgument) string {
	loginMsg := auth.BuildLoginCommand(ctx, profile, persistentAuth)
	return fmt.Sprintf("Try logging in again with `%s` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new", loginMsg)
}

func newTokenCommand(authArguments *auth.AuthArguments) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token [HOST]",
		Short: "Get authentication token",
	}

	var tokenTimeout time.Duration
	cmd.Flags().DurationVar(&tokenTimeout, "timeout", defaultTimeout,
		"Timeout for acquiring a token.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		profileName := ""
		profileFlag := cmd.Flag("profile")
		if profileFlag != nil {
			profileName = profileFlag.Value.String()
		}

		t, err := loadToken(ctx, loadTokenArgs{
			authArguments:      authArguments,
			profileName:        profileName,
			args:               args,
			tokenTimeout:       tokenTimeout,
			profiler:           profile.DefaultProfiler,
			persistentAuthOpts: nil,
		})
		if err != nil {
			return err
		}
		raw, err := json.MarshalIndent(t, "", "  ")
		if err != nil {
			return err
		}
		_, _ = cmd.OutOrStdout().Write(raw)
		return nil
	}

	return cmd
}

type loadTokenArgs struct {
	authArguments      *auth.AuthArguments
	profileName        string
	args               []string
	tokenTimeout       time.Duration
	profiler           profile.Profiler
	persistentAuthOpts []oauth.PersistentAuthOption
}

func loadToken(ctx context.Context, args loadTokenArgs) (*oauth2.Token, error) {
	// If a profile is provided we read the host from the .databrickscfg file
	if args.profileName != "" && len(args.args) > 0 {
		return nil, errors.New("providing both a profile and host is not supported")
	}

	err := setHostAndAccountId(ctx, args.profiler, args.profileName, args.authArguments, args.args)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, args.tokenTimeout)
	defer cancel()
	oauthArgument, err := args.authArguments.ToOAuthArgument()
	if err != nil {
		return nil, err
	}
	persistentAuth, err := oauth.NewPersistentAuth(ctx, args.persistentAuthOpts...)
	if err != nil {
		helpMsg := helpfulError(ctx, args.profileName, oauthArgument)
		return nil, fmt.Errorf("unexpected error creating persistent auth: %w. %s", err, helpMsg)
	}
	t, err := persistentAuth.Load(ctx, oauthArgument)
	if err != nil {
		helpMsg := helpfulError(ctx, args.profileName, oauthArgument)
		target := &oauth.InvalidRefreshTokenError{}
		if errors.As(err, &target) {
			return nil, auth.RewriteAuthError(ctx, args.authArguments.Host, args.authArguments.AccountId, args.profileName, err)
		}
		return nil, fmt.Errorf("unexpected error loading token: %w. %s", err, helpMsg)
	}
	return t, nil
}
