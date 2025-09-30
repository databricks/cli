package configure

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/cfgpickers"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

func configureInteractive(cmd *cobra.Command, flags *configureFlags, cfg *config.Config) error {
	ctx := cmd.Context()

	// Ask user to specify the host if not already set.
	if cfg.Host == "" {
		prompt := cmdio.Prompt(ctx)
		prompt.Label = "Databricks host"
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
		prompt.Label = "Personal access token"
		prompt.Mask = '*'
		out, err := prompt.Run()
		if err != nil {
			return err
		}
		cfg.Token = out
	}

	// Ask user to specify a cluster if not already set.
	if flags.ConfigureCluster && cfg.ClusterID == "" {
		// Create workspace client with configuration without the profile name set.
		w, err := databricks.NewWorkspaceClient(&databricks.Config{
			Host:  cfg.Host,
			Token: cfg.Token,
		})
		if err != nil {
			return err
		}
		clusterID, err := cfgpickers.AskForCluster(cmd.Context(), w, cfgpickers.WithoutSystemClusters())
		if err != nil {
			return err
		}
		cfg.ClusterID = clusterID
	}

	return nil
}

func configureNonInteractive(cmd *cobra.Command, flags *configureFlags, cfg *config.Config) error {
	if cfg.Host == "" {
		return errors.New("host must be set in non-interactive mode")
	}

	// Check presence of cluster ID before reading token to fail fast.
	if flags.ConfigureCluster && cfg.ClusterID == "" {
		return errors.New("cluster ID must be set in non-interactive mode")
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

func newConfigureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure authentication",
		Long: `Configure authentication.

This command adds a profile to your ~/.databrickscfg file.
You can write to a different file by setting the DATABRICKS_CONFIG_FILE environment variable.

If this command is invoked in non-interactive mode, it will read the token from stdin.
The host must be specified with the --host flag or the DATABRICKS_HOST environment variable.
		`,
	}

	var flags configureFlags
	flags.Register(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var cfg config.Config

		// Load environment variables, possibly the DEFAULT profile.
		err := config.ConfigAttributes.Configure(&cfg)
		if err != nil {
			return fmt.Errorf("unable to instantiate configuration from environment variables: %w", err)
		}

		// Populate configuration from flags (if set).
		if flags.Host != "" {
			cfg.Host = flags.Host
		}
		if flags.Profile != "" {
			cfg.Profile = flags.Profile
		}

		// Verify that the host is valid (if set).
		if cfg.Host != "" {
			err = validateHost(cfg.Host)
			if err != nil {
				return err
			}
		}

		ctx := cmd.Context()
		if cmdio.IsInTTY(ctx) && cmdio.IsOutTTY(ctx) {
			err = configureInteractive(cmd, &flags, &cfg)
		} else {
			err = configureNonInteractive(cmd, &flags, &cfg)
		}
		if err != nil {
			return err
		}

		// Clear the Databricks CLI path in token mode.
		// This is relevant for OAuth only.
		cfg.DatabricksCliPath = ""

		// Save profile to config file.
		return databrickscfg.SaveToProfile(ctx, &config.Config{
			Profile:    cfg.Profile,
			Host:       cfg.Host,
			Token:      cfg.Token,
			ClusterID:  cfg.ClusterID,
			ConfigFile: cfg.ConfigFile,
		})
	}

	return cmd
}

func New() *cobra.Command {
	return newConfigureCommand()
}
