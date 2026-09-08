package ssh

import (
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/ssh/internal/client"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// resolveServerTimeout returns the lifetime to submit the SSH tunnel server job with.
//
// Before --server-timeout existed the lifetime was max(24h, --shutdown-delay), so
// `ssh setup --shutdown-delay=48h` produced a working 48h tunnel and persisted that delay into
// the generated ProxyCommand. Keep honoring it whenever --server-timeout is not set: OpenSSH
// runs a persisted ProxyCommand verbatim, so a user whose host config predates the flag cannot
// add it, and rejecting the pair would break `ssh <name>` with an error naming a flag they have
// no way to pass. An explicit --server-timeout always wins, and ClientOptions.Validate still
// rejects a shutdown delay that outlives a lifetime the user asked for explicitly.
func resolveServerTimeout(flags *pflag.FlagSet, serverTimeout, shutdownDelay time.Duration) time.Duration {
	if flags.Changed("server-timeout") {
		return serverTimeout
	}
	return max(serverTimeout, shutdownDelay)
}

func newConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to your Databricks compute and workspace via SSH",
		Long: `Connect to your Databricks compute and workspace via SSH.

Connect to serverless:
  databricks ssh connect
  databricks ssh connect --accelerator=<GPU_type>   # AI Runtime
  databricks ssh connect --base-environment=<name>  # custom base environment

Connect to a dedicated cluster:
  databricks ssh connect --cluster=<cluster-id>`,
	}

	var clusterID string
	var connectionName string
	var accelerator string
	var proxyMode bool
	var ide string
	var serverMetadata string
	var shutdownDelay time.Duration
	var maxClients int
	var serverTimeout time.Duration
	var handoverTimeout time.Duration
	var releasesDir string
	var autoStartCluster bool
	var userKnownHostsFile string
	var liteswap string
	var skipSettingsCheck bool
	var environmentVersion int
	var baseEnvironment string
	var autoApprove bool
	var usagePolicyID string

	cmd.Flags().StringVar(&clusterID, "cluster", "", "Databricks dedicated cluster ID")
	cmd.Flags().DurationVar(&shutdownDelay, "shutdown-delay", defaultShutdownDelay, "Delay before shutting down the server after the last client disconnects")
	cmd.Flags().IntVar(&maxClients, "max-clients", defaultMaxClients, "Maximum number of SSH clients")
	cmd.Flags().DurationVar(&serverTimeout, "server-timeout", defaultServerTimeout, "Maximum lifetime of the SSH server; it is terminated after this duration even if clients are connected")
	cmd.Flags().BoolVar(&autoStartCluster, "auto-start-cluster", true, "Automatically start the cluster if it is not running")

	cmd.Flags().StringVar(&connectionName, "name", "", "Connection name to reuse across sessions (serverless only)")
	cmd.Flags().StringVar(&accelerator, "accelerator", "", "Serverless GPU accelerator type (GPU_1xA10 or GPU_8xH100)")
	cmd.Flags().StringVar(&ide, "ide", "", "Open remote IDE window (vscode or cursor)")
	cmd.Flags().StringVar(&usagePolicyID, "usage-policy-id", "", "Usage policy ID for the serverless SSH server job (serverless only)")

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

	cmd.Flags().IntVar(&environmentVersion, "environment-version", defaultEnvironmentVersion, "Environment version for AI Runtime")
	cmd.Flags().MarkHidden("environment-version")

	cmd.Flags().StringVar(&baseEnvironment, "base-environment", "", "Custom base environment for serverless compute: an env.yaml path, a workspace-base-environments resource ID, or a display name")

	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompts, installing IDE extensions and applying IDE settings without asking")

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
		if connectionName == "" && clusterID == "" && !proxyMode {
			connectionName = client.GenerateDefaultConnectionName(wsClient.Config.Host, accelerator, baseEnvironment)
		}
		// Serverless GPU compute can take much longer to provision than CPU compute,
		// so allow extra time for the SSH server job to start.
		startupTimeout := taskStartupTimeout
		if accelerator != "" {
			startupTimeout = gpuTaskStartupTimeout
		}
		// Only carry an explicitly-set environment version. Leaving it at 0 otherwise
		// lets the submit path default to minEnvironmentVersion and lets Validate
		// detect a real --environment-version + --base-environment conflict.
		if !cmd.Flags().Changed("environment-version") {
			environmentVersion = 0
		}
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
			KeepaliveInterval:    defaultKeepaliveInterval,
			ReleasesDir:          releasesDir,
			ServerTimeout:        resolveServerTimeout(cmd.Flags(), serverTimeout, shutdownDelay),
			TaskStartupTimeout:   startupTimeout,
			AutoStartCluster:     autoStartCluster,
			ClientPublicKeyName:  clientPublicKeyName,
			ClientPrivateKeyName: clientPrivateKeyName,
			UserKnownHostsFile:   userKnownHostsFile,
			Liteswap:             liteswap,
			SkipSettingsCheck:    skipSettingsCheck,
			EnvironmentVersion:   environmentVersion,
			BaseEnvironment:      baseEnvironment,
			AdditionalArgs:       args,
			AutoApprove:          autoApprove,
			UsagePolicyID:        usagePolicyID,
		}
		if err := opts.Validate(); err != nil {
			return err
		}
		return client.Run(ctx, wsClient, opts)
	}

	return cmd
}
