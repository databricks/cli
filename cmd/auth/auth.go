package auth

import (
	"context"
	"errors"

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

AWS: https://docs.databricks.com/dev-tools/auth/index.html
Azure: https://learn.microsoft.com/azure/databricks/dev-tools/auth
GCP: https://docs.gcp.databricks.com/dev-tools/auth/index.html`,
	}

	var authArguments auth.AuthArguments
	cmd.PersistentFlags().StringVar(&authArguments.Host, "host", "", "Databricks Host")
	cmd.PersistentFlags().StringVar(&authArguments.AccountID, "account-id", "", "Databricks Account ID")
	cmd.PersistentFlags().BoolVarP(&authArguments.IsUnifiedHost, "experimental-is-unified-host", "u", false, "Whether the host is a unified host")

	cmd.AddCommand(newEnvCommand())
	cmd.AddCommand(newLoginCommand(&authArguments))
	cmd.AddCommand(newProfilesCommand())
	cmd.AddCommand(newTokenCommand(&authArguments))
	cmd.AddCommand(newDescribeCommand())
	return cmd
}

func promptForHost(ctx context.Context) (string, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", errors.New("the command is being run in a non-interactive environment, please specify a host using --host")
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks host (e.g. https://<databricks-instance>.cloud.databricks.com)"
	return prompt.Run()
}

func promptForAccountID(ctx context.Context) (string, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", errors.New("the command is being run in a non-interactive environment, please specify an account ID using --account-id")
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks account ID"
	prompt.Default = ""
	prompt.AllowEdit = true
	return prompt.Run()
}

func promptForWorkspaceId(ctx context.Context) (string, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", nil
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks workspace ID (optional - provide only if using this profile for workspace operations, leave empty for account operations)"
	prompt.Default = ""
	prompt.AllowEdit = true
	result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}
