package auth

import (
	"github.com/databricks/bricks/libs/auth"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login HOST",
	Args:  cobra.ExactArgs(1),
	Short: "Authenticate this machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		u2m, err := auth.NewPersistentOAuth(args[0])
		if err != nil {
			return err
		}
		return u2m.Challenge(cmd.Context())
	},
}

func init() {
	authCmd.AddCommand(loginCmd)
}
