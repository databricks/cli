package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
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
	cmd.PersistentFlags().BoolVar(&authArguments.IsUnifiedHost, "experimental-is-unified-host", false, "Flag to indicate if the host is a unified host")
	cmd.PersistentFlags().StringVar(&authArguments.WorkspaceID, "workspace-id", "", "Databricks Workspace ID")

	cmd.AddCommand(newEnvCommand())
	cmd.AddCommand(newLoginCommand(&authArguments))
	cmd.AddCommand(newLogoutCommand())
	cmd.AddCommand(newProfilesCommand())
	cmd.AddCommand(newTokenCommand(&authArguments))
	cmd.AddCommand(newDescribeCommand())
	cmd.AddCommand(newSwitchCommand())
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

// validateProfileHostConflict checks that --profile and --host don't conflict.
// If the profile's host matches the provided host (after canonicalization),
// the flags are considered compatible. If the profile is not found or has no
// host, the check is skipped (let the downstream command handle it).
func validateProfileHostConflict(ctx context.Context, profileName, host string, profiler profile.Profiler) error {
	p, err := loadProfileByName(ctx, profileName, profiler)
	if err != nil {
		return err
	}
	if p == nil || p.Host == "" {
		return nil
	}

	profileHost := (&config.Config{Host: p.Host}).CanonicalHostName()
	flagHost := (&config.Config{Host: host}).CanonicalHostName()

	if profileHost != flagHost {
		return fmt.Errorf(
			"--profile %q has host %q, which conflicts with --host %q. Use --profile only to select a profile",
			profileName, p.Host, host,
		)
	}
	return nil
}

// profileHostConflictCheck is a PreRunE function that validates
// --profile and --host don't conflict.
func profileHostConflictCheck(cmd *cobra.Command, args []string) error {
	profileFlag := cmd.Flag("profile")
	hostFlag := cmd.Flag("host")

	// Only validate when both flags are explicitly set by the user.
	if profileFlag == nil || hostFlag == nil {
		return nil
	}
	if !profileFlag.Changed || !hostFlag.Changed {
		return nil
	}

	return validateProfileHostConflict(
		cmd.Context(),
		profileFlag.Value.String(),
		hostFlag.Value.String(),
		profile.DefaultProfiler,
	)
}
