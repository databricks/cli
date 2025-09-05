package ssh

import (
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/ssh"
	"github.com/spf13/cobra"
)

const (
	defaultClientPublicKeyName = "client-public-key"
	defaultShutdownDelay       = 10 * time.Minute
	defaultHandoverTimeout     = 30 * time.Minute
	defaultMaxClients          = 10
)

func newConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to Databricks compute via SSH",
		Long: `Connect to Databricks compute via SSH.

This command establishes an SSH connection to Databricks compute, setting up
the SSH server and handling the connection proxy.

` + disclaimer,
	}

	var clusterID string
	var proxyMode bool
	var serverMetadata string
	var shutdownDelay time.Duration
	var maxClients int
	var handoverTimeout time.Duration
	var releasesDir string

	cmd.Flags().StringVar(&clusterID, "cluster", "", "Databricks cluster ID (required)")
	cmd.MarkFlagRequired("cluster")
	cmd.Flags().DurationVar(&shutdownDelay, "shutdown-delay", defaultShutdownDelay, "Delay before shutting down the server after the last client disconnects")
	cmd.Flags().IntVar(&maxClients, "max-clients", defaultMaxClients, "Maximum number of SSH clients")

	cmd.Flags().BoolVar(&proxyMode, "proxy", false, "ProxyCommand mode")
	cmd.Flags().MarkHidden("proxy")
	cmd.Flags().StringVar(&serverMetadata, "metadata", "", "Metadata of the running SSH server (format: <user_name>,<port>)")
	cmd.Flags().MarkHidden("metadata")
	cmd.Flags().DurationVar(&handoverTimeout, "handover-timeout", defaultHandoverTimeout, "How often the CLI should reconnect to the server with new auth")
	cmd.Flags().MarkHidden("handover-timeout")

	cmd.Flags().StringVar(&releasesDir, "releases-dir", "", "Directory for local SSH tunnel development releases")
	cmd.Flags().MarkHidden("releases-dir")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := cmdctx.WorkspaceClient(ctx)
		opts := ssh.ClientOptions{
			ClusterID:           clusterID,
			ProxyMode:           proxyMode,
			ServerMetadata:      serverMetadata,
			ShutdownDelay:       shutdownDelay,
			MaxClients:          maxClients,
			HandoverTimeout:     handoverTimeout,
			ReleasesDir:         releasesDir,
			AdditionalArgs:      args,
			ClientPublicKeyName: defaultClientPublicKeyName,
		}
		return ssh.RunClient(ctx, client, opts)
	}

	return cmd
}
