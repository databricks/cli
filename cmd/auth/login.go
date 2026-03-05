package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/cfgpickers"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/experimental/auth/authconv"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	browserpkg "github.com/pkg/browser"
	"github.com/spf13/cobra"
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
)

func newLoginCommand(authArguments *auth.AuthArguments) *cobra.Command {
	defaultConfigPath := "~/.databrickscfg"
	if runtime.GOOS == "windows" {
		defaultConfigPath = "%USERPROFILE%\\.databrickscfg"
	}
	cmd := &cobra.Command{
		Use:   "login [HOST]",
		Short: "Log into a Databricks workspace or account",
		Long: fmt.Sprintf(`Log into a Databricks workspace or account.
This command logs you into the Databricks workspace or account and saves
the authentication configuration in a profile (in %s by default).

This profile can then be used to authenticate other Databricks CLI commands by
specifying the --profile flag. This profile can also be used to authenticate
other Databricks tooling that supports the Databricks Unified Authentication
Specification. This includes the Databricks Go, Python, and Java SDKs. For more information,
you can refer to the documentation linked below.
  AWS: https://docs.databricks.com/dev-tools/auth/index.html
  Azure: https://learn.microsoft.com/azure/databricks/dev-tools/auth
  GCP: https://docs.gcp.databricks.com/dev-tools/auth/index.html


This command requires a Databricks Host URL (using --host or as a positional argument
or implicitly inferred from the specified profile name)
and a profile name (using --profile) to be specified. If you don't specify these
values, you'll be prompted for values at runtime.

While this command always logs you into the specified host, the runtime behaviour
depends on the existing profiles you have set in your configuration file
(at %s by default).

1. If a profile with the specified name exists and specifies a host, you'll
   be logged into the host specified by the profile. The profile will be updated
   to use "databricks-cli" as the auth type if that was not the case before.

2. If a profile with the specified name exists but does not specify a host,
   you'll be prompted to specify a host. The profile will be updated to use the
   specified host. The auth type will be updated to "databricks-cli" if that was
   not the case before.

3. If a profile with the specified name exists and specifies a host, but you
   specify a host using --host (or as the [HOST] positional arg), the profile will
   be updated to use the newly specified host. The auth type will be updated to
   "databricks-cli" if that was not the case before.

4. If a profile with the specified name does not exist, a new profile will be
   created with the specified host. The auth type will be set to "databricks-cli".
`, defaultConfigPath, defaultConfigPath),
	}

	var loginTimeout time.Duration
	var configureCluster bool
	var configureServerless bool
	var scopes string
	cmd.Flags().DurationVar(&loginTimeout, "timeout", defaultTimeout,
		"Timeout for completing login challenge in the browser")
	cmd.Flags().BoolVar(&configureCluster, "configure-cluster", false,
		"Prompts to configure cluster")
	cmd.Flags().BoolVar(&configureServerless, "configure-serverless", false,
		"Prompts to configure serverless")
	cmd.Flags().StringVar(&scopes, "scopes", "",
		"Comma-separated list of OAuth scopes to request (defaults to 'all-apis')")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		profileName := cmd.Flag("profile").Value.String()

		// Cluster and Serverless are mutually exclusive.
		if configureCluster && configureServerless {
			return errors.New("please either configure serverless or cluster, not both")
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

		// Load unified host flags from the profile if not explicitly set via CLI flag
		if !cmd.Flag("experimental-is-unified-host").Changed && existingProfile != nil {
			authArguments.IsUnifiedHost = existingProfile.IsUnifiedHost
		}
		if !cmd.Flag("workspace-id").Changed && existingProfile != nil {
			authArguments.WorkspaceID = existingProfile.WorkspaceID
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
			for _, s := range strings.Split(scopes, ",") {
				scopesList = append(scopesList, strings.TrimSpace(s))
			}
		case existingProfile != nil && existingProfile.Scopes != "":
			// Preserve scopes from the existing profile so re-login
			// uses the same scopes the user previously configured.
			for _, s := range strings.Split(existingProfile.Scopes, ",") {
				scopesList = append(scopesList, strings.TrimSpace(s))
			}
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
		// 1. Configuring cluster and serverless;
		// 2. Saving the profile.

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
				ConfigFile:                 os.Getenv("DATABRICKS_CONFIG_FILE"),
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

	// Determine the host type and handle account ID / workspace ID accordingly
	cfg := &config.Config{
		Host:                       authArguments.Host,
		AccountID:                  authArguments.AccountID,
		WorkspaceID:                authArguments.WorkspaceID,
		Experimental_IsUnifiedHost: authArguments.IsUnifiedHost,
	}

	switch cfg.HostType() {
	case config.AccountHost:
		// Account host - prompt for account ID if not provided
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
		// Unified host requires an account ID for OAuth URL construction
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

		// Workspace ID is optional and determines API access level:
		// - With workspace ID: workspace-level APIs
		// - Without workspace ID: account-level APIs
		// If neither is provided via flags, prompt for workspace ID (most common case)
		hasWorkspaceID := authArguments.WorkspaceID != ""
		if !hasWorkspaceID {
			if existingProfile != nil && existingProfile.WorkspaceID != "" {
				authArguments.WorkspaceID = existingProfile.WorkspaceID
			} else {
				// Prompt for workspace ID for workspace-level access
				workspaceId, err := promptForWorkspaceID(ctx)
				if err != nil {
					return err
				}
				authArguments.WorkspaceID = workspaceId
			}
		}
	case config.WorkspaceHost:
		// Workspace host - no additional prompts needed
	default:
		return fmt.Errorf("unknown host type: %v", cfg.HostType())
	}

	return nil
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

// oauthLoginClearKeys returns profile keys that should be explicitly removed
// when performing an OAuth login. Derives auth credential fields dynamically
// from the SDK's ConfigAttributes to stay in sync as new auth methods are added.
func oauthLoginClearKeys() []string {
	return databrickscfg.AuthCredentialKeys()
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
