package ssh

import (
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/ssh"
	"github.com/spf13/cobra"
)

func newServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run SSH tunnel server",
		Long: `Run SSH tunnel server.

This command starts an SSH tunnel server that accepts WebSocket connections
and proxies them to local SSH daemon processes.

` + disclaimer,
		// This is an internal command spawned by the SSH client running the "ssh-server-bootstrap.py" job
		Hidden: true,
	}

	var maxClients int
	var shutdownDelay time.Duration
	var clusterID string
	var version string

	cmd.Flags().StringVar(&clusterID, "cluster", "", "Databricks cluster ID")
	cmd.MarkFlagRequired("cluster")
	cmd.Flags().IntVar(&maxClients, "max-clients", 10, "Maximum number of SSH clients")
	cmd.Flags().DurationVar(&shutdownDelay, "shutdown-delay", 10*time.Minute, "Delay before shutting down after no pings from clients")
	cmd.Flags().StringVar(&version, "version", "", "Client version of the Databricks CLI")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := cmdctx.WorkspaceClient(ctx)
		opts := ssh.ServerOptions{
			ClusterID:            clusterID,
			MaxClients:           maxClients,
			ShutdownDelay:        shutdownDelay,
			Version:              version,
			ConfigDir:            ".ssh-tunnel",
			ServerPrivateKeyName: "server-private-key",
			ServerPublicKeyName:  "server-public-key",
			DefaultPort:          7772,
			PortRange:            100,
		}
		err := ssh.RunServer(ctx, client, opts)
		if err != nil && ssh.IsNormalClosure(err) {
			return nil
		}
		return err
	}

	return cmd
}
