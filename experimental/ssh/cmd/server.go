package ssh

import (
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/ssh/internal/server"
	"github.com/databricks/cli/libs/cmdctx"
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
	var keysSecretScopeName string
	var authorizedKeyName string

	cmd.Flags().StringVar(&clusterID, "cluster", "", "Databricks cluster ID")
	cmd.MarkFlagRequired("cluster")
	cmd.Flags().StringVar(&keysSecretScopeName, "keys-secret-scope-name", "", "Databricks secret scope name to store SSH keys")
	cmd.MarkFlagRequired("keys-secret-scope-name")
	cmd.Flags().StringVar(&authorizedKeyName, "authorized-key-secret-name", "", "Authorized key secret name in the secret scope")
	cmd.MarkFlagRequired("authorized-key-secret-name")

	cmd.Flags().IntVar(&maxClients, "max-clients", defaultMaxClients, "Maximum number of SSH clients")
	cmd.Flags().DurationVar(&shutdownDelay, "shutdown-delay", defaultShutdownDelay, "Delay before shutting down after no pings from clients")
	cmd.Flags().StringVar(&version, "version", "", "Client version of the Databricks CLI")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		wsc := cmdctx.WorkspaceClient(ctx)
		opts := server.ServerOptions{
			ClusterID:            clusterID,
			MaxClients:           maxClients,
			ShutdownDelay:        shutdownDelay,
			Version:              version,
			ConfigDir:            serverConfigDir,
			KeysSecretScopeName:  keysSecretScopeName,
			AuthorizedKeyName:    authorizedKeyName,
			ServerPrivateKeyName: serverPrivateKeyName,
			ServerPublicKeyName:  serverPublicKeyName,
			DefaultPort:          defaultServerPort,
			PortRange:            serverPortRange,
		}
		return server.Run(ctx, wsc, opts)
	}

	return cmd
}
