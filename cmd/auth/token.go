package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

func helpfulError(ctx context.Context, profile string, persistentAuth u2m.OAuthArgument) string {
	loginMsg := auth.BuildLoginCommand(ctx, profile, persistentAuth)
	return fmt.Sprintf("Try logging in again with `%s` before retrying. If this fails, please report this issue to the Databricks CLI maintainers at https://github.com/databricks/cli/issues/new", loginMsg)
}

func newTokenCommand(authArguments *auth.AuthArguments) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token [HOST_OR_PROFILE]",
		Short: "Get authentication token",
		Long: `Get authentication token from the local cache in ~/.databricks/token-cache.json.
Refresh the access token if it is expired. Note: This command only works with
U2M authentication (using the 'databricks auth login' command). M2M authentication
using a client ID and secret is not supported.`,
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
	// authArguments is the parsed auth arguments, including the host and optionally the account ID.
	authArguments *auth.AuthArguments

	// profileName is the name of the specified profile. If no profile is specified, this is an empty string.
	profileName string

	// args is the list of arguments passed to the command.
	args []string

	// tokenTimeout is the timeout for retrieving (and potentially refreshing) an OAuth token.
	tokenTimeout time.Duration

	// profiler is the profiler to use for reading the host and account ID from the .databrickscfg file.
	profiler profile.Profiler

	// persistentAuthOpts are the options to pass to the persistent auth client.
	persistentAuthOpts []u2m.PersistentAuthOption
}

// loadToken loads an OAuth token from the persistent auth store. The host and account ID are read from
// the provided profiler if not explicitly provided. If the token cannot be refreshed, a helpful error message
// is printed to the user with steps to reauthenticate.
func loadToken(ctx context.Context, args loadTokenArgs) (*oauth2.Token, error) {
	// If a profile is provided we read the host from the .databrickscfg file
	if args.profileName != "" && len(args.args) > 0 {
		return nil, errors.New("providing both a profile and host is not supported")
	}

	// If no --profile flag, try resolving the positional arg as a profile name.
	// If it matches, use it. If not, fall through to host treatment.
	if args.profileName == "" && len(args.args) == 1 {
		candidateProfile, err := loadProfileByName(ctx, args.args[0], args.profiler)
		if err != nil {
			return nil, err
		}
		if candidateProfile != nil {
			args.profileName = args.args[0]
			args.args = nil
		}
	}

	existingProfile, err := loadProfileByName(ctx, args.profileName, args.profiler)
	if err != nil {
		return nil, err
	}

	// Load unified host flags from the profile if available
	if existingProfile != nil {
		if !args.authArguments.IsUnifiedHost && existingProfile.IsUnifiedHost {
			args.authArguments.IsUnifiedHost = existingProfile.IsUnifiedHost
		}
		if args.authArguments.WorkspaceID == "" && existingProfile.WorkspaceID != "" {
			args.authArguments.WorkspaceID = existingProfile.WorkspaceID
		}
	}

	err = setHostAndAccountId(ctx, existingProfile, args.authArguments, args.args)
	if err != nil {
		return nil, err
	}

	// When no profile was specified, check if multiple profiles match the
	// effective cache key for this host.
	if args.profileName == "" && args.authArguments.Host != "" {
		cfg := &config.Config{
			Host:                       args.authArguments.Host,
			AccountID:                  args.authArguments.AccountID,
			Experimental_IsUnifiedHost: args.authArguments.IsUnifiedHost,
		}
		// Canonicalize first so HostType() can correctly identify account hosts
		// even when the host string lacks a scheme (e.g. "accounts.cloud.databricks.com").
		cfg.CanonicalHostName()
		var matchFn profile.ProfileMatchFunction
		switch cfg.HostType() {
		case config.AccountHost, config.UnifiedHost:
			matchFn = profile.WithHostAndAccountID(args.authArguments.Host, args.authArguments.AccountID)
		default:
			matchFn = profile.WithHost(args.authArguments.Host)
		}

		matchingProfiles, err := args.profiler.LoadProfiles(ctx, matchFn)
		if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
			return nil, err
		}
		if len(matchingProfiles) > 1 {
			configPath, _ := args.profiler.GetPath(ctx)
			if configPath == "" {
				panic("configPath is empty but LoadProfiles returned multiple profiles")
			}
			if !cmdio.IsPromptSupported(ctx) {
				names := strings.Join(matchingProfiles.Names(), " and ")
				return nil, fmt.Errorf("%s match %s in %s. Use --profile to specify which profile to use",
					names, args.authArguments.Host, configPath)
			}
			selected, err := askForMatchingProfile(ctx, matchingProfiles, args.authArguments.Host)
			if err != nil {
				return nil, err
			}
			args.profileName = selected
		}
	}

	args.authArguments.Profile = args.profileName

	ctx, cancel := context.WithTimeout(ctx, args.tokenTimeout)
	defer cancel()
	oauthArgument, err := args.authArguments.ToOAuthArgument()
	if err != nil {
		return nil, err
	}
	allArgs := append(args.persistentAuthOpts, u2m.WithOAuthArgument(oauthArgument))
	persistentAuth, err := u2m.NewPersistentAuth(ctx, allArgs...)
	if err != nil {
		helpMsg := helpfulError(ctx, args.profileName, oauthArgument)
		return nil, fmt.Errorf("%w. %s", err, helpMsg)
	}
	t, err := persistentAuth.Token()
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
		if rewritten, rewrittenErr := auth.RewriteAuthError(ctx, args.authArguments.Host, args.authArguments.AccountID, args.profileName, err); rewritten {
			return nil, rewrittenErr
		}
		helpMsg := helpfulError(ctx, args.profileName, oauthArgument)
		return nil, fmt.Errorf("%w. %s", err, helpMsg)
	}
	return t, nil
}

func askForMatchingProfile(ctx context.Context, profiles profile.Profiles, host string) (string, error) {
	i, _, err := cmdio.RunSelect(ctx, &promptui.Select{
		Label:             "Multiple profiles match " + host,
		Items:             profiles,
		Searcher:          profiles.SearchCaseInsensitive,
		StartInSearchMode: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | faint }}",
			Active:   `{{.Name | bold}} ({{.Host|faint}})`,
			Inactive: `{{.Name}}`,
			Selected: `{{ "Using profile" | faint }}: {{ .Name | bold }}`,
		},
	})
	if err != nil {
		return "", err
	}
	return profiles[i].Name, nil
}
