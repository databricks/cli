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
	var secretScopeName string
	var authorizedKeySecretName string

	cmd.Flags().StringVar(&clusterID, "cluster", "", "Databricks cluster ID")
	cmd.MarkFlagRequired("cluster")
	cmd.Flags().StringVar(&secretScopeName, "secret-scope-name", "", "Databricks secret scope name to store SSH keys")
	cmd.MarkFlagRequired("secret-scope-name")
	cmd.Flags().StringVar(&authorizedKeySecretName, "authorized-key-secret-name", "", "Name of the secret containing the client public key")
	cmd.MarkFlagRequired("authorized-key-secret-name")

	cmd.Flags().IntVar(&maxClients, "max-clients", defaultMaxClients, "Maximum number of SSH clients")
	cmd.Flags().DurationVar(&shutdownDelay, "shutdown-delay", defaultShutdownDelay, "Delay before shutting down after no pings from clients")
	cmd.Flags().StringVar(&version, "version", "", "Client version of the Databricks CLI")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// The server can be executed under a directory with an invalid bundle configuration.
		// We do not want to error out in this case.
		// The auth is setup by the job logic that executes this command.
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		// The command should be executed in a non-interactive environment, but let's be explicit about no prompts.
		cmd.SetContext(root.SkipPrompt(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		wsc := cmdctx.WorkspaceClient(ctx)
		opts := server.ServerOptions{
			ClusterID:               clusterID,
			MaxClients:              maxClients,
			ShutdownDelay:           shutdownDelay,
			Version:                 version,
			ConfigDir:               serverConfigDir,
			SecretScopeName:         secretScopeName,
			ServerPrivateKeyName:    serverPrivateKeyName,
			ServerPublicKeyName:     serverPublicKeyName,
			AuthorizedKeySecretName: authorizedKeySecretName,
			DefaultPort:             defaultServerPort,
			PortRange:               serverPortRange,
		}
		return server.Run(ctx, wsc, opts)
	}

	return cmd
}
