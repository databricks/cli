package auth

import (
	"context"
	"time"

	"github.com/databricks/cli/libs/auth"
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
		return perisistentAuth.Challenge(ctx)
	},
}

func init() {
	authCmd.AddCommand(loginCmd)
	loginCmd.Flags().DurationVar(&loginTimeout, "timeout", auth.DefaultTimeout,
		"Timeout for completing login challenge in the browser")
}
