package ssh

import (
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/ssh"
	"github.com/spf13/cobra"
)

func newSetupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup SSH configuration for a Databricks compute",
		Long: `Setup SSH configuration for a Databricks compute.

This command configures SSH to connect to a Databricks compute by adding
an SSH host configuration to your SSH config file.

` + disclaimer,
	}

	var hostName string
	var clusterID string
	var sshConfigPath string
	var shutdownDelay time.Duration

	cmd.Flags().StringVar(&hostName, "name", "", "Host name to use in SSH config")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&clusterID, "cluster", "", "Databricks cluster ID")
	cmd.MarkFlagRequired("cluster")
	cmd.Flags().StringVar(&sshConfigPath, "ssh-config", "", "Path to SSH config file (default ~/.ssh/config)")
	cmd.Flags().DurationVar(&shutdownDelay, "shutdown-delay", 10*time.Minute, "SSH server will terminate after this delay if there are no active connections")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := cmdctx.WorkspaceClient(ctx)
		opts := ssh.SetupOptions{
			HostName:      hostName,
			ClusterID:     clusterID,
			SSHConfigPath: sshConfigPath,
			ShutdownDelay: shutdownDelay,
			Profile:       client.Config.Profile,
		}
		return ssh.Setup(ctx, client, opts)
	}

	return cmd
}
