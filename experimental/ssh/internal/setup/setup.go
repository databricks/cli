package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

type SetupOptions struct {
	// A host name to add to the SSH config
	HostName string
	// The cluster ID to connect to
	ClusterID string
	// Delay before shutting down the SSH tunnel, will be added as a --shutdown-delay flag to the ProxyCommand
	ShutdownDelay time.Duration
	// Optional path to the local ssh config. Defaults to ~/.ssh/config
	SSHConfigPath string
	// Optional path to the local directory to store SSH keys. Defaults to ~/.databricks/ssh-tunnel-keys
	SSHKeysDir string
	// Optional auth profile name. If present, will be added as --profile flag to the ProxyCommand
	Profile string
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

func resolveConfigPath(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ssh", "config"), nil
}

func generateHostConfig(opts SetupOptions) (string, error) {
	identityFilePath, err := keys.GetLocalSSHKeyPath(opts.ClusterID, opts.SSHKeysDir)
	if err != nil {
		return "", fmt.Errorf("failed to get local keys folder: %w", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	profileOption := ""
	if opts.Profile != "" {
		profileOption = "--profile=" + opts.Profile
	}

	hostConfig := fmt.Sprintf(`
Host %s
    User root
    ConnectTimeout 360
    StrictHostKeyChecking accept-new
    IdentityFile %q
    ProxyCommand %q ssh connect --proxy --cluster=%s --shutdown-delay=%s %s
`, opts.HostName, identityFilePath, execPath, opts.ClusterID, opts.ShutdownDelay, profileOption)

	return hostConfig, nil
}

func ensureSSHConfigExists(configPath string) error {
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		sshDir := filepath.Dir(configPath)
		err = os.MkdirAll(sshDir, 0o700)
		if err != nil {
			return fmt.Errorf("failed to create SSH directory: %w", err)
		}
		err = os.WriteFile(configPath, []byte(""), 0o600)
		if err != nil {
			return fmt.Errorf("failed to create SSH config file: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check SSH config file: %w", err)
	}
	return nil
}

func checkExistingHosts(content []byte, hostName string) (bool, error) {
	existingContent := string(content)
	pattern := fmt.Sprintf(`(?m)^\s*Host\s+%s\s*$`, regexp.QuoteMeta(hostName))
	matched, err := regexp.MatchString(pattern, existingContent)
	if err != nil {
		return false, fmt.Errorf("failed to check for existing host: %w", err)
	}
	if matched {
		return true, nil
	}
	return false, nil
}

func createBackup(content []byte, configPath string) (string, error) {
	backupPath := configPath + ".bak"
	err := os.WriteFile(backupPath, content, 0o600)
	if err != nil {
		return backupPath, fmt.Errorf("failed to create backup of SSH config file: %w", err)
	}
	return backupPath, nil
}

func updateSSHConfigFile(configPath, hostConfig, hostName string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH config file: %w", err)
	}

	existingContent := string(content)
	if !strings.HasSuffix(existingContent, "\n") && existingContent != "" {
		existingContent += "\n"
	}
	newContent := existingContent + hostConfig

	err = os.WriteFile(configPath, []byte(newContent), 0o600)
	if err != nil {
		return fmt.Errorf("failed to update SSH config file: %w", err)
	}

	return nil
}

func Setup(ctx context.Context, client *databricks.WorkspaceClient, opts SetupOptions) error {
	err := validateClusterAccess(ctx, client, opts.ClusterID)
	if err != nil {
		return err
	}

	configPath, err := resolveConfigPath(opts.SSHConfigPath)
	if err != nil {
		return err
	}

	hostConfig, err := generateHostConfig(opts)
	if err != nil {
		return err
	}

	err = ensureSSHConfigExists(configPath)
	if err != nil {
		return err
	}

	existingContent, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH config file: %w", err)
	}

	if len(existingContent) > 0 {
		exists, err := checkExistingHosts(existingContent, opts.HostName)
		if err != nil {
			return err
		}
		if exists {
			cmdio.LogString(ctx, fmt.Sprintf("Host '%s' already exists in the SSH config, skipping setup", opts.HostName))
			return nil
		}
		backupPath, err := createBackup(existingContent, configPath)
		if err != nil {
			return err
		}
		cmdio.LogString(ctx, "Created backup of existing SSH config at "+backupPath)
	}

	cmdio.LogString(ctx, "Adding new entry to the SSH config:\n"+hostConfig)

	err = updateSSHConfigFile(configPath, hostConfig, opts.HostName)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, fmt.Sprintf("Updated SSH config file at %s with '%s' host", configPath, opts.HostName))
	cmdio.LogString(ctx, fmt.Sprintf("You can now connect to the cluster using 'ssh %s' terminal command, or use remote capabilities of your IDE", opts.HostName))
	return nil
}
