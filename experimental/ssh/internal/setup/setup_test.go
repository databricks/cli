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
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildClusterItems_UniqueNames verifies that clusters with distinct names
// are shown with their original name.
func TestBuildClusterItems_UniqueNames(t *testing.T) {
	clusters := []compute.ClusterDetails{
		{ClusterId: "id-1", ClusterName: "alpha"},
		{ClusterId: "id-2", ClusterName: "beta"},
	}
	items := buildClusterItems(clusters)
	require.Len(t, items, 2)
	assert.Equal(t, "alpha", items[0].Name)
	assert.Equal(t, "id-1", items[0].Id)
	assert.Equal(t, "beta", items[1].Name)
	assert.Equal(t, "id-2", items[1].Id)
}

// TestBuildClusterItems_DuplicateNames verifies that when two clusters share a
// name, both entries are disambiguated by appending their cluster ID.
func TestBuildClusterItems_DuplicateNames(t *testing.T) {
	clusters := []compute.ClusterDetails{
		{ClusterId: "id-1", ClusterName: "shared"},
		{ClusterId: "id-2", ClusterName: "shared"},
	}
	items := buildClusterItems(clusters)
	require.Len(t, items, 2)
	assert.Equal(t, "shared (id-1)", items[0].Name)
	assert.Equal(t, "id-1", items[0].Id)
	assert.Equal(t, "shared (id-2)", items[1].Name)
	assert.Equal(t, "id-2", items[1].Id)
}

// TestBuildClusterItems_ThreeDuplicateNames verifies disambiguation when three
// clusters share the same name.
func TestBuildClusterItems_ThreeDuplicateNames(t *testing.T) {
	clusters := []compute.ClusterDetails{
		{ClusterId: "id-a", ClusterName: "dup"},
		{ClusterId: "id-b", ClusterName: "dup"},
		{ClusterId: "id-c", ClusterName: "dup"},
	}
	items := buildClusterItems(clusters)
	require.Len(t, items, 3)
	assert.Equal(t, "dup (id-a)", items[0].Name)
	assert.Equal(t, "dup (id-b)", items[1].Name)
	assert.Equal(t, "dup (id-c)", items[2].Name)
}

// TestBuildClusterItems_MixedDuplicates verifies that only the duplicate names
// get a suffix while unique names stay unchanged.
func TestBuildClusterItems_MixedDuplicates(t *testing.T) {
	clusters := []compute.ClusterDetails{
		{ClusterId: "id-1", ClusterName: "unique"},
		{ClusterId: "id-2", ClusterName: "dup"},
		{ClusterId: "id-3", ClusterName: "dup"},
	}
	items := buildClusterItems(clusters)
	require.Len(t, items, 3)
	assert.Equal(t, "unique", items[0].Name)
	assert.Equal(t, "dup (id-2)", items[1].Name)
	assert.Equal(t, "dup (id-3)", items[2].Name)
}

func TestValidateClusterAccess_SingleUser(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	clustersAPI := m.GetMockClustersAPI()

	clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeSingleUser,
	}, nil)

	err := validateClusterAccess(ctx, m.WorkspaceClient, "cluster-123")
	assert.NoError(t, err)
}

func TestValidateClusterAccess_InvalidAccessMode(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
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
	ctx := cmdio.MockDiscard(t.Context())
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
	}

	result, err := generateHostConfig(t.Context(), opts, proxyCommand)
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
	}

	result, err := generateHostConfig(t.Context(), opts, proxyCommand)
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

	result, err := generateHostConfig(t.Context(), opts, "")
	assert.NoError(t, err)

	// Check that quotes are properly escaped
	expectedPath := filepath.Join(specialDir, "cluster-123")
	assert.Contains(t, result, fmt.Sprintf(`IdentityFile %q`, expectedPath))
}

func TestSetup_SuccessfulWithNewConfigFile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
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

	err := Setup(ctx, m.WorkspaceClient, opts)
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

func TestSetup_AutoApproveRecreatesExistingHost(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	// Pre-seed an existing host config so PromptRecreateConfig would fire without --auto-approve.
	hostConfigDir := filepath.Join(tmpDir, ".databricks", "ssh-tunnel-configs")
	require.NoError(t, os.MkdirAll(hostConfigDir, 0o700))
	existingHostConfig := filepath.Join(hostConfigDir, "test-host")
	require.NoError(t, os.WriteFile(existingHostConfig, []byte("# stale\nHost test-host\n    User stale\n"), 0o600))

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
		AutoApprove:   true,
	}

	err := Setup(ctx, m.WorkspaceClient, opts)
	assert.NoError(t, err)

	// Host config should be recreated (no longer contains the stale User).
	content, err := os.ReadFile(existingHostConfig)
	require.NoError(t, err)
	s := string(content)
	assert.NotContains(t, s, "User stale")
	assert.Contains(t, s, "--cluster=cluster-123")
}

func TestSetup_PromptsForClusterWhenNotProvided(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configPath := filepath.Join(tmpDir, "ssh_config")

	// Replace the cluster picker with a stub returning a fixed ID. This lets the
	// test exercise the empty-ClusterID path of Setup without prompting.
	origPrompt := clusterSelectionPrompt
	t.Cleanup(func() { clusterSelectionPrompt = origPrompt })
	promptCalled := false
	clusterSelectionPrompt = func(_ context.Context, _ *databricks.WorkspaceClient) (string, error) {
		promptCalled = true
		return "picked-cluster", nil
	}

	m := mocks.NewMockWorkspaceClient(t)
	clustersAPI := m.GetMockClustersAPI()
	clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "picked-cluster"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeSingleUser,
	}, nil)

	opts := SetupOptions{
		HostName:      "test-host",
		SSHConfigPath: configPath,
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
	}

	err := Setup(ctx, m.WorkspaceClient, opts)
	require.NoError(t, err)
	assert.True(t, promptCalled, "cluster picker should run when ClusterID is empty")

	// The picked ID must be serialized into the ProxyCommand's --cluster= flag.
	hostConfigPath := filepath.Join(tmpDir, ".databricks", "ssh-tunnel-configs", "test-host")
	hostContent, err := os.ReadFile(hostConfigPath)
	require.NoError(t, err)
	hostConfigStr := string(hostContent)
	assert.Contains(t, hostConfigStr, "--cluster=picked-cluster")
	assert.NotContains(t, hostConfigStr, "--cluster= ")
}

func TestSetup_SuccessfulWithExistingConfigFile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
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
