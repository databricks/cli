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
	var secretsScope string
	var publicKeySecretName string

	cmd.Flags().StringVar(&clusterID, "cluster", "", "Databricks cluster ID")
	cmd.MarkFlagRequired("cluster")
	cmd.Flags().StringVar(&secretsScope, "secrets-scope-name", "", "Databricks secrets scope name")
	cmd.MarkFlagRequired("secrets-scope-name")
	cmd.Flags().StringVar(&publicKeySecretName, "client-key-name", "", "Databricks client key name")
	cmd.MarkFlagRequired("client-key-name")

	cmd.Flags().IntVar(&maxClients, "max-clients", defaultMaxClients, "Maximum number of SSH clients")
	cmd.Flags().DurationVar(&shutdownDelay, "shutdown-delay", defaultShutdownDelay, "Delay before shutting down after no pings from clients")
	cmd.Flags().StringVar(&version, "version", "", "Client version of the Databricks CLI")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := cmdctx.WorkspaceClient(ctx)
		opts := server.ServerOptions{
			ClusterID:            clusterID,
			MaxClients:           maxClients,
			ShutdownDelay:        shutdownDelay,
			Version:              version,
			ConfigDir:            serverConfigDir,
			SecretsScope:         secretsScope,
			ClientPublicKeyName:  publicKeySecretName,
			ServerPrivateKeyName: serverPrivateKeyName,
			ServerPublicKeyName:  serverPublicKeyName,
			DefaultPort:          defaultServerPort,
			PortRange:            serverPortRange,
		}
		return server.RunServer(ctx, client, opts)
	}

	return cmd
}
