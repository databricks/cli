package auth

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/credentials/oauth"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication related commands",
		Long: `Authentication related commands. For more information regarding how
authentication for the Databricks CLI and SDKs work please refer to the documentation
linked below.

AWS: https://docs.databricks.com/dev-tools/auth/index.html
Azure: https://learn.microsoft.com/azure/databricks/dev-tools/auth
GCP: https://docs.gcp.databricks.com/dev-tools/auth/index.html`,
	}

	var oauthArgument oauth.BasicOAuthArgument
	cmd.PersistentFlags().StringVar(&oauthArgument.Host, "host", oauthArgument.Host, "Databricks Host")
	cmd.PersistentFlags().StringVar(&oauthArgument.AccountID, "account-id", oauthArgument.AccountID, "Databricks Account ID")

	cmd.AddCommand(newEnvCommand())
	cmd.AddCommand(newLoginCommand(&oauthArgument))
	cmd.AddCommand(newProfilesCommand())
	cmd.AddCommand(newTokenCommand(&oauthArgument))
	cmd.AddCommand(newDescribeCommand())
	return cmd
}

func promptForHost(ctx context.Context) (string, error) {
	if !cmdio.IsInTTY(ctx) {
		return "", fmt.Errorf("the command is being run in a non-interactive environment, please specify a host using --host")
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks host (e.g. https://<databricks-instance>.cloud.databricks.com)"
	return prompt.Run()
}

func promptForAccountID(ctx context.Context) (string, error) {
	if !cmdio.IsInTTY(ctx) {
		return "", fmt.Errorf("the command is being run in a non-interactive environment, please specify an account ID using --account-id")
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks account ID"
	prompt.Default = ""
	prompt.AllowEdit = true
	return prompt.Run()
}
