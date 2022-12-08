package secrets

import (
	"github.com/databricks/bricks/cmd/secrets/secrets"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "secrets",
	Short: `The Secrets API allows you to manage secrets, secret scopes, and access permissions.`,
	Long: `The Secrets API allows you to manage secrets, secret scopes, and access
  permissions.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(secrets.Cmd)
}
