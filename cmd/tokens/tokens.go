package tokens

import (
	"github.com/databricks/bricks/cmd/tokens/tokens"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "tokens",
	Short: `The Token API allows you to create, list, and revoke tokens that can be used to authenticate and access Databricks REST APIs.`,
	Long: `The Token API allows you to create, list, and revoke tokens that can be used
  to authenticate and access Databricks REST APIs.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(tokens.Cmd)
}
