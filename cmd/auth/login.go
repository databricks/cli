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

var loginTimeout time.Duration
var configureCluster bool

func configureHost(ctx context.Context, args []string, argIndex int) error {
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

var loginCmd = &cobra.Command{
	Use:   "login [HOST]",
	Short: "Authenticate this machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if persistentAuth.Host == "" {
			configureHost(ctx, args, 0)
		}
		defer persistentAuth.Close()

		// We need the config without the profile before it's used to initialise new workspace client below.
		// Otherwise it will complain about non existing profile because it was not yet saved.
		cfg := config.Config{
			Host:     persistentAuth.Host,
			AuthType: "databricks-cli",
		}
		if cfg.IsAccountClient() && persistentAuth.AccountID == "" {
			accountId, err := promptForAccountId(ctx)
			if err != nil {
				return err
			}
			persistentAuth.AccountID = accountId
		}
		cfg.AccountID = persistentAuth.AccountID

		ctx, cancel := context.WithTimeout(ctx, loginTimeout)
		defer cancel()

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
		err := persistentAuth.Challenge(ctx)
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
	},
}

func init() {
	authCmd.AddCommand(loginCmd)
	loginCmd.Flags().DurationVar(&loginTimeout, "timeout", auth.DefaultTimeout,
		"Timeout for completing login challenge in the browser")

	loginCmd.Flags().BoolVar(&configureCluster, "configure-cluster", false,
		"Prompts to configure cluster")
}
