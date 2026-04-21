package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/browser"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
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

// profileSelectionResult represents the user's choice from the interactive
// profile picker.
type profileSelectionResult int

const (
	profileSelected   profileSelectionResult = iota // User picked a profile
	enterHostSelected                               // User chose "Enter a host URL manually"
	createNewSelected                               // User chose "Create a new profile"
)

// applyUnifiedHostFlags copies unified host fields from the profile to the
// auth arguments when they are not already set. WorkspaceID is NOT copied
// here; it is deferred to setHostAndAccountId() so that URL query params
// (?o=...) can override stale profile values.
func applyUnifiedHostFlags(p *profile.Profile, args *auth.AuthArguments) {
	if p == nil {
		return
	}
	if !args.IsUnifiedHost && p.IsUnifiedHost {
		args.IsUnifiedHost = p.IsUnifiedHost
	}
}

func newTokenCommand(authArguments *auth.AuthArguments) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token [PROFILE]",
		Short: "Get authentication token",
		Long: `Get authentication token from the local cache in ~/.databricks/token-cache.json.
Refresh the access token if it is expired or close to expiry. Use --force-refresh
to bypass expiry checks. Note: This command only works with U2M authentication
(using the 'databricks auth login' command). M2M authentication using a client ID
and secret is not supported.`,
	}

	var tokenTimeout time.Duration
	cmd.Flags().DurationVar(&tokenTimeout, "timeout", defaultTimeout,
		"Timeout for acquiring a token.")

	var forceRefresh bool
	cmd.Flags().BoolVar(&forceRefresh, "force-refresh", false,
		"Force a token refresh even if the cached token is still valid.")

	cmd.PreRunE = profileHostConflictCheck

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		profileName := cmd.Flag("profile").Value.String()

		t, err := loadToken(ctx, loadTokenArgs{
			authArguments:      authArguments,
			profileName:        profileName,
			args:               args,
			tokenTimeout:       tokenTimeout,
			forceRefresh:       forceRefresh,
			profiler:           profile.DefaultProfiler,
			persistentAuthOpts: nil,
		})
		if err != nil {
			return err
		}
		// Only honor the explicit --output text flag, not implicit text mode
		// (e.g. from DATABRICKS_OUTPUT_FORMAT). auth token defaults to JSON,
		// and changing that implicitly would break scripts that parse JSON output.
		textMode := cmd.Flag("output").Changed && root.OutputType(cmd) == flags.OutputText
		return writeTokenOutput(cmd.OutOrStdout(), t, textMode)
	}

	return cmd
}

func writeTokenOutput(w io.Writer, t *oauth2.Token, textMode bool) error {
	if textMode {
		_, err := fmt.Fprintln(w, t.AccessToken)
		return err
	}

	raw, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
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

	// forceRefresh forces a token refresh even if the cached token is still valid.
	forceRefresh bool

	// profiler is the profiler to use for reading the host and account ID from the .databrickscfg file.
	profiler profile.Profiler

	// persistentAuthOpts are the options to pass to the persistent auth client.
	persistentAuthOpts []u2m.PersistentAuthOption
}

// loadToken loads an OAuth token from the persistent auth store. The host and account ID are read from
// the provided profiler if not explicitly provided. If the token cannot be refreshed, a helpful error message
// is printed to the user with steps to reauthenticate.
func loadToken(ctx context.Context, args loadTokenArgs) (*oauth2.Token, error) {
	// The positional argument is a shorthand that resolves to either a
	// profile or a host. It cannot be combined with explicit flags.
	if len(args.args) > 0 && (args.authArguments.Host != "" || args.profileName != "") {
		return nil, fmt.Errorf("argument %q cannot be combined with --host or --profile. Use the --host and --profile flags instead", args.args[0])
	}

	// Resolve the positional arg as a profile name first, then as a host.
	// Error if it matches neither. This runs before the DATABRICKS_CONFIG_PROFILE
	// env var check so that an explicit positional argument always goes through
	// profile-first resolution.
	if len(args.args) == 1 {
		resolvedProfile, resolvedHost, err := resolvePositionalArg(ctx, args.args[0], args.profiler)
		if err != nil {
			return nil, err
		}
		if resolvedProfile != "" {
			args.profileName = resolvedProfile
			args.args = nil
		} else {
			args.authArguments.Host = resolvedHost
			args.args = nil
		}
	}

	// When no explicit --profile flag or positional arg is provided, check the
	// env var. This handles the case where downstream tools (like the Terraform
	// provider) pass --host but not --profile, while DATABRICKS_CONFIG_PROFILE
	// is set.
	if args.profileName == "" {
		args.profileName = env.Get(ctx, "DATABRICKS_CONFIG_PROFILE")
	}

	existingProfile, err := loadProfileByName(ctx, args.profileName, args.profiler)
	if err != nil {
		return nil, err
	}

	applyUnifiedHostFlags(existingProfile, args.authArguments)

	// When no explicit profile, host, or positional args are provided, attempt to
	// resolve the target through environment variables or interactive profile selection.
	if args.profileName == "" && args.authArguments.Host == "" && len(args.args) == 0 {
		var resolvedProfile string
		resolvedProfile, existingProfile, err = resolveNoArgsToken(ctx, args.profiler, args.authArguments)
		if err != nil {
			return nil, err
		}
		args.profileName = resolvedProfile
		applyUnifiedHostFlags(existingProfile, args.authArguments)
	}

	err = setHostAndAccountId(ctx, existingProfile, args.authArguments, args.args)
	if err != nil {
		return nil, err
	}

	// When no profile was specified, resolve the host to a profile in
	// .databrickscfg. This ensures the token cache lookup uses the profile
	// key (e.g. "logfood") rather than the host URL, which is important
	// because the SDK's dualWrite is a transitional mechanism: it writes
	// tokens under both keys for backward compatibility with older SDKs
	// that only know host keys, but the profile key is the intended
	// primary key. Once older SDKs have migrated to profile-based keys,
	// dualWrite and the host key can be removed entirely.
	if args.profileName == "" && args.authArguments.Host != "" {
		// Match profiles by host and available identifiers. For SPOG workspace
		// profiles (host + account_id + workspace_id), use all three to
		// disambiguate between workspaces sharing the same host and account.
		var matchFn profile.ProfileMatchFunction
		if args.authArguments.AccountID != "" && args.authArguments.WorkspaceID != "" {
			matchFn = profile.WithHostAccountIDAndWorkspaceID(args.authArguments.Host, args.authArguments.AccountID, args.authArguments.WorkspaceID)
		} else if args.authArguments.AccountID != "" {
			matchFn = profile.WithHostAndAccountID(args.authArguments.Host, args.authArguments.AccountID)
		} else {
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
			selected, err := profile.SelectProfile(ctx, profile.SelectConfig{
				Label:             "Multiple profiles match " + args.authArguments.Host,
				StartInSearchMode: true,
				Profiles:          matchingProfiles,
				ActiveTemplate:    `{{.Name | bold}} ({{.Host|faint}})`,
				InactiveTemplate:  `{{.Name}}`,
				SelectedTemplate:  `{{ "Using profile" | faint }}: {{ .Name | bold }}`,
			})
			if err != nil {
				return nil, err
			}
			args.profileName = selected
			existingProfile, err = loadProfileByName(ctx, selected, args.profiler)
			if err != nil {
				return nil, err
			}
		} else if len(matchingProfiles) == 1 {
			args.profileName = matchingProfiles[0].Name
			existingProfile = &matchingProfiles[0]
		}
	}

	// Check if the resolved profile uses M2M authentication (client credentials).
	// The auth token command only supports U2M OAuth tokens.
	if existingProfile != nil && existingProfile.HasClientCredentials {
		return nil, fmt.Errorf(
			"profile %q uses M2M authentication (client_id/client_secret). "+
				"`databricks auth token` only supports U2M (user-to-machine) authentication tokens. "+
				"To authenticate as a service principal, use the Databricks SDK directly",
			args.profileName,
		)
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
	var t *oauth2.Token
	if args.forceRefresh {
		t, err = persistentAuth.ForceRefreshToken()
	} else {
		t, err = persistentAuth.Token()
	}
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

// resolveNoArgsToken resolves a profile or host when `auth token` is invoked
// with no explicit profile, host, or positional arguments. It checks environment
// variables first, then falls back to interactive profile selection or a clear
// non-interactive error.
//
// Returns the resolved profile name and profile (if any). The host and related
// fields on authArgs are updated in place when resolved via environment variables.
func resolveNoArgsToken(ctx context.Context, profiler profile.Profiler, authArgs *auth.AuthArguments) (string, *profile.Profile, error) {
	// Step 1: Try DATABRICKS_HOST env var (highest priority).
	if envHost := env.Get(ctx, "DATABRICKS_HOST"); envHost != "" {
		authArgs.Host = envHost
		if v := env.Get(ctx, "DATABRICKS_ACCOUNT_ID"); v != "" {
			authArgs.AccountID = v
		}
		if v := env.Get(ctx, "DATABRICKS_WORKSPACE_ID"); v != "" {
			authArgs.WorkspaceID = v
		}
		return "", nil, nil
	}

	// Step 2: Try DATABRICKS_CONFIG_PROFILE env var.
	if envProfile := env.Get(ctx, "DATABRICKS_CONFIG_PROFILE"); envProfile != "" {
		p, err := loadProfileByName(ctx, envProfile, profiler)
		if err != nil {
			return "", nil, err
		}
		return envProfile, p, nil
	}

	// Step 3: No env vars resolved. Load all profiles for interactive selection
	// or non-interactive error.
	allProfiles, err := profiler.LoadProfiles(ctx, profile.MatchAllProfiles)
	if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
		return "", nil, err
	}

	if !cmdio.IsPromptSupported(ctx) {
		if len(allProfiles) > 0 {
			return "", nil, errors.New("no profile specified. Use --profile <name> to specify which profile to use")
		}
		return "", nil, errors.New("no profiles configured. Run 'databricks auth login' to create a profile")
	}

	// Interactive: show profile picker.
	result, selectedName, err := promptForProfileSelection(ctx, allProfiles)
	if err != nil {
		return "", nil, err
	}
	switch result {
	case enterHostSelected:
		// Fall through — setHostAndAccountId will prompt for the host.
		return "", nil, nil
	case createNewSelected:
		return runInlineLogin(ctx, profiler)
	default:
		p, err := loadProfileByName(ctx, selectedName, profiler)
		if err != nil {
			return "", nil, err
		}
		return selectedName, p, nil
	}
}

// profileSelectItem is used by promptForProfileSelection to render both
// regular profiles and special action options in the same select list.
type profileSelectItem struct {
	Name string
	Host string
}

// promptForProfileSelection shows a promptui select list with all configured
// profiles plus "Enter a host URL" and "Create a new profile" options.
// Returns the selection type and, when a profile is selected, its name.
func promptForProfileSelection(ctx context.Context, profiles profile.Profiles) (profileSelectionResult, string, error) {
	items := make([]profileSelectItem, 0, len(profiles)+2)
	for _, p := range profiles {
		items = append(items, profileSelectItem{Name: p.Name, Host: p.Host})
	}
	createProfileIdx := len(items)
	items = append(items, profileSelectItem{Name: "Create a new profile"})
	enterHostIdx := len(items)
	items = append(items, profileSelectItem{Name: "Enter a host URL manually"})

	i, _, err := cmdio.RunSelect(ctx, &promptui.Select{
		Label:             "Select a profile",
		Items:             items,
		StartInSearchMode: len(profiles) > 5,
		Searcher: func(input string, index int) bool {
			input = strings.ToLower(input)
			name := strings.ToLower(items[index].Name)
			host := strings.ToLower(items[index].Host)
			return strings.Contains(name, input) || strings.Contains(host, input)
		},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | faint }}",
			Active:   `{{.Name | bold}}{{if .Host}} ({{.Host|faint}}){{end}}`,
			Inactive: `{{.Name}}{{if .Host}} ({{.Host}}){{end}}`,
			Selected: `{{ "Using profile" | faint }}: {{ .Name | bold }}`,
		},
	})
	if err != nil {
		return 0, "", err
	}

	switch i {
	case enterHostIdx:
		return enterHostSelected, "", nil
	case createProfileIdx:
		return createNewSelected, "", nil
	default:
		return profileSelected, profiles[i].Name, nil
	}
}

// runInlineLogin runs a minimal interactive login flow: prompts for a profile
// name and host, performs the OAuth challenge, saves the profile to
// .databrickscfg, and returns the new profile name and profile.
func runInlineLogin(ctx context.Context, profiler profile.Profiler) (string, *profile.Profile, error) {
	profileName, err := promptForProfile(ctx, "DEFAULT")
	if err != nil {
		return "", nil, err
	}

	existingProfile, err := loadProfileByName(ctx, profileName, profiler)
	if err != nil {
		return "", nil, err
	}

	loginArgs := &auth.AuthArguments{}
	applyUnifiedHostFlags(existingProfile, loginArgs)

	err = setHostAndAccountId(ctx, existingProfile, loginArgs, nil)
	if err != nil {
		return "", nil, err
	}

	loginArgs.Profile = profileName

	// Preserve scopes from the existing profile so the inline login
	// uses the same scopes the user previously configured.
	var scopesList []string
	if existingProfile != nil && existingProfile.Scopes != "" {
		scopesList = splitScopes(existingProfile.Scopes)
	}

	oauthArgument, err := loginArgs.ToOAuthArgument()
	if err != nil {
		return "", nil, err
	}
	persistentAuthOpts := []u2m.PersistentAuthOption{
		u2m.WithOAuthArgument(oauthArgument),
		u2m.WithBrowser(func(url string) error { return browser.Open(ctx, url) }),
	}
	if len(scopesList) > 0 {
		persistentAuthOpts = append(persistentAuthOpts, u2m.WithScopes(scopesList))
	}
	persistentAuth, err := u2m.NewPersistentAuth(ctx, persistentAuthOpts...)
	if err != nil {
		return "", nil, err
	}
	defer persistentAuth.Close()

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if err = persistentAuth.Challenge(); err != nil {
		return "", nil, err
	}

	clearKeys := oauthLoginClearKeys()
	clearKeys = append(clearKeys, "experimental_is_unified_host")

	err = databrickscfg.SaveToProfile(ctx, &config.Config{
		Profile:     profileName,
		Host:        loginArgs.Host,
		AuthType:    authTypeDatabricksCLI,
		AccountID:   loginArgs.AccountID,
		WorkspaceID: loginArgs.WorkspaceID,
		ConfigFile:  env.Get(ctx, "DATABRICKS_CONFIG_FILE"),
		Scopes:      scopesList,
	}, clearKeys...)
	if err != nil {
		return "", nil, err
	}

	cmdio.LogString(ctx, fmt.Sprintf("Profile %s was successfully saved", profileName))

	p, err := loadProfileByName(ctx, profileName, profiler)
	if err != nil {
		return "", nil, err
	}
	return profileName, p, nil
}
