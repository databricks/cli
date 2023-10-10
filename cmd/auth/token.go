package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/spf13/cobra"
)

func newTokenCommand(persistentAuth *auth.PersistentAuth) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token [HOST]",
		Short: "Get authentication token",
	}

	var tokenTimeout time.Duration
	cmd.Flags().DurationVar(&tokenTimeout, "timeout", auth.DefaultTimeout,
		"Timeout for acquiring a token.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		var profileName string
		profileFlag := cmd.Flag("profile")
		if profileFlag != nil {
			profileName = profileFlag.Value.String()
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

		ctx, cancel := context.WithTimeout(ctx, tokenTimeout)
		defer cancel()
		t, err := persistentAuth.Load(ctx)
		if err != nil {
			return err
		}
		raw, err := json.MarshalIndent(t, "", "  ")
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(raw)
		return nil
	}

	return cmd
}
