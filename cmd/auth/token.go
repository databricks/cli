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
	Use:   "token HOST",
	Args:  cobra.ExactArgs(1),
	Short: "Get authentication token",
	RunE: func(cmd *cobra.Command, args []string) error {
		u2m, err := auth.NewPersistentOAuth(args[0])
		if err != nil {
			return err
		}
		defer u2m.Close()
		ctx, cancel := context.WithTimeout(cmd.Context(), tokenTimeout)
		defer cancel()
		t, err := u2m.Load(ctx)
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
