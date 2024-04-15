package auth

import (
	"context"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication related commands",
		Long: `Authentication related commands. For more information regarding how
authentication for the Databricks CLI and SDKs work please refer to the documentation
linked below.

AWS: https://docs.databricks.com/en/dev-tools/auth/index.html
Azure: https://learn.microsoft.com/en-us/azure/databricks/dev-tools/auth
GCP: https://docs.gcp.databricks.com/en/dev-tools/auth/index.html`,
	}

	var perisistentAuth auth.PersistentAuth
	cmd.PersistentFlags().StringVar(&perisistentAuth.Host, "host", perisistentAuth.Host, "Databricks Host")
	cmd.PersistentFlags().StringVar(&perisistentAuth.AccountID, "account-id", perisistentAuth.AccountID, "Databricks Account ID")

	cmd.AddCommand(newEnvCommand())
	cmd.AddCommand(newLoginCommand(&perisistentAuth))
	cmd.AddCommand(newProfilesCommand())
	cmd.AddCommand(newTokenCommand(&perisistentAuth))
	cmd.AddCommand(newDescribeCommand())
	return cmd
}

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
