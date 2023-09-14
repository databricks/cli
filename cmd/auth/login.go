package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/compute"
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

func newLoginCommand(persistentAuth *auth.PersistentAuth) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login [HOST]",
		Short: "Authenticate this machine",
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
		} else {
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

		// If the chosen profile has a hostname and the user hasn't specified a host, infer the host from the profile.
		_, profiles, err := databrickscfg.LoadProfiles(func(p databrickscfg.Profile) bool {
			return p.Name == profileName
		})
		if err != nil {
			return err
		}
		if persistentAuth.Host == "" {
			if len(profiles) > 0 && profiles[0].Host != "" {
				persistentAuth.Host = profiles[0].Host
			} else {
				configureHost(ctx, persistentAuth, args, 0)
			}
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

			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "Loading list of clusters to select from"
			names, err := w.Clusters.ClusterDetailsClusterNameToClusterIdMap(ctx, compute.ListClustersRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load clusters list. Original error: %w", err)
			}
			clusterId, err := cmdio.Select(ctx, names, "Choose cluster")
			if err != nil {
				return err
			}
			cfg.ClusterID = clusterId
		}

		cfg.Profile = profileName
		err = databrickscfg.SaveToProfile(ctx, &cfg)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Profile %s was successfully saved", profileName))
		return nil
	}

	return cmd
}
