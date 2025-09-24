package ssh

import (
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/ssh/internal/setup"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

func newSetupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup SSH configuration for Databricks compute",
		Long: `Setup SSH configuration for Databricks compute.

This command configures SSH to connect to Databricks compute by adding
an SSH host configuration to your SSH config file.

` + disclaimer,
	}

	var hostName string
	var clusterID string
	var sshConfigPath string
	var shutdownDelay time.Duration
	var autoStartCluster bool

	cmd.Flags().StringVar(&hostName, "name", "", "Host name to use in SSH config")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&clusterID, "cluster", "", "Databricks cluster ID")
	cmd.Flags().BoolVar(&autoStartCluster, "auto-start-cluster", true, "Automatically start the cluster when establishing the ssh connection")
	cmd.Flags().StringVar(&sshConfigPath, "ssh-config", "", "Path to SSH config file (default ~/.ssh/config)")
	cmd.Flags().DurationVar(&shutdownDelay, "shutdown-delay", defaultShutdownDelay, "SSH server will terminate after this delay if there are no active connections")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := cmdctx.WorkspaceClient(ctx)
		opts := setup.SetupOptions{
			HostName:         hostName,
			ClusterID:        clusterID,
			AutoStartCluster: autoStartCluster,
			SSHConfigPath:    sshConfigPath,
			ShutdownDelay:    shutdownDelay,
			Profile:          client.Config.Profile,
		}
		return setup.Setup(ctx, client, opts)
	}

	return cmd
}
