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
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

func configureHost(ctx context.Context, persistentAuth *auth.PersistentAuth, args []string, argIndex int) error {
	if len(args) > argIndex {
		persistentAuth.Host = args[argIndex]
		return nil
	}

	host, err := promptForHost(ctx)
	if err != nil {
		return err
	}
	persistentAuth.Host = host
	return nil
}

const minimalDbConnectVersion = "13.1"

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
	cmd.Flags().DurationVar(&loginTimeout, "timeout", auth.DefaultTimeout,
		"Timeout for completing login challenge in the browser")
	cmd.Flags().BoolVar(&configureCluster, "configure-cluster", false,
		"Prompts to configure cluster")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		var profileName string
		profileFlag := cmd.Flag("profile")
		if profileFlag != nil && profileFlag.Value.String() != "" {
			profileName = profileFlag.Value.String()
		} else if cmdio.IsInTTY(ctx) {
			prompt := cmdio.Prompt(ctx)
			prompt.Label = "Databricks Profile Name"
			prompt.Default = persistentAuth.ProfileName()
			prompt.AllowEdit = true
			profile, err := prompt.Run()
			if err != nil {
				return err
			}
			profileName = profile
		}

		err := setHost(ctx, profileName, persistentAuth, args)
		if err != nil {
			return err
		}
		defer persistentAuth.Close()

		// We need the config without the profile before it's used to initialise new workspace client below.
		// Otherwise it will complain about non existing profile because it was not yet saved.
		cfg := config.Config{
			Host:     persistentAuth.Host,
			AuthType: "databricks-cli",
		}
		if cfg.IsAccountClient() && persistentAuth.AccountID == "" {
			accountId, err := promptForAccountID(ctx)
			if err != nil {
				return err
			}
			persistentAuth.AccountID = accountId
		}
		cfg.AccountID = persistentAuth.AccountID

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

func setHost(ctx context.Context, profileName string, persistentAuth *auth.PersistentAuth, args []string) error {
	// If the chosen profile has a hostname and the user hasn't specified a host, infer the host from the profile.
	_, profiles, err := databrickscfg.LoadProfiles(ctx, func(p databrickscfg.Profile) bool {
		return p.Name == profileName
	})
	// Tolerate ErrNoConfiguration here, as we will write out a configuration as part of the login flow.
	if err != nil && !errors.Is(err, databrickscfg.ErrNoConfiguration) {
		return err
	}
	if persistentAuth.Host == "" {
		if len(profiles) > 0 && profiles[0].Host != "" {
			persistentAuth.Host = profiles[0].Host
		} else {
			configureHost(ctx, persistentAuth, args, 0)
		}
	}
	return nil
}
