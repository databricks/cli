package auth

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication related commands",
}

var persistentAuth auth.PersistentAuth

func promptForHost(ctx context.Context) (string, error) {
	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks Host"
	prompt.Default = "https://"
	prompt.AllowEdit = true
	// Validate?
	host, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return host, nil
}

func promptForAccountID(ctx context.Context) (string, error) {
	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks Account ID"
	prompt.Default = ""
	prompt.AllowEdit = true
	// Validate?
	accountId, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return accountId, nil
}

func init() {
	root.RootCmd.AddCommand(authCmd)
	authCmd.PersistentFlags().StringVar(&persistentAuth.Host, "host", persistentAuth.Host, "Databricks Host")
	authCmd.PersistentFlags().StringVar(&persistentAuth.AccountID, "account-id", persistentAuth.AccountID, "Databricks Account ID")
}
