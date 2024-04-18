package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/cfgpickers"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

func promptForProfile(ctx context.Context, dv string) (string, error) {
	if !cmdio.IsInTTY(ctx) {
		return "", fmt.Errorf("the command is being run in a non-interactive environment, please specify a profile using --profile")
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks Profile Name"
	prompt.Default = dv
	prompt.AllowEdit = true
	return prompt.Run()
}

func getHostFromProfile(ctx context.Context, profileName string) (string, error) {
	_, profiles, err := databrickscfg.LoadProfiles(ctx, func(p databrickscfg.Profile) bool {
		return p.Name == profileName
	})

	// Tolerate ErrNoConfiguration here, as we will write out a configuration file
	// as part of the login flow.
	if err != nil && errors.Is(err, databrickscfg.ErrNoConfiguration) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	// Return host from profile
	if len(profiles) > 0 && profiles[0].Host != "" {
		return profiles[0].Host, nil
	}
	return "", nil
}

const minimalDbConnectVersion = "13.1"

func newLoginCommand(persistentAuth *auth.PersistentAuth) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login [HOST]",
		Short: "Authenticate this machine",
	}

	var loginTimeout time.Duration
	var configureCluster bool
	var profileName string
	cmd.Flags().DurationVar(&loginTimeout, "timeout", auth.DefaultTimeout,
		"Timeout for completing login challenge in the browser")
	cmd.Flags().BoolVar(&configureCluster, "configure-cluster", false,
		"Prompts to configure cluster")
	cmd.Flags().StringVarP(&profileName, "profile", "p", "", `Name of the profile.`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// If the user has not specified a profile name, prompt for one.
		if profileName == "" {
			var err error
			profileName, err = promptForProfile(ctx, persistentAuth.DefaultProfileName())
			if err != nil {
				return err
			}
		}

		// Set the host based on the provided arguments and flags.
		err := setHost(ctx, profileName, persistentAuth, args)
		if err != nil {
			return err
		}

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
		defer persistentAuth.Close()

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
func setHost(ctx context.Context, profileName string, persistentAuth *auth.PersistentAuth, args []string) error {
	// If both [HOST] and --host are provided, return an error.
	if len(args) > 0 && persistentAuth.Host != "" {
		return fmt.Errorf("please only provide a host as an argument or a flag, not both")
	}

	// If [HOST] is provided, set the host to the provided positional argument.
	if len(args) > 0 && persistentAuth.Host == "" {
		persistentAuth.Host = args[0]
	}

	// If neither [HOST] nor --host are provided, and the profile has a host, use it.
	// Otherwise, prompt the user for a host.
	if len(args) == 0 && persistentAuth.Host == "" {
		hostName, err := getHostFromProfile(ctx, profileName)
		if err != nil {
			return err
		}
		if hostName == "" {
			var err error
			hostName, err = promptForHost(ctx)
			if err != nil {
				return err
			}
		}
		persistentAuth.Host = hostName
	}
	return nil
}
