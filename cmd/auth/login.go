package auth

import (
	"context"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func getProfilesMap(ctx context.Context) (map[string]string, error) {
	profiles, err := getAllProfiles(ctx)
	if err != nil {
		return nil, err
	}

	profilesMap := make(map[string]string, len(profiles))

	for _, v := range profiles {
		profilesMap[v.Name] = v.Name
	}

	return profilesMap, nil
}

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

		profileFlag := cmd.Flag("profile")
		if profileFlag != nil && profileFlag.Value.String() != "" {
			perisistentAuth.Profile = profileFlag.Value.String()
		} else {
			profiles, err := getProfilesMap(cmd.Context())
			if err != nil {
				return err
			}
			if len(profiles) > 0 {
				profile, err := cmdio.Select(ctx, profiles, "~/.databrickscfg profile")
				if err != nil {
					return err
				}
				perisistentAuth.Profile = profile
			}
		}
		return perisistentAuth.Challenge(ctx)
	},
}

func init() {
	authCmd.AddCommand(loginCmd)
	loginCmd.Flags().DurationVar(&loginTimeout, "timeout", auth.DefaultTimeout,
		"Timeout for completing login challenge in the browser")
}
