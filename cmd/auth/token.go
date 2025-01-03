package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/credentials/oauth"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type tokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func buildLoginCommand(ctx context.Context, profile string, persistentAuth oauth.OAuthArgument) string {
	executable := os.Args[0]
	cmd := []string{
		executable,
		"auth",
		"login",
	}
	if profile != "" {
		cmd = append(cmd, "--profile", profile)
	} else {
		cmd = append(cmd, "--host", persistentAuth.GetHost(ctx))
		if accountId := persistentAuth.GetAccountId(ctx); accountId != "" {
			cmd = append(cmd, "--account-id", accountId)
		}
	}
	return strings.Join(cmd, " ")
}

func helpfulError(ctx context.Context, profile string, persistentAuth oauth.OAuthArgument) string {
	loginMsg := buildLoginCommand(ctx, profile, persistentAuth)
	return fmt.Sprintf("Try logging in again with `%s` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new", loginMsg)
}

func newTokenCommand(oauthArgument oauth.OAuthArgument) *cobra.Command {
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
			oauthArgument:      oauthArgument,
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
	oauthArgument      oauth.OAuthArgument
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

	oauthArgument, err := setHostAndAccountId(ctx, args.profiler, args.profileName, args.oauthArgument, args.args)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, args.tokenTimeout)
	defer cancel()
	persistentAuth, err := oauth.NewPersistentAuth(ctx)
	if err != nil {
		helpMsg := helpfulError(ctx, args.profileName, oauthArgument)
		return nil, fmt.Errorf("unexpected error creating persistent auth: %w. %s", err, helpMsg)
	}
	t, err := persistentAuth.Load(ctx, oauthArgument)
	var httpErr *httpclient.HttpError
	if errors.As(err, &httpErr) {
		helpMsg := helpfulError(ctx, args.profileName, oauthArgument)
		t := &tokenErrorResponse{}
		err = json.Unmarshal([]byte(httpErr.Message), t)
		if err != nil {
			return nil, fmt.Errorf("unexpected parsing token response: %w. %s", err, helpMsg)
		}
		if t.ErrorDescription == "Refresh token is invalid" {
			return nil, fmt.Errorf("a new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run `%s`", buildLoginCommand(ctx, args.profileName, oauthArgument))
		} else {
			return nil, fmt.Errorf("unexpected error refreshing token: %s. %s", t.ErrorDescription, helpMsg)
		}
	} else if err != nil {
		return nil, fmt.Errorf("unexpected error refreshing token: %w. %s", err, helpfulError(ctx, args.profileName, oauthArgument))
	}
	return t, nil
}
