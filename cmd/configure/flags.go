package configure

import (
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

type configureFlags struct {
	Host    string
	Profile string

	// Flag to request a prompt for cluster configuration.
	ConfigureCluster bool
}

// Register flags with command.
func (f *configureFlags) Register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Host, "host", "", "Databricks workspace host.")
	cmd.Flags().StringVar(&f.Profile, "profile", "DEFAULT", "Name for the connection profile to configure.")
	cmd.Flags().BoolVar(&f.ConfigureCluster, "configure-cluster", false, "Prompts to configure cluster")

	// Include token flag for compatibility with the legacy CLI.
	// It doesn't actually do anything because we always use PATs.
	cmd.Flags().Bool("token", true, "Configure using Databricks Personal Access Token")
	cmd.Flags().MarkHidden("token")
}

func (f *configureFlags) PopulateConfig(cfg *config.Config) error {
	if f.Host != "" {
		cfg.Host = f.Host
	}

	if f.Profile != "" {
		cfg.Profile = f.Profile
	}

	return nil
}
