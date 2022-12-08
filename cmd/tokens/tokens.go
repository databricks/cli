package tokens

import (
	"github.com/databricks/bricks/cmd/tokens/tokens"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "tokens",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(tokens.Cmd)
}
