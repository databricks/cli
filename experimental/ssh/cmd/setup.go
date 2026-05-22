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
		Short: "[EXPERIMENTAL] Setup SSH configuration for dedicated (single-user) clusters",
		Long: `[EXPERIMENTAL] Setup SSH configuration for dedicated (single-user) clusters.

This is an experimental feature and is subject to change.

After running setup, you can connect with ` + "`ssh <name>`" + `.

For serverless connections, use ` + "`databricks ssh connect`" + ` (no setup step needed).`,
	}

	var hostName string
	var clusterID string
	var sshConfigPath string
	var shutdownDelay time.Duration
	var autoStartCluster bool
	var autoApprove bool

	cmd.Flags().StringVar(&hostName, "name", "", "Host name to use in SSH config")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&clusterID, "cluster", "", "Databricks cluster ID")
	cmd.Flags().BoolVar(&autoStartCluster, "auto-start-cluster", true, "Automatically start the cluster when establishing the ssh connection")
	cmd.Flags().StringVar(&sshConfigPath, "ssh-config", "", "Path to SSH config file (default ~/.ssh/config)")
	cmd.Flags().DurationVar(&shutdownDelay, "shutdown-delay", defaultShutdownDelay, "SSH server will terminate after this delay if there are no active connections")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompts, recreating existing SSH host configs without asking")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// We want to avoid the situation where the setup command works because it pulls the auth config from a bundle,
		// but later on the `ssh host-name` command fails when executed outside of the bundle directory.
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		wsClient := cmdctx.WorkspaceClient(ctx)
		setupOpts := setup.SetupOptions{
			HostName:         hostName,
			ClusterID:        clusterID,
			AutoStartCluster: autoStartCluster,
			SSHConfigPath:    sshConfigPath,
			ShutdownDelay:    shutdownDelay,
			Profile:          wsClient.Config.Profile,
			AutoApprove:      autoApprove,
		}
		return setup.Setup(ctx, wsClient, setupOpts)
	}

	return cmd
}
