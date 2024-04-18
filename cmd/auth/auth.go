package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/google/uuid"
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
	if !cmdio.IsInTTY(ctx) {
		return "", fmt.Errorf("the command is being run in a non-interactive environment, please specify a host using --host")
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks Host"
	prompt.Validate = func(host string) error {
		if !strings.HasPrefix(host, "https://") {
			return fmt.Errorf("host URL must have a https:// prefix")
		}
		return nil
	}
	return prompt.Run()
}

func promptForAccountID(ctx context.Context) (string, error) {
	if !cmdio.IsInTTY(ctx) {
		return "", fmt.Errorf("the command is being run in a non-interactive environment, please specify an account ID using --account-id")
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks Account ID"
	prompt.Default = ""
	prompt.AllowEdit = true
	prompt.Validate = func(accountID string) error {
		_, err := uuid.Parse(accountID)
		if err != nil {
			return fmt.Errorf("account ID must be a valid UUID: %w", err)
		}
		return nil
	}
	return prompt.Run()
}
