package setup

import (
	"context"
	"errors"
	"fmt"
	"time"

	sshclient "github.com/databricks/cli/experimental/ssh/internal/client"
	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/cli/experimental/ssh/internal/sshconfig"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

type SetupOptions struct {
	// A host name to add to the SSH config
	HostName string
	// The cluster ID to connect to
	ClusterID string
	// Whether to automatically start the cluster during ssh connection if it is not running
	AutoStartCluster bool
	// Delay before shutting down the SSH tunnel, will be added as a --shutdown-delay flag to the ProxyCommand
	ShutdownDelay time.Duration
	// Maximum number of concurrent SSH clients the server accepts, will be added as a --max-clients
	// flag to the ProxyCommand. Fixed when the server job is submitted, so the ProxyCommand is the
	// only place it can be set for a host configured through setup.
	MaxClients int
	// Maximum lifetime of the SSH server, will be added as a --server-timeout flag to the ProxyCommand.
	// Also fixed at submission time.
	ServerTimeout time.Duration
	// Optional path to the local ssh config. Defaults to ~/.ssh/config
	SSHConfigPath string
	// Optional path to the local directory to store SSH keys. Defaults to ~/.databricks/ssh-tunnel-keys
	SSHKeysDir string
	// Optional auth profile name. If present, will be added as --profile flag to the ProxyCommand
	Profile string
	// Skip confirmation prompts (e.g. recreate existing host config without asking)
	AutoApprove bool
}

func generateHostConfig(ctx context.Context, opts SetupOptions, proxyCommand string) (string, error) {
	identityFilePath, err := keys.GetLocalSSHKeyPath(ctx, opts.ClusterID, opts.SSHKeysDir)
	if err != nil {
		return "", fmt.Errorf("failed to get local keys folder: %w", err)
	}

	// The ProxyCommand writes this file before the connection reaches host key
	// verification, so it does not have to exist yet. It carries no directory override, so
	// resolve the default one here as well.
	knownHostsPath, err := sshconfig.GetKnownHostsPath(ctx, opts.ClusterID, "")
	if err != nil {
		return "", err
	}

	// The ProxyCommand pins the server's key under the cluster ID (the session ID for a
	// dedicated cluster), but the block is written as `Host <opts.HostName>`. When the
	// user-facing name differs from the cluster ID, ssh would look the key up under the name
	// and fail strict checking, so pass the cluster ID as HostKeyAlias to match the pinned
	// entry (DECO-27882).
	hostConfig := sshconfig.GenerateHostConfig(opts.HostName, "root", identityFilePath, knownHostsPath, opts.ClusterID, proxyCommand)
	return hostConfig, nil
}

// clusterSelectionPrompt is a package-level var so tests can replace it with a mock.
var clusterSelectionPrompt = defaultClusterSelectionPrompt

func defaultClusterSelectionPrompt(ctx context.Context, client *databricks.WorkspaceClient) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading clusters.")
	clusters, err := client.Clusters.ClusterDetailsClusterNameToClusterIdMap(ctx, compute.ListClustersRequest{
		FilterBy: &compute.ListClustersFilterBy{
			ClusterSources: []compute.ClusterSource{compute.ClusterSourceApi, compute.ClusterSourceUi},
		},
	})
	sp.Close()
	if err != nil {
		return "", fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify cluster argument. Original error: %w", err)
	}
	id, err := cmdio.Select(ctx, clusters, "The cluster to connect to")
	if err != nil {
		return "", err
	}
	return id, nil
}

func Setup(ctx context.Context, client *databricks.WorkspaceClient, opts SetupOptions) error {
	// Reject invalid server-lifecycle flag values before the cluster picker and
	// cluster-access check: these values don't depend on cluster details.
	if opts.MaxClients < 1 {
		return fmt.Errorf("--max-clients must be at least 1, got %d", opts.MaxClients)
	}
	if opts.ServerTimeout < time.Second {
		return fmt.Errorf("--server-timeout must be at least 1s, got %s", opts.ServerTimeout)
	}
	if opts.ShutdownDelay > opts.ServerTimeout {
		return fmt.Errorf("--shutdown-delay (%s) cannot be longer than --server-timeout (%s)", opts.ShutdownDelay, opts.ServerTimeout)
	}

	if opts.ClusterID == "" {
		id, err := clusterSelectionPrompt(ctx, client)
		if err != nil {
			return err
		}
		opts.ClusterID = id
	}

	if opts.ClusterID == "" {
		return errors.New("cluster ID is required")
	}

	err := sshclient.ValidateClusterAccess(ctx, client, opts.ClusterID)
	if err != nil {
		return err
	}

	// Build the ProxyCommand after the cluster ID is resolved. When the user
	// omits --cluster, the ID is only known after the interactive picker above,
	// so building it earlier would serialize an empty --cluster= flag.
	clientOpts := sshclient.ClientOptions{
		ClusterID:        opts.ClusterID,
		AutoStartCluster: opts.AutoStartCluster,
		ShutdownDelay:    opts.ShutdownDelay,
		MaxClients:       opts.MaxClients,
		ServerTimeout:    opts.ServerTimeout,
		Profile:          opts.Profile,
	}
	// The ProxyCommand is persisted in the SSH config, so reject values that would produce a
	// tunnel that can never work (e.g. --max-clients=0) here rather than at first `ssh <name>`.
	if err := clientOpts.Validate(); err != nil {
		return err
	}
	proxyCommand, err := clientOpts.ToProxyCommand()
	if err != nil {
		return fmt.Errorf("failed to generate ProxyCommand: %w", err)
	}

	configPath, err := sshconfig.GetMainConfigPathOrDefault(ctx, opts.SSHConfigPath)
	if err != nil {
		return err
	}

	err = sshconfig.EnsureIncludeDirective(ctx, configPath)
	if err != nil {
		return err
	}

	hostConfig, err := generateHostConfig(ctx, opts, proxyCommand)
	if err != nil {
		return err
	}

	exists, err := sshconfig.HostConfigExists(ctx, opts.HostName)
	if err != nil {
		return err
	}

	recreate := false
	if exists {
		if opts.AutoApprove {
			recreate = true
			cmdio.LogString(ctx, fmt.Sprintf("Host '%s' already exists, recreating (--auto-approve)", opts.HostName))
		} else {
			recreate, err = sshconfig.PromptRecreateConfig(ctx, opts.HostName)
			if err != nil {
				return err
			}
			if !recreate {
				cmdio.LogString(ctx, fmt.Sprintf("Skipping setup for host '%s'", opts.HostName))
				return nil
			}
		}
	}

	cmdio.LogString(ctx, "Adding new entry to the SSH config:\n"+hostConfig)

	_, err = sshconfig.CreateOrUpdateHostConfig(ctx, opts.HostName, hostConfig, recreate)
	if err != nil {
		return err
	}

	hostConfigPath, err := sshconfig.GetHostConfigPath(ctx, opts.HostName)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, fmt.Sprintf("Created SSH config file at %s for '%s' host", hostConfigPath, opts.HostName))
	cmdio.LogString(ctx, fmt.Sprintf("You can now connect to the cluster using 'ssh %s' terminal command, or use remote capabilities of your IDE", opts.HostName))
	return nil
}
