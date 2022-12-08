package tokenmanagement

import (
	token_management "github.com/databricks/bricks/cmd/tokenmanagement/token-management"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "tokenmanagement",
	Short: `Enables administrators to get all tokens and delete tokens for other users.`,
	Long:  `Enables administrators to get all tokens and delete tokens for other users.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(token_management.Cmd)
}
