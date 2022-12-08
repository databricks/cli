package secrets

import (
	"github.com/databricks/bricks/cmd/secrets/secrets"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "secrets",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(secrets.Cmd)
}
