package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/databricks/cli/libs/auth"
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
			persistentAuth.Profile = profileFlag.Value.String()
			// If a profile is provided we read the host from the .databrickscfg file
			if profileName != "" && len(args) > 0 {
				return errors.New("providing both a profile and a hostname is not supported")
			}
		}

		err := setHost(ctx, persistentAuth, args)
		if err != nil {
			return err
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
