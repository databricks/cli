package configure

import (
	"context"
	"fmt"
	"net/url"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

func validateHost(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	if u.Host == "" || u.Scheme != "https" {
		return fmt.Errorf("must start with https://")
	}
	if u.Path != "" && u.Path != "/" {
		return fmt.Errorf("must use empty path")
	}
	return nil
}

func configureFromFlags(cmd *cobra.Command, ctx context.Context, cfg *config.Config) error {
	// Configure profile name if set.
	profile, err := cmd.Flags().GetString("profile")
	if err != nil {
		return fmt.Errorf("read --profile flag: %w", err)
	}
	if profile != "" {
		cfg.Profile = profile
	}

	// Configure host if set.
	host, err := cmd.Flags().GetString("host")
	if err != nil {
		return fmt.Errorf("read --host flag: %w", err)
	}
	if host != "" {
		cfg.Host = host
	}

	// Validate host if set.
	if cfg.Host != "" {
		err = validateHost(cfg.Host)
		if err != nil {
			return err
		}
	}

	return nil
}

func configureInteractive(cmd *cobra.Command, ctx context.Context, cfg *config.Config) error {
	err := configureFromFlags(cmd, ctx, cfg)
	if err != nil {
		return err
	}

	// Ask user to specify the host if not already set.
	if cfg.Host == "" {
		prompt := cmdio.Prompt(ctx)
		prompt.Label = "Databricks Host"
		prompt.Default = "https://"
		prompt.AllowEdit = true
		prompt.Validate = validateHost
		out, err := prompt.Run()
		if err != nil {
			return err
		}
		cfg.Host = out
	}

	// Ask user to specify the token is not already set.
	if cfg.Token == "" {
		prompt := cmdio.Prompt(ctx)
		prompt.Label = "Personal Access Token"
		prompt.Mask = '*'
		out, err := prompt.Run()
		if err != nil {
			return err
		}
		cfg.Token = out
	}

	return nil
}

func configureNonInteractive(cmd *cobra.Command, ctx context.Context, cfg *config.Config) error {
	err := configureFromFlags(cmd, ctx, cfg)
	if err != nil {
		return err
	}

	if cfg.Host == "" {
		return fmt.Errorf("host must be set in non-interactive mode")
	}

	// Read token from stdin if not already set.
	if cfg.Token == "" {
		_, err := fmt.Fscanf(cmd.InOrStdin(), "%s\n", &cfg.Token)
		if err != nil {
			return err
		}
	}

	return nil
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure authentication",
	Long: `Configure authentication.

This command adds a profile to your ~/.databrickscfg file.
You can write to a different file by setting the DATABRICKS_CONFIG_FILE environment variable.

If this command is invoked in non-interactive mode, it will read the token from stdin.
The host must be specified with the --host flag.
	`,
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfg config.Config

		// Load environment variables, possibly the DEFAULT profile.
		err := config.ConfigAttributes.Configure(&cfg)
		if err != nil {
			return fmt.Errorf("unable to instantiate configuration from environment variables: %w", err)
		}

		ctx := cmd.Context()
		interactive := cmdio.IsInTTY(ctx) && cmdio.IsOutTTY(ctx)
		var fn func(*cobra.Command, context.Context, *config.Config) error
		if interactive {
			fn = configureInteractive
		} else {
			fn = configureNonInteractive
		}
		err = fn(cmd, ctx, &cfg)
		if err != nil {
			return err
		}

		// Clear the Databricks CLI path in token mode.
		// This is relevant for OAuth only.
		cfg.DatabricksCliPath = ""

		// Save profile to config file.
		return databrickscfg.SaveToProfile(ctx, &cfg)
	},
}

func init() {
	root.RootCmd.AddCommand(configureCmd)
	configureCmd.Flags().String("host", "", "Databricks workspace host.")
	configureCmd.Flags().String("profile", "DEFAULT", "Name for the connection profile to configure.")

	// Include token flag for compatibility with the legacy CLI.
	// It doesn't actually do anything because we always use PATs.
	configureCmd.Flags().BoolP("token", "t", true, "Configure using Databricks Personal Access Token")
	configureCmd.Flags().MarkHidden("token")
}
