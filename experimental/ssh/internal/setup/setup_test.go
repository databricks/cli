package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/client"
	"github.com/databricks/cli/experimental/ssh/internal/sshconfig"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	assert.Contains(t, result, "--cluster=cluster-123")
	assert.Contains(t, result, "--shutdown-delay=30s")
	assert.Contains(t, result, "--profile=test-profile")

	expectedKeyPath := filepath.Join(tmpDir, "cluster-123")
	assert.Contains(t, result, fmt.Sprintf(`IdentityFile %q`, expectedKeyPath))

	// `ssh <name>` reaches ssh through this block and nothing else, so the host key the
	// ProxyCommand pins has to be the one it verifies against (DECO-27882).
	assert.Contains(t, result, "StrictHostKeyChecking yes")
	expectedKnownHostsPath, err := sshconfig.GetKnownHostsPath(t.Context(), "cluster-123", "")
	require.NoError(t, err)
	assert.Contains(t, result, fmt.Sprintf(`UserKnownHostsFile %q`, expectedKnownHostsPath))

	// The host name (test-host) differs from the cluster ID the key is pinned under, so the
	// block has to carry HostKeyAlias cluster-123 or strict checking looks the key up under
	// test-host and fails (DECO-27882).
	assert.Contains(t, result, "\n    HostKeyAlias cluster-123\n")
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
		SingleUserName:   "me@example.com",
	}, nil)

	opts := SetupOptions{
		HostName:      "test-host",
		ClusterID:     "cluster-123",
		SSHConfigPath: configPath,
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
		MaxClients:    10,
		ServerTimeout: 24 * time.Hour,
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
	// The written block pins the key lookup to the cluster ID, which differs from the host
	// name test-host (DECO-27882).
	assert.Contains(t, hostConfigStr, "HostKeyAlias cluster-123")
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
		SingleUserName:   "me@example.com",
	}, nil)

	opts := SetupOptions{
		HostName:      "test-host",
		ClusterID:     "cluster-123",
		SSHConfigPath: configPath,
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
		MaxClients:    10,
		ServerTimeout: 24 * time.Hour,
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

func TestSetup_SerializesServerLifecycleFlags(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockClustersAPI().EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeSingleUser,
		SingleUserName:   "me@example.com",
	}, nil)

	opts := SetupOptions{
		HostName:      "test-host",
		ClusterID:     "cluster-123",
		SSHConfigPath: filepath.Join(tmpDir, "ssh_config"),
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
		MaxClients:    25,
		ServerTimeout: 48 * time.Hour,
	}

	require.NoError(t, Setup(ctx, m.WorkspaceClient, opts))

	// The ProxyCommand is the invocation that submits the server job, so both values have to
	// reach the persisted host config or the user's choice is silently dropped.
	hostContent, err := os.ReadFile(filepath.Join(tmpDir, ".databricks", "ssh-tunnel-configs", "test-host"))
	require.NoError(t, err)
	assert.Contains(t, string(hostContent), "--max-clients=25")
	assert.Contains(t, string(hostContent), "--server-timeout=48h0m0s")
}

func TestSetup_RejectsUnusableServerLifecycleFlags(t *testing.T) {
	tests := []struct {
		name    string
		opts    SetupOptions
		wantErr string
	}{
		{
			name:    "zero max clients",
			opts:    SetupOptions{ServerTimeout: 24 * time.Hour},
			wantErr: "--max-clients must be at least 1, got 0",
		},
		{
			name:    "zero server timeout",
			opts:    SetupOptions{MaxClients: 10},
			wantErr: "--server-timeout must be at least 1s, got 0s",
		},
		{
			name:    "shutdown delay longer than server timeout",
			opts:    SetupOptions{MaxClients: 10, ShutdownDelay: 48 * time.Hour, ServerTimeout: 24 * time.Hour},
			wantErr: "--shutdown-delay (48h0m0s) cannot be longer than --server-timeout (24h0m0s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(t.Context())
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)
			t.Setenv("USERPROFILE", tmpDir)

			// Validation fires before any cluster API calls, so no mock expectations needed.
			m := mocks.NewMockWorkspaceClient(t)

			opts := tt.opts
			opts.HostName = "test-host"
			opts.ClusterID = "cluster-123"
			opts.SSHConfigPath = filepath.Join(tmpDir, "ssh_config")
			opts.SSHKeysDir = tmpDir

			assert.EqualError(t, Setup(ctx, m.WorkspaceClient, opts), tt.wantErr)

			// Nothing is written when the values are rejected.
			_, err := os.Stat(filepath.Join(tmpDir, ".databricks", "ssh-tunnel-configs", "test-host"))
			assert.ErrorIs(t, err, os.ErrNotExist)
		})
	}
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
		SingleUserName:   "me@example.com",
	}, nil)

	opts := SetupOptions{
		HostName:      "test-host",
		SSHConfigPath: configPath,
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
		MaxClients:    10,
		ServerTimeout: 24 * time.Hour,
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
		SingleUserName:   "me@example.com",
	}, nil)

	opts := SetupOptions{
		HostName:      "new-host",
		ClusterID:     "cluster-456",
		SSHConfigPath: configPath,
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 60 * time.Second,
		MaxClients:    10,
		ServerTimeout: 24 * time.Hour,
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
