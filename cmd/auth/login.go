package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/cfgpickers"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth/authconv"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	browserpkg "github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

func promptForProfile(ctx context.Context, defaultValue string) (string, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", nil
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks profile name [" + defaultValue + "]"
	prompt.AllowEdit = true
	result, err := prompt.Run()
	if result == "" {
		// Manually return the default value. We could use the prompt.Default
		// field, but be inconsistent with other prompts in the CLI.
		return defaultValue, err
	}
	return result, err
}

const (
	minimalDbConnectVersion = "13.1"
	defaultTimeout          = 1 * time.Hour
	authTypeDatabricksCLI   = "databricks-cli"
	discoveryFallbackTip    = "\n\nTip: you can specify a workspace directly with: databricks auth login --host <url>"
)

// discoveryErr wraps an error (or creates a new one) and appends the
// discovery fallback tip so users know they can bypass login.databricks.com.
func discoveryErr(msg string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w%s", msg, err, discoveryFallbackTip)
	}
	return fmt.Errorf("%s%s", msg, discoveryFallbackTip)
}

type discoveryPersistentAuth interface {
	Challenge() error
	Token() (*oauth2.Token, error)
	Close() error
}

// discoveryClient abstracts the external dependencies of discoveryLogin so
// they can be replaced in tests without package-level variable mutation.
type discoveryClient interface {
	NewOAuthArgument(profileName string) (*u2m.BasicDiscoveryOAuthArgument, error)
	NewPersistentAuth(ctx context.Context, opts ...u2m.PersistentAuthOption) (discoveryPersistentAuth, error)
	IntrospectToken(ctx context.Context, host, accessToken string) (*auth.IntrospectionResult, error)
}

type defaultDiscoveryClient struct{}

func (d *defaultDiscoveryClient) NewOAuthArgument(profileName string) (*u2m.BasicDiscoveryOAuthArgument, error) {
	return u2m.NewBasicDiscoveryOAuthArgument(profileName)
}

func (d *defaultDiscoveryClient) NewPersistentAuth(ctx context.Context, opts ...u2m.PersistentAuthOption) (discoveryPersistentAuth, error) {
	return u2m.NewPersistentAuth(ctx, opts...)
}

func (d *defaultDiscoveryClient) IntrospectToken(ctx context.Context, host, accessToken string) (*auth.IntrospectionResult, error) {
	return auth.IntrospectToken(ctx, host, accessToken, nil)
}

func newLoginCommand(authArguments *auth.AuthArguments) *cobra.Command {
	defaultConfigPath := "~/.databrickscfg"
	if runtime.GOOS == "windows" {
		defaultConfigPath = "%USERPROFILE%\\.databrickscfg"
	}
	cmd := &cobra.Command{
		Use:   "login [PROFILE_OR_HOST]",
		Short: "Log into a Databricks workspace or account",
		Long: fmt.Sprintf(`Log into a Databricks workspace or account.

This command authenticates via OAuth in the browser and saves the result
to a configuration profile (in %s by default). Other Databricks CLI
commands and SDKs can use this profile via the --profile flag. For more
information, see:
  AWS: https://docs.databricks.com/dev-tools/auth/index.html
  Azure: https://learn.microsoft.com/azure/databricks/dev-tools/auth
  GCP: https://docs.gcp.databricks.com/dev-tools/auth/index.html

If no host is provided, the CLI opens login.databricks.com where you can
authenticate and select a workspace.

The positional argument is resolved as a profile name first. If no profile with
that name exists and the argument looks like a URL, it is used as a host. The
host URL may include query parameters to set the workspace and account ID:

  databricks auth login --host "https://<host>?o=<workspace_id>&account_id=<id>"

Note: URLs containing "?" must be quoted to prevent shell interpretation.

If a profile with the given name already exists, it is updated. Otherwise
a new profile is created.
`, defaultConfigPath),
	}

	var loginTimeout time.Duration
	var configureCluster bool
	var configureServerless bool
	var skipWorkspace bool
	var scopes string
	cmd.Flags().DurationVar(&loginTimeout, "timeout", defaultTimeout,
		"Timeout for completing login challenge in the browser")
	cmd.Flags().BoolVar(&configureCluster, "configure-cluster", false,
		"Prompts to configure cluster")
	cmd.Flags().BoolVar(&configureServerless, "configure-serverless", false,
		"Prompts to configure serverless")
	cmd.Flags().BoolVar(&skipWorkspace, "skip-workspace", false,
		"Skip workspace selection for account-level access")
	cmd.Flags().StringVar(&scopes, "scopes", "",
		"Comma-separated list of OAuth scopes to request (defaults to 'all-apis')")

	cmd.PreRunE = profileHostConflictCheck

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		profileName := cmd.Flag("profile").Value.String()

		// Cluster and Serverless are mutually exclusive.
		if configureCluster && configureServerless {
			return errors.New("please either configure serverless or cluster, not both")
		}

		// Resolve positional argument as profile or host.
		if len(args) > 0 && authArguments.Host != "" {
			return errors.New("please only provide a positional argument or --host, not both")
		}
		if profileName == "" && len(args) == 1 {
			resolvedProfile, resolvedHost, err := resolvePositionalArg(ctx, args[0], profile.DefaultProfiler)
			if err != nil {
				return err
			}
			if resolvedProfile != "" {
				profileName = resolvedProfile
				args = nil
			} else {
				authArguments.Host = resolvedHost
				args = nil
			}
		}

		// If the user has not specified a profile name, prompt for one.
		if profileName == "" {
			var err error
			profileName = getProfileName(authArguments)
			if profileName == "" {
				profileName = "DEFAULT"
			}
			profileName, err = promptForProfile(ctx, profileName)
			if err != nil {
				return err
			}
		}

		// Load parameters from the existing profile if any.
		existingProfile, err := loadProfileByName(ctx, profileName, profile.DefaultProfiler)
		if err != nil {
			return err
		}

		// If no host is available from any source, use the discovery flow
		// via login.databricks.com.
		if shouldUseDiscovery(authArguments.Host, args, existingProfile) {
			if err := validateDiscoveryFlagCompatibility(cmd); err != nil {
				return err
			}
			return discoveryLogin(ctx, &defaultDiscoveryClient{}, profileName, loginTimeout, scopes, existingProfile, getBrowserFunc(cmd))
		}

		// Load unified host flag from the profile if not explicitly set via CLI flag.
		// WorkspaceID is NOT loaded here; it is deferred to setHostAndAccountId()
		// so that URL query params (?o=...) can override stale profile values.
		if !cmd.Flag("experimental-is-unified-host").Changed && existingProfile != nil {
			authArguments.IsUnifiedHost = existingProfile.IsUnifiedHost
		}

		err = setHostAndAccountId(ctx, existingProfile, authArguments, args)
		if err != nil {
			return err
		}

		authArguments.Profile = profileName

		var scopesList []string
		switch {
		case scopes != "":
			// Explicit --scopes flag takes precedence.
			scopesList = splitScopes(scopes)
		case existingProfile != nil && existingProfile.Scopes != "":
			// Preserve scopes from the existing profile so re-login
			// uses the same scopes the user previously configured.
			scopesList = splitScopes(existingProfile.Scopes)
		}

		oauthArgument, err := authArguments.ToOAuthArgument()
		if err != nil {
			return err
		}
		persistentAuthOpts := []u2m.PersistentAuthOption{
			u2m.WithOAuthArgument(oauthArgument),
			u2m.WithBrowser(getBrowserFunc(cmd)),
		}
		if len(scopesList) > 0 {
			persistentAuthOpts = append(persistentAuthOpts, u2m.WithScopes(scopesList))
		}
		persistentAuth, err := u2m.NewPersistentAuth(ctx, persistentAuthOpts...)
		if err != nil {
			return err
		}
		defer persistentAuth.Close()

		ctx, cancel := context.WithTimeout(ctx, loginTimeout)
		defer cancel()

		if err = persistentAuth.Challenge(); err != nil {
			return err
		}
		// At this point, an OAuth token has been successfully minted and stored
		// in the CLI cache. The rest of the command focuses on:
		// 1. Workspace selection for SPOG hosts (best-effort);
		// 2. Configuring cluster and serverless;
		// 3. Saving the profile.

		// If discovery gave us an account_id but we still have no workspace_id,
		// prompt the user to select a workspace. This applies to any host where
		// .well-known/databricks-config returned an account_id, regardless of
		// whether IsUnifiedHost is set.
		shouldPromptWorkspace := authArguments.AccountID != "" &&
			authArguments.WorkspaceID == "" &&
			!skipWorkspace

		if skipWorkspace && authArguments.WorkspaceID == "" {
			authArguments.WorkspaceID = auth.WorkspaceIDNone
		}

		if shouldPromptWorkspace {
			wsID, wsErr := promptForWorkspaceSelection(ctx, authArguments, persistentAuth)
			if wsErr != nil {
				log.Warnf(ctx, "Workspace selection failed: %v", wsErr)
			} else if wsID == "" {
				// User selected "Skip" from the prompt.
				authArguments.WorkspaceID = auth.WorkspaceIDNone
			} else {
				authArguments.WorkspaceID = wsID
			}
		}

		var clusterID, serverlessComputeID string

		// Keys to explicitly remove from the profile. OAuth login always
		// clears incompatible credential fields (PAT, basic auth, M2M).
		clearKeys := oauthLoginClearKeys()

		// Boolean false is zero-valued and skipped by SaveToProfile's IsZero
		// check. Explicitly clear experimental_is_unified_host when false so
		// it doesn't remain sticky from a previous login.
		if !authArguments.IsUnifiedHost {
			clearKeys = append(clearKeys, "experimental_is_unified_host")
		}

		switch {
		case configureCluster:
			// Create a workspace client to list clusters for interactive selection.
			// We use a custom CredentialsStrategy that wraps the token we just minted,
			// avoiding the need to spawn a child CLI process (which AuthType "databricks-cli" does).
			w, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host:                       authArguments.Host,
				AccountID:                  authArguments.AccountID,
				WorkspaceID:                authArguments.WorkspaceID,
				Experimental_IsUnifiedHost: authArguments.IsUnifiedHost,
				Credentials:                config.NewTokenSourceStrategy("login-token", authconv.AuthTokenSource(persistentAuth)),
			})
			if err != nil {
				return err
			}
			clusterID, err = cfgpickers.AskForCluster(ctx, w,
				cfgpickers.WithDatabricksConnect(minimalDbConnectVersion))
			if err != nil {
				return err
			}
			// Cluster and serverless are mutually exclusive.
			clearKeys = append(clearKeys, "serverless_compute_id")
		case configureServerless:
			serverlessComputeID = "auto"
			// Cluster and serverless are mutually exclusive.
			clearKeys = append(clearKeys, "cluster_id")
		default:
			// Neither flag: preserve both from existing profile via merge semantics.
		}

		if profileName != "" {
			err := databrickscfg.SaveToProfile(ctx, &config.Config{
				Profile:                    profileName,
				Host:                       authArguments.Host,
				AuthType:                   authTypeDatabricksCLI,
				AccountID:                  authArguments.AccountID,
				WorkspaceID:                authArguments.WorkspaceID,
				Experimental_IsUnifiedHost: authArguments.IsUnifiedHost,
				ClusterID:                  clusterID,
				ConfigFile:                 env.Get(ctx, "DATABRICKS_CONFIG_FILE"),
				ServerlessComputeID:        serverlessComputeID,
				Scopes:                     scopesList,
			}, clearKeys...)
			if err != nil {
				return err
			}

			cmdio.LogString(ctx, fmt.Sprintf("Profile %s was successfully saved", profileName))
		}

		return nil
	}

	return cmd
}

// Sets the host in the persistentAuth object based on the provided arguments and flags.
// Follows the following precedence:
// 1. [HOST] (first positional argument) or --host flag. Error if both are specified.
// 2. Profile host, if available.
// 3. Prompt the user for the host.
//
// Set the account in the persistentAuth object based on the flags.
// Follows the following precedence:
// 1. --account-id flag.
// 2. account-id from the specified profile, if available.
// 3. Prompt the user for the account-id.
func setHostAndAccountId(ctx context.Context, existingProfile *profile.Profile, authArguments *auth.AuthArguments, args []string) error {
	// If both [HOST] and --host are provided, return an error.
	host := authArguments.Host
	if len(args) > 0 && host != "" {
		return errors.New("please only provide a host as an argument or a flag, not both")
	}

	// If the chosen profile has a hostname and the user hasn't specified a host, infer the host from the profile.
	if host == "" {
		if len(args) > 0 {
			// If [HOST] is provided, set the host to the provided positional argument.
			authArguments.Host = args[0]
		} else if existingProfile != nil && existingProfile.Host != "" {
			// If neither [HOST] nor --host are provided, and the profile has a host, use it.
			authArguments.Host = existingProfile.Host
		} else {
			// If neither [HOST] nor --host are provided, and the profile does not have a host,
			// then prompt the user for a host.
			hostName, err := promptForHost(ctx)
			if err != nil {
				return err
			}
			authArguments.Host = hostName
		}
	}

	authArguments.Host = strings.TrimSuffix(authArguments.Host, "/")

	// Extract query parameters from the host URL (?o=workspace_id, ?a=account_id).
	// URL params from explicit --host override stale profile values.
	params := auth.ExtractHostQueryParams(authArguments.Host)
	authArguments.Host = params.Host
	if authArguments.WorkspaceID == "" {
		authArguments.WorkspaceID = params.WorkspaceID
	}
	if authArguments.AccountID == "" {
		authArguments.AccountID = params.AccountID
	}

	// Inherit workspace_id from the existing profile AFTER URL param extraction.
	// This ensures URL params (?o=...) take precedence over stale profile values,
	// while explicit CLI flags (--workspace-id) still win (already set on authArguments).
	if authArguments.WorkspaceID == "" && existingProfile != nil && existingProfile.WorkspaceID != "" {
		authArguments.WorkspaceID = existingProfile.WorkspaceID
	}

	// Call discovery to populate account_id/workspace_id from the host's
	// .well-known/databricks-config endpoint. This is best-effort: failures
	// are logged as warnings and never block login.
	runHostDiscovery(ctx, authArguments)

	// Determine the host type and handle account ID / workspace ID accordingly
	cfg := &config.Config{
		Host:                       authArguments.Host,
		AccountID:                  authArguments.AccountID,
		WorkspaceID:                authArguments.WorkspaceID,
		Experimental_IsUnifiedHost: authArguments.IsUnifiedHost,
	}

	switch cfg.HostType() {
	case config.AccountHost:
		// Account host: prompt for account ID if not provided
		if authArguments.AccountID == "" {
			if existingProfile != nil && existingProfile.AccountID != "" {
				authArguments.AccountID = existingProfile.AccountID
			} else {
				accountId, err := promptForAccountID(ctx)
				if err != nil {
					return err
				}
				authArguments.AccountID = accountId
			}
		}
	case config.UnifiedHost:
		// Unified host requires an account ID for OAuth URL construction.
		// Workspace selection happens post-OAuth via promptForWorkspaceSelection.
		if authArguments.AccountID == "" {
			if existingProfile != nil && existingProfile.AccountID != "" {
				authArguments.AccountID = existingProfile.AccountID
			} else {
				accountId, err := promptForAccountID(ctx)
				if err != nil {
					return err
				}
				authArguments.AccountID = accountId
			}
		}
	case config.WorkspaceHost:
		// Regular workspace host: no additional prompts needed.
		// If discovery already populated account_id/workspace_id, those are kept.
	default:
		return fmt.Errorf("unknown host type: %v", cfg.HostType())
	}

	return nil
}

// runHostDiscovery calls EnsureResolved() with a temporary config to fetch
// .well-known/databricks-config from the host. Populates account_id and
// workspace_id from discovery if not already set.
func runHostDiscovery(ctx context.Context, authArguments *auth.AuthArguments) {
	if authArguments.Host == "" {
		return
	}

	cfg := &config.Config{
		Host:               authArguments.Host,
		AccountID:          authArguments.AccountID,
		WorkspaceID:        authArguments.WorkspaceID,
		HTTPTimeoutSeconds: 5,
		// Use only ConfigAttributes (env vars + struct tags), skip config file
		// loading to avoid interference from existing profiles.
		Loaders: []config.Loader{config.ConfigAttributes},
	}

	err := cfg.EnsureResolved()
	if err != nil {
		log.Warnf(ctx, "Host metadata discovery failed: %v", err)
		return
	}

	if authArguments.AccountID == "" && cfg.AccountID != "" {
		authArguments.AccountID = cfg.AccountID
	}
	if authArguments.WorkspaceID == "" && cfg.WorkspaceID != "" {
		authArguments.WorkspaceID = cfg.WorkspaceID
	}
	if authArguments.DiscoveryURL == "" && cfg.DiscoveryURL != "" {
		authArguments.DiscoveryURL = cfg.DiscoveryURL
	}
}

// getProfileName returns the default profile name for a given host/account ID.
// If the account ID is provided, the profile name is "ACCOUNT-<account-id>".
// Otherwise, the profile name is the first part of the host URL.
func getProfileName(authArguments *auth.AuthArguments) string {
	if authArguments.AccountID != "" {
		return "ACCOUNT-" + authArguments.AccountID
	}
	host := strings.TrimPrefix(authArguments.Host, "https://")
	split := strings.Split(host, ".")
	return split[0]
}

func loadProfileByName(ctx context.Context, profileName string, profiler profile.Profiler) (*profile.Profile, error) {
	if profileName == "" {
		return nil, nil
	}

	if profiler == nil {
		return nil, errors.New("profiler cannot be nil")
	}

	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(profileName))
	// Tolerate ErrNoConfiguration here, as we will write out a configuration as part of the login flow.
	if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
		return nil, err
	}

	if len(profiles) > 0 {
		// LoadProfiles returns only one profile per name, even with multiple profiles in the config file with the same name.
		return &profiles[0], nil
	}
	return nil, nil
}

// shouldUseDiscovery returns true if the discovery flow should be used
// (no host available from any source).
func shouldUseDiscovery(hostFlag string, args []string, existingProfile *profile.Profile) bool {
	if hostFlag != "" {
		return false
	}
	if len(args) > 0 {
		return false
	}
	if existingProfile != nil && existingProfile.Host != "" {
		return false
	}
	return true
}

// discoveryIncompatibleFlags lists flags that require --host and are incompatible
// with the discovery login flow via login.databricks.com.
var discoveryIncompatibleFlags = []string{
	"account-id",
	"workspace-id",
	"experimental-is-unified-host",
	"configure-cluster",
	"configure-serverless",
}

// validateDiscoveryFlagCompatibility returns an error if any flags that require
// --host were explicitly set. These flags are meaningless in discovery mode
// and could lead to incorrect profile configuration.
func validateDiscoveryFlagCompatibility(cmd *cobra.Command) error {
	for _, name := range discoveryIncompatibleFlags {
		if cmd.Flag(name).Changed {
			return fmt.Errorf("--%s requires --host to be specified", name)
		}
	}
	return nil
}

// openURLSuppressingStderr opens a URL in the browser while suppressing stderr output.
// This prevents xdg-open error messages from being displayed to the user.
func openURLSuppressingStderr(url string) error {
	// Save the original stderr from the browser package
	originalStderr := browserpkg.Stderr
	defer func() {
		browserpkg.Stderr = originalStderr
	}()

	// Redirect stderr to discard to suppress xdg-open errors
	browserpkg.Stderr = io.Discard

	// Call the browser open function
	return browserpkg.OpenURL(url)
}

// discoveryLogin runs the login.databricks.com discovery flow. The user
// authenticates in the browser, selects a workspace, and the CLI receives
// the workspace host from the OAuth callback's iss parameter.
func discoveryLogin(ctx context.Context, dc discoveryClient, profileName string, timeout time.Duration, scopes string, existingProfile *profile.Profile, browserFunc func(string) error) error {
	arg, err := dc.NewOAuthArgument(profileName)
	if err != nil {
		return discoveryErr("setting up login.databricks.com", err)
	}

	scopesList := splitScopes(scopes)
	if len(scopesList) == 0 && existingProfile != nil && existingProfile.Scopes != "" {
		scopesList = splitScopes(existingProfile.Scopes)
	}

	opts := []u2m.PersistentAuthOption{
		u2m.WithOAuthArgument(arg),
		u2m.WithBrowser(browserFunc),
		u2m.WithDiscoveryLogin(),
	}
	if len(scopesList) > 0 {
		opts = append(opts, u2m.WithScopes(scopesList))
	}

	// Apply timeout before creating PersistentAuth so Challenge() respects it.
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	persistentAuth, err := dc.NewPersistentAuth(ctx, opts...)
	if err != nil {
		return discoveryErr("setting up login.databricks.com", err)
	}
	defer persistentAuth.Close()

	cmdio.LogString(ctx, "Opening login.databricks.com in your browser...")
	if err := persistentAuth.Challenge(); err != nil {
		return discoveryErr("login via login.databricks.com failed", err)
	}

	discoveredHost := arg.GetDiscoveredHost()
	if discoveredHost == "" {
		return discoveryErr("login succeeded but no workspace host was discovered", nil)
	}

	// Run host metadata discovery on the discovered host to detect SPOG hosts
	// and populate account_id/workspace_id. This ensures profiles created via
	// login.databricks.com have the same metadata as profiles created via the
	// regular --host login path.
	hostArgs := &auth.AuthArguments{Host: discoveredHost}
	runHostDiscovery(ctx, hostArgs)
	accountID := hostArgs.AccountID
	workspaceID := hostArgs.WorkspaceID

	// Best-effort introspection as a fallback for workspace_id when host
	// metadata discovery didn't return it (e.g. classic workspace hosts).
	tok, err := persistentAuth.Token()
	if err != nil {
		return fmt.Errorf("retrieving token after login: %w", err)
	}

	introspection, err := dc.IntrospectToken(ctx, discoveredHost, tok.AccessToken)
	if err != nil {
		log.Debugf(ctx, "token introspection failed (non-fatal): %v", err)
	} else {
		if workspaceID == "" {
			workspaceID = introspection.WorkspaceID
		}
		if accountID == "" {
			accountID = introspection.AccountID
		}

		if existingProfile != nil && existingProfile.AccountID != "" && introspection.AccountID != "" &&
			existingProfile.AccountID != introspection.AccountID {
			log.Warnf(ctx, "detected account ID %q differs from existing profile account ID %q",
				introspection.AccountID, existingProfile.AccountID)
		}
	}

	configFile := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	clearKeys := oauthLoginClearKeys()
	// Discovery login always produces a workspace-level profile pointing at the
	// discovered host. Any previous routing metadata (is_unified_host,
	// cluster_id, serverless_compute_id) from a prior login to a different host
	// type must be cleared so they don't leak into the new profile. account_id
	// and workspace_id are re-added from discovery/introspection results.
	clearKeys = append(clearKeys,
		"account_id",
		"workspace_id",
		"experimental_is_unified_host",
		"cluster_id",
		"serverless_compute_id",
	)
	err = databrickscfg.SaveToProfile(ctx, &config.Config{
		Profile:     profileName,
		Host:        discoveredHost,
		AuthType:    authTypeDatabricksCLI,
		AccountID:   accountID,
		WorkspaceID: workspaceID,
		Scopes:      scopesList,
		ConfigFile:  configFile,
	}, clearKeys...)
	if err != nil {
		if configFile != "" {
			return fmt.Errorf("saving profile %q to %s: %w", profileName, configFile, err)
		}
		return fmt.Errorf("saving profile %q: %w", profileName, err)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Profile %s was successfully saved", profileName))
	return nil
}

// splitScopes splits a comma-separated scopes string into a trimmed slice.
func splitScopes(scopes string) []string {
	var result []string
	for _, s := range strings.Split(scopes, ",") {
		scope := strings.TrimSpace(s)
		if scope == "" {
			continue
		}
		result = append(result, scope)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// oauthLoginClearKeys returns profile keys that should be explicitly removed
// when performing an OAuth login. Derives auth credential fields dynamically
// from the SDK's ConfigAttributes to stay in sync as new auth methods are added.
func oauthLoginClearKeys() []string {
	return databrickscfg.AuthCredentialKeys()
}

// promptForWorkspaceSelection lists workspaces for a SPOG account and lets the
// user pick one. Returns the selected workspace ID or empty string if skipped.
// This is best-effort: errors are returned to the caller for logging, not shown
// to the user.
func promptForWorkspaceSelection(ctx context.Context, authArguments *auth.AuthArguments, persistentAuth *u2m.PersistentAuth) (string, error) {
	if !cmdio.IsPromptSupported(ctx) {
		cmdio.LogString(ctx, "To use workspace commands, set workspace_id in your profile or pass --workspace-id.")
		return "", nil
	}

	a, err := databricks.NewAccountClient(&databricks.Config{
		Host:        authArguments.Host,
		AccountID:   authArguments.AccountID,
		Credentials: config.NewTokenSourceStrategy("login-token", authconv.AuthTokenSource(persistentAuth)),
	})
	if err != nil {
		return "", err
	}

	workspaces, err := a.Workspaces.List(ctx)
	if err != nil {
		log.Debugf(ctx, "Failed to load workspaces (this can happen if the user has no account-level access): %v", err)
		return promptForWorkspaceID(ctx)
	}

	if len(workspaces) == 0 {
		return "", nil
	}

	const maxWorkspaces = 50
	if len(workspaces) > maxWorkspaces {
		cmdio.LogString(ctx, fmt.Sprintf("Account has %d workspaces. Showing first %d. Use --workspace-id to specify directly.", len(workspaces), maxWorkspaces))
		workspaces = workspaces[:maxWorkspaces]
	}

	if len(workspaces) == 1 {
		wsID := strconv.FormatInt(workspaces[0].WorkspaceId, 10)
		cmdio.LogString(ctx, fmt.Sprintf("Auto-selected workspace %q (%s)", workspaces[0].WorkspaceName, wsID))
		return wsID, nil
	}

	items := make([]cmdio.Tuple, 0, len(workspaces)+1)
	for _, ws := range workspaces {
		items = append(items, cmdio.Tuple{
			Name: ws.WorkspaceName,
			Id:   strconv.FormatInt(ws.WorkspaceId, 10),
		})
	}
	// Allow skipping workspace selection for account-level access.
	items = append(items, cmdio.Tuple{
		Name: "Skip (account-level access only)",
		Id:   "",
	})

	selected, err := cmdio.SelectOrdered(ctx, items, "Select a workspace")
	if err != nil {
		return "", err
	}
	return selected, nil
}

// promptForWorkspaceID asks the user to manually enter a workspace ID.
// Returns empty string if the user provides no input.
func promptForWorkspaceID(ctx context.Context) (string, error) {
	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Enter workspace ID (empty to skip)"
	prompt.AllowEdit = true
	result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

// getBrowserFunc returns a function that opens the given URL in the browser.
// It respects the BROWSER environment variable:
// - empty string: uses the default browser
// - "none": prints the URL to stdout without opening a browser
// - custom command: executes the specified command with the URL as argument
func getBrowserFunc(cmd *cobra.Command) func(url string) error {
	browser := env.Get(cmd.Context(), "BROWSER")
	switch browser {
	case "":
		return openURLSuppressingStderr
	case "none":
		return func(url string) error {
			cmdio.LogString(cmd.Context(), "Please complete authentication by opening this link in your browser:\n"+url)
			return nil
		}
	default:
		return func(url string) error {
			// Run the browser command via a shell.
			// It can be a script or a binary and scripts cannot be executed directly on Windows.
			e, err := exec.NewCommandExecutor(".")
			if err != nil {
				return err
			}

			e.WithInheritOutput()
			cmd, err := e.StartCommand(cmd.Context(), fmt.Sprintf("%q %q", browser, url))
			if err != nil {
				return err
			}

			return cmd.Wait()
		}
	}
}
