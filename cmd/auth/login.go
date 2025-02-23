package auth

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/cfgpickers"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

func promptForProfile(ctx context.Context, defaultValue string) (string, error) {
	if !cmdio.IsInTTY(ctx) {
		return "", nil
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks profile name"
	prompt.Default = defaultValue
	prompt.AllowEdit = true
	return prompt.Run()
}

const (
	minimalDbConnectVersion = "13.1"
	defaultTimeout          = 1 * time.Hour
)

func newLoginCommand(persistentAuth *auth.PersistentAuth) *cobra.Command {
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
	cmd.Flags().DurationVar(&loginTimeout, "timeout", defaultTimeout,
		"Timeout for completing login challenge in the browser")
	cmd.Flags().BoolVar(&configureCluster, "configure-cluster", false,
		"Prompts to configure cluster")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		profileName := cmd.Flag("profile").Value.String()

		// If the user has not specified a profile name, prompt for one.
		if profileName == "" {
			var err error
			profileName, err = promptForProfile(ctx, persistentAuth.ProfileName())
			if err != nil {
				return err
			}
		}

		// Set the host and account-id based on the provided arguments and flags.
		err := setHostAndAccountId(ctx, profileName, persistentAuth, args)
		if err != nil {
			return err
		}
		defer persistentAuth.Close()

		// We need the config without the profile before it's used to initialise new workspace client below.
		// Otherwise it will complain about non existing profile because it was not yet saved.
		cfg := config.Config{
			Host:      persistentAuth.Host,
			AccountID: persistentAuth.AccountID,
			AuthType:  "databricks-cli",
		}

		ctx, cancel := context.WithTimeout(ctx, loginTimeout)
		defer cancel()

		err = persistentAuth.Challenge(ctx)
		if err != nil {
			return err
		}

		if configureCluster {
			w, err := databricks.NewWorkspaceClient((*databricks.Config)(&cfg))
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			clusterID, err := cfgpickers.AskForCluster(ctx, w,
				cfgpickers.WithDatabricksConnect(minimalDbConnectVersion))
			if err != nil {
				return err
			}
			cfg.ClusterID = clusterID
		}

		if profileName != "" {
			err = databrickscfg.SaveToProfile(ctx, &config.Config{
				Profile:   profileName,
				Host:      cfg.Host,
				AuthType:  cfg.AuthType,
				AccountID: cfg.AccountID,
				ClusterID: cfg.ClusterID,
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
func setHostAndAccountId(ctx context.Context, profileName string, persistentAuth *auth.PersistentAuth, args []string) error {
	// If both [HOST] and --host are provided, return an error.
	if len(args) > 0 && persistentAuth.Host != "" {
		return errors.New("please only provide a host as an argument or a flag, not both")
	}

	profiler := profile.GetProfiler(ctx)
	// If the chosen profile has a hostname and the user hasn't specified a host, infer the host from the profile.
	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(profileName))
	// Tolerate ErrNoConfiguration here, as we will write out a configuration as part of the login flow.
	if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
		return err
	}

	if persistentAuth.Host == "" {
		if len(args) > 0 {
			// If [HOST] is provided, set the host to the provided positional argument.
			persistentAuth.Host = args[0]
		} else if len(profiles) > 0 && profiles[0].Host != "" {
			// If neither [HOST] nor --host are provided, and the profile has a host, use it.
			persistentAuth.Host = profiles[0].Host
		} else {
			// If neither [HOST] nor --host are provided, and the profile does not have a host,
			// then prompt the user for a host.
			hostName, err := promptForHost(ctx)
			if err != nil {
				return err
			}
			persistentAuth.Host = hostName
		}
	}

	// If the account-id was not provided as a cmd line flag, try to read it from
	// the specified profile.
	isAccountClient := (&config.Config{Host: persistentAuth.Host}).IsAccountClient()
	if isAccountClient && persistentAuth.AccountID == "" {
		if len(profiles) > 0 && profiles[0].AccountID != "" {
			persistentAuth.AccountID = profiles[0].AccountID
		} else {
			// Prompt user for the account-id if it we could not get it from a
			// profile.
			accountId, err := promptForAccountID(ctx)
			if err != nil {
				return err
			}
			persistentAuth.AccountID = accountId
		}
	}
	return nil
}
