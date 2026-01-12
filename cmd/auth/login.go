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
	cmd.Flags().DurationVar(&loginTimeout, "timeout", defaultTimeout,
		"Timeout for completing login challenge in the browser")
	cmd.Flags().BoolVar(&configureCluster, "configure-cluster", false,
		"Prompts to configure cluster")
	cmd.Flags().BoolVar(&configureServerless, "configure-serverless", false,
		"Prompts to configure serverless")

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

		oauthArgument, err := authArguments.ToOAuthArgument()
		if err != nil {
			return err
		}
		persistentAuthOpts := []u2m.PersistentAuthOption{
			u2m.WithOAuthArgument(oauthArgument),
		}
		persistentAuthOpts = append(persistentAuthOpts, u2m.WithBrowser(getBrowserFunc(cmd)))
		persistentAuth, err := u2m.NewPersistentAuth(ctx, persistentAuthOpts...)
		if err != nil {
			return err
		}
		defer persistentAuth.Close()

		// We need the config without the profile before it's used to initialise new workspace client below.
		// Otherwise it will complain about non existing profile because it was not yet saved.
		cfg := config.Config{
			Host:      authArguments.Host,
			AccountID: authArguments.AccountID,
			AuthType:  "databricks-cli",
		}
		databricksCfgFile := os.Getenv("DATABRICKS_CONFIG_FILE")
		if databricksCfgFile != "" {
			cfg.ConfigFile = databricksCfgFile
		}

		ctx, cancel := context.WithTimeout(ctx, loginTimeout)
		defer cancel()

		if err = persistentAuth.Challenge(); err != nil {
			return err
		}

		switch {
		case configureCluster:
			w, err := databricks.NewWorkspaceClient((*databricks.Config)(&cfg))
			if err != nil {
				return err
			}
			clusterID, err := cfgpickers.AskForCluster(ctx, w,
				cfgpickers.WithDatabricksConnect(minimalDbConnectVersion))
			if err != nil {
				return err
			}
			cfg.ClusterID = clusterID
		case configureServerless:
			cfg.ClusterID = ""
			cfg.ServerlessComputeID = "auto"
		default:
			// Respect the existing profile if it exists, even if it has
			// both cluster and serverless configured. Tools relying on
			// these fields from the profile will need to handle this case.
			//
			// TODO: consider whether we should use this an an opportunity
			// to clean up the profile under the assumption that serverless
			// is the preferred option.
			if existingProfile != nil {
				cfg.ClusterID = existingProfile.ClusterID
				cfg.ServerlessComputeID = existingProfile.ServerlessComputeID
			}
		}

		if profileName != "" {
			err = databrickscfg.SaveToProfile(ctx, &config.Config{
				Profile:                    profileName,
				Host:                       cfg.Host,
				AuthType:                   cfg.AuthType,
				AccountID:                  cfg.AccountID,
				WorkspaceId:                authArguments.WorkspaceID,
				Experimental_IsUnifiedHost: authArguments.IsUnifiedHost,
				ClusterID:                  cfg.ClusterID,
				ConfigFile:                 cfg.ConfigFile,
				ServerlessComputeID:        cfg.ServerlessComputeID,
			})
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
		WorkspaceId:                authArguments.WorkspaceID,
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
		// Unified host - prompt for either workspace ID or account ID
		// User must specify exactly one of them
		hasWorkspaceID := authArguments.WorkspaceID != ""
		hasAccountID := authArguments.AccountID != ""

		if hasWorkspaceID && hasAccountID {
			return errors.New("unified host cannot have both workspace_id and account_id; please specify only one")
		}

		if !hasWorkspaceID && !hasAccountID {
			// Neither provided - prompt user to choose
			// For now, prompt for workspace ID by default (most common case)
			// TODO: Could add a prompt to ask which one the user wants
			workspaceId, err := promptForWorkspaceID(ctx)
			if err != nil {
				return err
			}
			authArguments.WorkspaceID = workspaceId
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
			fmt.Fprintf(cmd.OutOrStdout(), "Please complete authentication by opening this link in your browser:\n%s\n", url)
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
