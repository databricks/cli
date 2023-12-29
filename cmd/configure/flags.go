package configure

import (
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
