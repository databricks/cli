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
	}

	var perisistentAuth auth.PersistentAuth
	cmd.PersistentFlags().StringVar(&perisistentAuth.Host, "host", perisistentAuth.Host, "Databricks Host")
	cmd.PersistentFlags().StringVar(&perisistentAuth.AccountID, "account-id", perisistentAuth.AccountID, "Databricks Account ID")
	cmd.PersistentFlags().BoolVar(&perisistentAuth.BindPublicAddress, "bind-public", perisistentAuth.BindPublicAddress, "Allow OAUTH redirect to bind to all local IP addresses including public addresses (NOTE: this is less secure)")
	cmd.AddCommand(newEnvCommand())
	cmd.AddCommand(newLoginCommand(&perisistentAuth))
	cmd.AddCommand(newProfilesCommand())
	cmd.AddCommand(newTokenCommand(&perisistentAuth))
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
