package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/spf13/cobra"
)

var tokenTimeout time.Duration

var tokenCmd = &cobra.Command{
	Use:   "token [HOST]",
	Short: "Get authentication token",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if persistentAuth.Host == "" {
			configureHost(ctx, args, 0)
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
	},
}

func init() {
	authCmd.AddCommand(tokenCmd)
	tokenCmd.Flags().DurationVar(&tokenTimeout, "timeout", auth.DefaultTimeout,
		"Timeout for acquiring a token.")
}
