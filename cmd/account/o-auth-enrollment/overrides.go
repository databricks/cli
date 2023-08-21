package o_auth_enrollment

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/oauth2"
	"github.com/spf13/cobra"
)

func promptForBasicAccountConfig(ctx context.Context) (*databricks.Config, error) {
	if !cmdio.IsInTTY(ctx) {
		return nil, fmt.Errorf("this command requires a TTY")
	}
	// OAuth Enrollment only works on AWS
	host, err := cmdio.DefaultPrompt(ctx, "Host", "https://accounts.cloud.databricks.com")
	if err != nil {
		return nil, fmt.Errorf("host: %w", err)
	}
	accountID, err := cmdio.SimplePrompt(ctx, "Account ID")
	if err != nil {
		return nil, fmt.Errorf("account: %w", err)
	}
	username, err := cmdio.SimplePrompt(ctx, "Username")
	if err != nil {
		return nil, fmt.Errorf("username: %w", err)
	}
	password, err := cmdio.Secret(ctx, "Password")
	if err != nil {
		return nil, fmt.Errorf("password: %w", err)
	}
	return &databricks.Config{
		Host:      host,
		AccountID: accountID,
		Username:  username,
		Password:  password,
	}, nil
}

func enableOAuthForAccount(ctx context.Context, cfg *databricks.Config) error {
	ac, err := databricks.NewAccountClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to instantiate account client: %w", err)
	}
	// The enrollment is executed asynchronously, so the API returns HTTP 204 immediately
	err = ac.OAuthEnrollment.Create(ctx, oauth2.CreateOAuthEnrollment{
		EnableAllPublishedApps: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create oauth enrollment: %w", err)
	}
	enableSpinner := cmdio.Spinner(ctx)
	// The actual enrollment take a few minutes
	err = retries.Wait(ctx, 10*time.Minute, func() *retries.Err {
		status, err := ac.OAuthEnrollment.Get(ctx)
		if err != nil {
			return retries.Halt(err)
		}
		if !status.IsEnabled {
			msg := "Enabling OAuth..."
			enableSpinner <- msg
			return retries.Continues(msg)
		}
		enableSpinner <- "OAuth is enabled"
		close(enableSpinner)
		return nil
	})
	if err != nil {
		return fmt.Errorf("wait for enrollment: %w", err)
	}
	// enable Databricks CLI, so that `databricks auth login` works
	_, err = ac.PublishedAppIntegration.Create(ctx, oauth2.CreatePublishedAppIntegration{
		AppId: "databricks-cli",
	})
	if err != nil {
		return fmt.Errorf("failed to enable databricks CLI: %w", err)
	}
	return nil
}

func newEnable() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable Databricks CLI, Tableau Desktop, and PowerBI for this account.",
		Long: `Before you can do 'databricks auth login', you have to enable OAuth for this account.

This command prompts you for Account ID, username, and password and waits until OAuth is enabled.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := promptForBasicAccountConfig(ctx)
			if err != nil {
				return fmt.Errorf("account config: %w", err)
			}
			return enableOAuthForAccount(ctx, cfg)
		},
	}
}

func init() {
	cmdOverrides = append(cmdOverrides, func(c *cobra.Command) {
		c.AddCommand(newEnable())
	})
}
