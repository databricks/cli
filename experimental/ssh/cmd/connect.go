package ssh

import (
	"errors"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/ssh/internal/client"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
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
	var connectionName string
	var accelerator string
	var proxyMode bool
	var ide string
	var serverMetadata string
	var shutdownDelay time.Duration
	var maxClients int
	var handoverTimeout time.Duration
	var releasesDir string
	var autoStartCluster bool
	var userKnownHostsFile string
	var liteswap string
	var skipSettingsCheck bool

	cmd.Flags().StringVar(&clusterID, "cluster", "", "Databricks cluster ID (for dedicated clusters)")
	cmd.Flags().DurationVar(&shutdownDelay, "shutdown-delay", defaultShutdownDelay, "Delay before shutting down the server after the last client disconnects")
	cmd.Flags().IntVar(&maxClients, "max-clients", defaultMaxClients, "Maximum number of SSH clients")
	cmd.Flags().BoolVar(&autoStartCluster, "auto-start-cluster", true, "Automatically start the cluster if it is not running")

	cmd.Flags().StringVar(&connectionName, "name", "", "Connection name (for serverless compute)")
	cmd.Flags().MarkHidden("name")
	cmd.Flags().StringVar(&accelerator, "accelerator", "", "GPU accelerator type (GPU_1xA10 or GPU_8xH100)")
	cmd.Flags().MarkHidden("accelerator")
	cmd.Flags().StringVar(&ide, "ide", "", "Open remote IDE window (vscode or cursor)")
	cmd.Flags().MarkHidden("ide")

	cmd.Flags().BoolVar(&proxyMode, "proxy", false, "ProxyCommand mode")
	cmd.Flags().MarkHidden("proxy")
	cmd.Flags().StringVar(&serverMetadata, "metadata", "", "Metadata of the running SSH server (format: <user_name>,<port>)")
	cmd.Flags().MarkHidden("metadata")
	cmd.Flags().DurationVar(&handoverTimeout, "handover-timeout", defaultHandoverTimeout, "How often the CLI should reconnect to the server with new auth")
	cmd.Flags().MarkHidden("handover-timeout")

	cmd.Flags().StringVar(&releasesDir, "releases-dir", "", "Directory for local SSH tunnel development releases")
	cmd.Flags().MarkHidden("releases-dir")

	cmd.Flags().StringVar(&userKnownHostsFile, "user-known-hosts-file", "", "Path to user known hosts file for SSH client")
	cmd.Flags().MarkHidden("user-known-hosts-file")

	cmd.Flags().StringVar(&liteswap, "liteswap", "", "Liteswap header value for traffic routing (dev/test only)")
	cmd.Flags().MarkHidden("liteswap")

	cmd.Flags().BoolVar(&skipSettingsCheck, "skip-settings-check", false, "Skip checking and updating IDE settings")
	cmd.Flags().MarkHidden("skip-settings-check")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// CLI in the proxy mode is executed by the ssh client and can't prompt for input
		if proxyMode {
			cmd.SetContext(root.SkipPrompt(cmd.Context()))
		}
		// We want to avoid the situation where the connect command works because it pulls the auth config from a bundle,
		// but fails if it's executed outside of it (which will happen when using remote development IDE features).
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		wsClient := cmdctx.WorkspaceClient(ctx)

		if !proxyMode && clusterID == "" && connectionName == "" {
			return errors.New("please provide --cluster flag with the cluster ID, or --name flag with the connection name (for serverless compute)")
		}

		if accelerator != "" && connectionName == "" {
			return errors.New("--accelerator flag can only be used with serverless compute (--name flag)")
		}

		// Remove when we add support for serverless CPU
		if connectionName != "" && accelerator == "" {
			return errors.New("--name flag requires --accelerator to be set (for now we only support serverless GPU compute)")
		}

		// TODO: validate connectionName if provided

		opts := client.ClientOptions{
			Profile:              wsClient.Config.Profile,
			ClusterID:            clusterID,
			ConnectionName:       connectionName,
			Accelerator:          accelerator,
			ProxyMode:            proxyMode,
			IDE:                  ide,
			ServerMetadata:       serverMetadata,
			ShutdownDelay:        shutdownDelay,
			MaxClients:           maxClients,
			HandoverTimeout:      handoverTimeout,
			ReleasesDir:          releasesDir,
			ServerTimeout:        max(serverTimeout, shutdownDelay),
			TaskStartupTimeout:   taskStartupTimeout,
			AutoStartCluster:     autoStartCluster,
			ClientPublicKeyName:  clientPublicKeyName,
			ClientPrivateKeyName: clientPrivateKeyName,
			UserKnownHostsFile:   userKnownHostsFile,
			Liteswap:             liteswap,
			SkipSettingsCheck:    skipSettingsCheck,
			AdditionalArgs:       args,
		}
		return client.Run(ctx, wsClient, opts)
	}

	return cmd
}
