package auth

import (
	"context"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

var loginTimeout time.Duration

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

		return databrickscfg.SaveToProfile(ctx, &config.Config{
			Host:      perisistentAuth.Host,
			AccountID: perisistentAuth.AccountID,
			AuthType:  "databricks-cli",
			Profile:   profileName,
		})
	},
}

func init() {
	authCmd.AddCommand(loginCmd)
	loginCmd.Flags().DurationVar(&loginTimeout, "timeout", auth.DefaultTimeout,
		"Timeout for completing login challenge in the browser")
}
