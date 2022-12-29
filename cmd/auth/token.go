package auth

import (
	"encoding/json"

	"github.com/databricks/bricks/libs/auth"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token HOST",
	Args:  cobra.ExactArgs(1),
	Short: "Get authentication token",
	RunE: func(cmd *cobra.Command, args []string) error {
		u2m, err := auth.NewPersistentOAuth(args[0])
		if err != nil {
			return err
		}
		t, err := u2m.Load(cmd.Context())
		if err != nil {
			return err
		}
		t.RefreshToken = ""
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
}
