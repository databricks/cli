package auth

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication related commands",
}

var perisistentAuth auth.PersistentAuth

func init() {
	root.RootCmd.AddCommand(authCmd)
	authCmd.PersistentFlags().StringVar(&perisistentAuth.Host, "host", perisistentAuth.Host, "Databricks Host")
	authCmd.PersistentFlags().StringVar(&perisistentAuth.AccountID, "account-id", perisistentAuth.AccountID, "Databricks Account ID")
}
