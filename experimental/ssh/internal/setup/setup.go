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
	// Optional path to the local ssh config. Defaults to ~/.ssh/config
	SSHConfigPath string
	// Optional path to the local directory to store SSH keys. Defaults to ~/.databricks/ssh-tunnel-keys
	SSHKeysDir string
	// Optional auth profile name. If present, will be added as --profile flag to the ProxyCommand
	Profile string
	// Skip confirmation prompts (e.g. recreate existing host config without asking)
	AutoApprove bool
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

func generateHostConfig(ctx context.Context, opts SetupOptions, proxyCommand string) (string, error) {
	identityFilePath, err := keys.GetLocalSSHKeyPath(ctx, opts.ClusterID, opts.SSHKeysDir)
	if err != nil {
		return "", fmt.Errorf("failed to get local keys folder: %w", err)
	}

	hostConfig := sshconfig.GenerateHostConfig(opts.HostName, "root", identityFilePath, proxyCommand)
	return hostConfig, nil
}

// clusterSelectionPrompt is a package-level var so tests can replace it with a mock.
var clusterSelectionPrompt = defaultClusterSelectionPrompt

// buildClusterItems converts a list of cluster details into a slice of
// display tuples suitable for an ordered picker. When two or more clusters
// share the same name, the cluster ID is appended in parentheses to each
// duplicate entry so the user can tell them apart.
func buildClusterItems(clusters []compute.ClusterDetails) []cmdio.Tuple {
	seen := make(map[string]bool, len(clusters))
	items := make([]cmdio.Tuple, 0, len(clusters))
	for _, c := range clusters {
		label := c.ClusterName
		if seen[label] {
			// A previous cluster already used this name; go back and append
			// the ID to that earlier entry too, so the list is unambiguous.
			for i, item := range items {
				if item.Name == label {
					items[i].Name = label + " (" + item.Id + ")"
				}
			}
			label = label + " (" + c.ClusterId + ")"
		}
		seen[c.ClusterName] = true
		items = append(items, cmdio.Tuple{Name: label, Id: c.ClusterId})
	}
	return items
}

func defaultClusterSelectionPrompt(ctx context.Context, client *databricks.WorkspaceClient) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading clusters.")
	all, err := client.Clusters.ListAll(ctx, compute.ListClustersRequest{
		FilterBy: &compute.ListClustersFilterBy{
			ClusterSources: []compute.ClusterSource{compute.ClusterSourceApi, compute.ClusterSourceUi},
		},
	})
	sp.Close()
	if err != nil {
		return "", fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify cluster argument. Original error: %w", err)
	}

	// Databricks allows clusters to share names; use buildClusterItems to
	// disambiguate duplicates before passing them to the interactive picker.
	items := buildClusterItems(all)

	id, err := cmdio.SelectOrdered(ctx, items, "The cluster to connect to")
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

	// Build the ProxyCommand after the cluster ID is resolved. When the user
	// omits --cluster, the ID is only known after the interactive picker above,
	// so building it earlier would serialize an empty --cluster= flag.
	clientOpts := sshclient.ClientOptions{
		ClusterID:        opts.ClusterID,
		AutoStartCluster: opts.AutoStartCluster,
		ShutdownDelay:    opts.ShutdownDelay,
		Profile:          opts.Profile,
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
