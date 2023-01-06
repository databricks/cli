package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/databricks/bricks/libs/auth"
	"github.com/spf13/cobra"
)

var tokenTimeout time.Duration

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Get authentication token",
	RunE: func(cmd *cobra.Command, args []string) error {
		defer perisistentAuth.Close()
		ctx, cancel := context.WithTimeout(cmd.Context(), tokenTimeout)
		defer cancel()
		t, err := perisistentAuth.Load(ctx)
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
