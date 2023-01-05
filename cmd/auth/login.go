package auth

import (
	"context"
	"time"

	"github.com/databricks/bricks/libs/auth"
	"github.com/spf13/cobra"
)

var loginTimeout time.Duration

var loginCmd = &cobra.Command{
	Use:   "login HOST",
	Args:  cobra.ExactArgs(1),
	Short: "Authenticate this machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		u2m, err := auth.NewPersistentOAuth(args[0])
		if err != nil {
			return err
		}
		defer u2m.Close()
		ctx, cancel := context.WithTimeout(cmd.Context(), loginTimeout)
		defer cancel()
		return u2m.Challenge(ctx)
	},
}

func init() {
	authCmd.AddCommand(loginCmd)
	loginCmd.Flags().DurationVar(&loginTimeout, "timeout", auth.DefaultTimeout,
		"Timeout for completing login challenge in the browser")
}
