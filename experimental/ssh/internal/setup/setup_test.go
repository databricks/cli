package setup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/client"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateClusterAccess_SingleUser(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	m := mocks.NewMockWorkspaceClient(t)
	clustersAPI := m.GetMockClustersAPI()

	clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeSingleUser,
	}, nil)

	err := validateClusterAccess(ctx, m.WorkspaceClient, "cluster-123")
	assert.NoError(t, err)
}

func TestValidateClusterAccess_InvalidAccessMode(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	m := mocks.NewMockWorkspaceClient(t)
	clustersAPI := m.GetMockClustersAPI()

	clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeUserIsolation,
	}, nil)

	err := validateClusterAccess(ctx, m.WorkspaceClient, "cluster-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not have dedicated access mode")
}

func TestValidateClusterAccess_ClusterNotFound(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	m := mocks.NewMockWorkspaceClient(t)
	clustersAPI := m.GetMockClustersAPI()

	clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "nonexistent"}).Return(nil, errors.New("cluster not found"))

	err := validateClusterAccess(ctx, m.WorkspaceClient, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cluster information for cluster ID 'nonexistent'")
}

func TestGenerateProxyCommand(t *testing.T) {
	opts := client.ClientOptions{
		ClusterID:        "cluster-123",
		AutoStartCluster: true,
		ShutdownDelay:    45 * time.Second,
	}
	cmd, err := opts.ToProxyCommand()
	assert.NoError(t, err)
	assert.Contains(t, cmd, "ssh connect --proxy --cluster=cluster-123 --auto-start-cluster=true --shutdown-delay=45s")
	assert.NotContains(t, cmd, "--metadata")
	assert.NotContains(t, cmd, "--profile")
	assert.NotContains(t, cmd, "--handover-timeout")
}

func TestGenerateProxyCommand_WithExtraArgs(t *testing.T) {
	opts := client.ClientOptions{
		ClusterID:        "cluster-123",
		AutoStartCluster: true,
		ShutdownDelay:    45 * time.Second,
		Profile:          "test-profile",
		ServerMetadata:   "user,2222",
		HandoverTimeout:  2 * time.Minute,
	}
	cmd, err := opts.ToProxyCommand()
	assert.NoError(t, err)
	assert.Contains(t, cmd, "ssh connect --proxy --cluster=cluster-123 --auto-start-cluster=true --shutdown-delay=45s")
	assert.Contains(t, cmd, " --metadata=user,2222")
	assert.Contains(t, cmd, " --handover-timeout=2m0s")
	assert.Contains(t, cmd, " --profile=test-profile")
}

func TestGenerateProxyCommand_ServerlessMode(t *testing.T) {
	opts := client.ClientOptions{
		ConnectionName: "my-connection",
		ShutdownDelay:  45 * time.Second,
		ServerMetadata: "user,2222,serverless-cluster-id",
	}
	cmd, err := opts.ToProxyCommand()
	assert.NoError(t, err)
	assert.Contains(t, cmd, "ssh connect --proxy --name=my-connection --shutdown-delay=45s")
	assert.Contains(t, cmd, " --metadata=user,2222,serverless-cluster-id")
	assert.NotContains(t, cmd, "--cluster=")
	assert.NotContains(t, cmd, "--auto-start-cluster")
}

func TestGenerateProxyCommand_ServerlessModeWithAccelerator(t *testing.T) {
	opts := client.ClientOptions{
		ConnectionName: "my-connection",
		ShutdownDelay:  45 * time.Second,
		Accelerator:    "GPU_1xA10",
		ServerMetadata: "user,2222,serverless-cluster-id",
	}
	cmd, err := opts.ToProxyCommand()
	assert.NoError(t, err)
	assert.Contains(t, cmd, "ssh connect --proxy --name=my-connection --shutdown-delay=45s")
	assert.Contains(t, cmd, " --accelerator=GPU_1xA10")
	assert.Contains(t, cmd, " --metadata=user,2222,serverless-cluster-id")
	assert.NotContains(t, cmd, "--cluster=")
	assert.NotContains(t, cmd, "--auto-start-cluster")
}

func TestGenerateHostConfig_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	clientOpts := client.ClientOptions{
		ClusterID:        "cluster-123",
		AutoStartCluster: true,
		ShutdownDelay:    30 * time.Second,
		Profile:          "test-profile",
	}
	proxyCommand, err := clientOpts.ToProxyCommand()
	require.NoError(t, err)

	opts := SetupOptions{
		HostName:      "test-host",
		ClusterID:     "cluster-123",
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
		Profile:       "test-profile",
		ProxyCommand:  proxyCommand,
	}

	result, err := generateHostConfig(opts)
	assert.NoError(t, err)

	assert.Contains(t, result, "Host test-host")
	assert.Contains(t, result, "User root")
	assert.Contains(t, result, "StrictHostKeyChecking accept-new")
	assert.Contains(t, result, "--cluster=cluster-123")
	assert.Contains(t, result, "--shutdown-delay=30s")
	assert.Contains(t, result, "--profile=test-profile")

	expectedKeyPath := filepath.Join(tmpDir, "cluster-123")
	assert.Contains(t, result, fmt.Sprintf(`IdentityFile %q`, expectedKeyPath))
}

func TestGenerateHostConfig_WithoutProfile(t *testing.T) {
	tmpDir := t.TempDir()

	clientOpts := client.ClientOptions{
		ClusterID:        "cluster-123",
		AutoStartCluster: true,
		ShutdownDelay:    30 * time.Second,
		Profile:          "",
	}
	proxyCommand, err := clientOpts.ToProxyCommand()
	require.NoError(t, err)

	opts := SetupOptions{
		HostName:      "test-host",
		ClusterID:     "cluster-123",
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
		Profile:       "",
		ProxyCommand:  proxyCommand,
	}

	result, err := generateHostConfig(opts)
	assert.NoError(t, err)

	assert.NotContains(t, result, "--profile=")
	assert.Contains(t, result, "Host test-host")
	assert.Contains(t, result, "--cluster=cluster-123")
}

func TestGenerateHostConfig_PathEscaping(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	// Create a directory with quotes in the name for testing escaping
	specialDir := filepath.Join(tmpDir, `path"with"quotes`)

	opts := SetupOptions{
		HostName:      "test-host",
		ClusterID:     "cluster-123",
		SSHKeysDir:    specialDir,
		ShutdownDelay: 30 * time.Second,
	}

	result, err := generateHostConfig(opts)
	assert.NoError(t, err)

	// Check that quotes are properly escaped
	expectedPath := filepath.Join(specialDir, "cluster-123")
	assert.Contains(t, result, fmt.Sprintf(`IdentityFile %q`, expectedPath))
}

func TestSetup_SuccessfulWithNewConfigFile(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configPath := filepath.Join(tmpDir, "ssh_config")

	m := mocks.NewMockWorkspaceClient(t)
	clustersAPI := m.GetMockClustersAPI()

	clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeSingleUser,
	}, nil)

	opts := SetupOptions{
		HostName:      "test-host",
		ClusterID:     "cluster-123",
		SSHConfigPath: configPath,
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
		Profile:       "test-profile",
	}

	clientOpts := client.ClientOptions{
		ClusterID:        opts.ClusterID,
		AutoStartCluster: opts.AutoStartCluster,
		ShutdownDelay:    opts.ShutdownDelay,
		Profile:          opts.Profile,
	}
	proxyCommand, err := clientOpts.ToProxyCommand()
	require.NoError(t, err)
	opts.ProxyCommand = proxyCommand

	err = Setup(ctx, m.WorkspaceClient, opts)
	assert.NoError(t, err)

	// Check that main config has Include directive
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	configStr := string(content)
	assert.Contains(t, configStr, "Include")
	// SSH config uses forward slashes on all platforms
	assert.Contains(t, configStr, ".databricks/ssh-tunnel-configs/*")

	// Check that host config file was created
	hostConfigPath := filepath.Join(tmpDir, ".databricks", "ssh-tunnel-configs", "test-host")
	hostContent, err := os.ReadFile(hostConfigPath)
	assert.NoError(t, err)
	hostConfigStr := string(hostContent)
	assert.Contains(t, hostConfigStr, "Host test-host")
	assert.Contains(t, hostConfigStr, "--cluster=cluster-123")
	assert.Contains(t, hostConfigStr, "--profile=test-profile")
}

func TestSetup_SuccessfulWithExistingConfigFile(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configPath := filepath.Join(tmpDir, "ssh_config")

	// Create existing config file
	existingContent := "# Existing SSH Config\nHost existing-host\n    User root\n"
	err := os.WriteFile(configPath, []byte(existingContent), 0o600)
	require.NoError(t, err)

	m := mocks.NewMockWorkspaceClient(t)
	clustersAPI := m.GetMockClustersAPI()

	clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-456"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeSingleUser,
	}, nil)

	opts := SetupOptions{
		HostName:      "new-host",
		ClusterID:     "cluster-456",
		SSHConfigPath: configPath,
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 60 * time.Second,
	}

	clientOpts := client.ClientOptions{
		ClusterID:        opts.ClusterID,
		AutoStartCluster: opts.AutoStartCluster,
		ShutdownDelay:    opts.ShutdownDelay,
		Profile:          opts.Profile,
	}
	proxyCommand, err := clientOpts.ToProxyCommand()
	require.NoError(t, err)
	opts.ProxyCommand = proxyCommand

	err = Setup(ctx, m.WorkspaceClient, opts)
	assert.NoError(t, err)

	// Check that main config has Include directive and preserves existing content
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	configStr := string(content)
	assert.Contains(t, configStr, "Include")
	// SSH config uses forward slashes on all platforms
	assert.Contains(t, configStr, ".databricks/ssh-tunnel-configs/*")
	assert.Contains(t, configStr, "# Existing SSH Config")
	assert.Contains(t, configStr, "Host existing-host")

	// Check that host config file was created
	hostConfigPath := filepath.Join(tmpDir, ".databricks", "ssh-tunnel-configs", "new-host")
	hostContent, err := os.ReadFile(hostConfigPath)
	assert.NoError(t, err)
	hostConfigStr := string(hostContent)
	assert.Contains(t, hostConfigStr, "Host new-host")
	assert.Contains(t, hostConfigStr, "--cluster=cluster-456")
}
