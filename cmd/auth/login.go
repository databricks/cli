package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

var loginTimeout time.Duration
var configureCluster bool

var loginCmd = &cobra.Command{
	Use:   "login [HOST]",
	Short: "Authenticate this machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		if perisistentAuth.Host == "" && len(args) == 1 {
			perisistentAuth.Host = args[0]
		}

		defer perisistentAuth.Close()
		ctx, cancel := context.WithTimeout(cmd.Context(), loginTimeout)
		defer cancel()

		var profileName string
		profileFlag := cmd.Flag("profile")
		if profileFlag != nil && profileFlag.Value.String() != "" {
			profileName = profileFlag.Value.String()
		} else {
			prompt := cmdio.Prompt(ctx)
			prompt.Label = "Databricks Profile Name"
			prompt.Default = perisistentAuth.ProfileName()
			prompt.AllowEdit = true
			profile, err := prompt.Run()
			if err != nil {
				return err
			}
			profileName = profile
		}
		err := perisistentAuth.Challenge(ctx)
		if err != nil {
			return err
		}

		err = databrickscfg.SaveToProfile(ctx, &config.Config{
			Host:      perisistentAuth.Host,
			AccountID: perisistentAuth.AccountID,
			AuthType:  "databricks-cli",
			Profile:   profileName,
		})

		if err != nil {
			return err
		}

		if configureCluster {
			err := root.MustWorkspaceClient(cmd, args)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			w := root.WorkspaceClient(ctx)

			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "Loading names for Clusters drop-down."
			names, err := w.Clusters.ClusterInfoClusterNameToClusterIdMap(ctx, compute.ListClustersRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			clusterId, err := cmdio.Select(ctx, names, "The cluster to be attached")
			if err != nil {
				return err
			}
			err = databrickscfg.SaveToProfile(ctx, &config.Config{
				Host:      perisistentAuth.Host,
				AccountID: perisistentAuth.AccountID,
				AuthType:  "databricks-cli",
				Profile:   profileName,
				ClusterId: clusterId,
			})

			if err != nil {
				return err
			}
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
