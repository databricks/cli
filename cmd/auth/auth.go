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

	var persistentAuth auth.PersistentAuth
	cmd.PersistentFlags().StringVar(&persistentAuth.Host, "host", persistentAuth.Host, "Databricks Host")
	cmd.PersistentFlags().StringVar(&persistentAuth.AccountID, "account-id", persistentAuth.AccountID, "Databricks Account ID")

	hidden := false
	cmd.AddCommand(newEnvCommand())
	cmd.AddCommand(newLoginCommand(hidden, &persistentAuth))
	cmd.AddCommand(newProfilesCommand())
	cmd.AddCommand(newTokenCommand(&persistentAuth))
	cmd.AddCommand(newDescribeCommand())
	return cmd
}

// NewTopLevelLoginCommand creates a new login command for use in a top-level command group.
// This is useful for custom CLIs where the 'auth' command group does not exist.
func NewTopLevelLoginCommand(hidden bool) *cobra.Command {
	var persistentAuth auth.PersistentAuth
	cmd := newLoginCommand(hidden, &persistentAuth)
	cmd.Flags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.Flags().StringVar(&persistentAuth.Host, "host", persistentAuth.Host, "Databricks Host")
	cmd.Flags().StringVar(&persistentAuth.AccountID, "account-id", persistentAuth.AccountID, "Databricks Account ID")
	return cmd
}

func promptForHost(ctx context.Context) (string, error) {
	if !cmdio.IsInTTY(ctx) {
		return "", errors.New("the command is being run in a non-interactive environment, please specify a host using --host")
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks host (e.g. https://<databricks-instance>.cloud.databricks.com)"
	return prompt.Run()
}

func promptForAccountID(ctx context.Context) (string, error) {
	if !cmdio.IsInTTY(ctx) {
		return "", errors.New("the command is being run in a non-interactive environment, please specify an account ID using --account-id")
	}

	prompt := cmdio.Prompt(ctx)
	prompt.Label = "Databricks account ID"
	prompt.Default = ""
	prompt.AllowEdit = true
	return prompt.Run()
}
