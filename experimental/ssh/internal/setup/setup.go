package setup

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	// Optional path to the local ssh config. Defaults to ~/.ssh/config
	SSHConfigPath string
	// Optional path to the local directory to store SSH keys. Defaults to ~/.databricks/ssh-tunnel-keys
	SSHKeysDir string
	// Optional auth profile name. If present, will be added as --profile flag to the ProxyCommand
	Profile string
	// Proxy command to use for the SSH connection
	ProxyCommand string
}

func validateClusterAccess(ctx context.Context, client *databricks.WorkspaceClient, clusterID string) error {
	clusterInfo, err := client.Clusters.Get(ctx, compute.GetClusterRequest{ClusterId: clusterID})
	if err != nil {
		return fmt.Errorf("failed to get cluster information for cluster ID '%s': %w", clusterID, err)
	}
	if clusterInfo.DataSecurityMode != compute.DataSecurityModeSingleUser {
		return fmt.Errorf("cluster '%s' does not have dedicated access mode. Current access mode: %s. Please ensure the cluster is configured with dedicated access mode (single user)", clusterID, clusterInfo.DataSecurityMode)
	}
	return nil
}

func generateHostConfig(opts SetupOptions) (string, error) {
	identityFilePath, err := keys.GetLocalSSHKeyPath(opts.ClusterID, opts.SSHKeysDir)
	if err != nil {
		return "", fmt.Errorf("failed to get local keys folder: %w", err)
	}

	hostConfig := fmt.Sprintf(`
Host %s
    User root
    ConnectTimeout 360
    StrictHostKeyChecking accept-new
    IdentitiesOnly yes
    IdentityFile %q
    ProxyCommand %s
`, opts.HostName, identityFilePath, opts.ProxyCommand)

	return hostConfig, nil
}

func clusterSelectionPrompt(ctx context.Context, client *databricks.WorkspaceClient) (string, error) {
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

	err := validateClusterAccess(ctx, client, opts.ClusterID)
	if err != nil {
		return err
	}

	configPath, err := sshconfig.GetMainConfigPathOrDefault(opts.SSHConfigPath)
	if err != nil {
		return err
	}

	err = sshconfig.EnsureIncludeDirective(configPath)
	if err != nil {
		return err
	}

	hostConfig, err := generateHostConfig(opts)
	if err != nil {
		return err
	}

	exists, err := sshconfig.HostConfigExists(opts.HostName)
	if err != nil {
		return err
	}

	recreate := false
	if exists {
		recreate, err = sshconfig.PromptRecreateConfig(ctx, opts.HostName)
		if err != nil {
			return err
		}
		if !recreate {
			cmdio.LogString(ctx, fmt.Sprintf("Skipping setup for host '%s'", opts.HostName))
			return nil
		}
	}

	cmdio.LogString(ctx, "Adding new entry to the SSH config:\n"+hostConfig)

	_, err = sshconfig.CreateOrUpdateHostConfig(ctx, opts.HostName, hostConfig, recreate)
	if err != nil {
		return err
	}

	hostConfigPath, err := sshconfig.GetHostConfigPath(opts.HostName)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, fmt.Sprintf("Created SSH config file at %s for '%s' host", hostConfigPath, opts.HostName))
	cmdio.LogString(ctx, fmt.Sprintf("You can now connect to the cluster using 'ssh %s' terminal command, or use remote capabilities of your IDE", opts.HostName))
	return nil
}
