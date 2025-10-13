package setup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateClusterAccess_SingleUser(t *testing.T) {
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	clustersAPI := m.GetMockClustersAPI()

	clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeSingleUser,
	}, nil)

	err := validateClusterAccess(ctx, m.WorkspaceClient, "cluster-123")
	assert.NoError(t, err)
}

func TestValidateClusterAccess_InvalidAccessMode(t *testing.T) {
	ctx := context.Background()
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
	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	clustersAPI := m.GetMockClustersAPI()

	clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "nonexistent"}).Return(nil, errors.New("cluster not found"))

	err := validateClusterAccess(ctx, m.WorkspaceClient, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cluster information for cluster ID 'nonexistent'")
}

func TestGenerateProxyCommand(t *testing.T) {
	cmd, err := GenerateProxyCommand("cluster-123", true, 45*time.Second, "", "", 0, 0)
	assert.NoError(t, err)
	assert.Contains(t, cmd, "ssh connect --proxy --cluster=cluster-123 --auto-start-cluster=true --shutdown-delay=45s")
	assert.NotContains(t, cmd, "--metadata")
	assert.NotContains(t, cmd, "--profile")
	assert.NotContains(t, cmd, "--handover-timeout")
}

func TestGenerateProxyCommand_WithExtraArgs(t *testing.T) {
	cmd, err := GenerateProxyCommand("cluster-123", true, 45*time.Second, "test-profile", "user", 2222, 2*time.Minute)
	assert.NoError(t, err)
	assert.Contains(t, cmd, "ssh connect --proxy --cluster=cluster-123 --auto-start-cluster=true --shutdown-delay=45s")
	assert.Contains(t, cmd, " --metadata=user,2222")
	assert.Contains(t, cmd, " --handover-timeout=2m0s")
	assert.Contains(t, cmd, " --profile=test-profile")
}

func TestGenerateHostConfig_Valid(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	opts := SetupOptions{
		HostName:      "test-host",
		ClusterID:     "cluster-123",
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
		Profile:       "test-profile",
	}

	result, err := generateHostConfig(opts)
	assert.NoError(t, err)

	assert.Contains(t, result, "Host test-host")
	assert.Contains(t, result, "User root")
	assert.Contains(t, result, "StrictHostKeyChecking accept-new")
	assert.Contains(t, result, "--cluster=cluster-123")
	assert.Contains(t, result, "--shutdown-delay=30s")
	assert.Contains(t, result, "--profile=test-profile")

	// Check that identity file path is included
	expectedKeyPath := filepath.Join(tmpDir, "cluster-123")
	assert.Contains(t, result, fmt.Sprintf(`IdentityFile %q`, expectedKeyPath))
}

func TestGenerateHostConfig_WithoutProfile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	opts := SetupOptions{
		HostName:      "test-host",
		ClusterID:     "cluster-123",
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
		Profile:       "", // No profile
	}

	result, err := generateHostConfig(opts)
	assert.NoError(t, err)

	// Should not contain profile option
	assert.NotContains(t, result, "--profile=")
	// But should contain other elements
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

func TestEnsureSSHConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".ssh", "config")

	err := ensureSSHConfigExists(configPath)
	assert.NoError(t, err)

	// Check that directory was created
	_, err = os.Stat(filepath.Dir(configPath))
	assert.NoError(t, err)

	// Check that file was created
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Check that file is empty
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Empty(t, content)
}

func TestCheckExistingHosts_NoExistingHost(t *testing.T) {
	content := []byte(`Host other-host
    User root
    HostName example.com

Host another-host
    User admin
`)
	exists, err := checkExistingHosts(content, "test-host")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCheckExistingHosts_HostAlreadyExists(t *testing.T) {
	content := []byte(`Host test-host
    User root
    HostName example.com

Host another-host
    User admin
`)
	exists, err := checkExistingHosts(content, "another-host")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCheckExistingHosts_EmptyContent(t *testing.T) {
	content := []byte("")
	exists, err := checkExistingHosts(content, "test-host")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCheckExistingHosts_HostNameWithWhitespaces(t *testing.T) {
	content := []byte(` Host  test-host  `)
	exists, err := checkExistingHosts(content, "test-host")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCheckExistingHosts_PartialNameMatch(t *testing.T) {
	content := []byte(`Host test-host-long`)
	exists, err := checkExistingHosts(content, "test-host")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCreateBackup_CreatesBackupSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	content := []byte("original content")

	backupPath, err := createBackup(content, configPath)
	assert.NoError(t, err)
	assert.Equal(t, configPath+".bak", backupPath)

	// Check that backup file was created with correct content
	backupContent, err := os.ReadFile(backupPath)
	assert.NoError(t, err)
	assert.Equal(t, content, backupContent)
}

func TestCreateBackup_OverwritesExistingBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	backupPath := configPath + ".bak"

	// Create existing backup
	oldContent := []byte("old backup")
	err := os.WriteFile(backupPath, oldContent, 0o644)
	require.NoError(t, err)

	// Create new backup
	newContent := []byte("new content")
	resultPath, err := createBackup(newContent, configPath)
	assert.NoError(t, err)
	assert.Equal(t, backupPath, resultPath)

	// Check that backup was overwritten
	backupContent, err := os.ReadFile(backupPath)
	assert.NoError(t, err)
	assert.Equal(t, newContent, backupContent)
}

func TestUpdateSSHConfigFile_UpdatesSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create initial config file
	initialContent := "# SSH Config\nHost existing\n    User root\n"
	err := os.WriteFile(configPath, []byte(initialContent), 0o600)
	require.NoError(t, err)

	hostConfig := "\nHost new-host\n    User root\n    HostName example.com\n"
	err = updateSSHConfigFile(configPath, hostConfig, "new-host")
	assert.NoError(t, err)

	// Check that content was appended
	finalContent, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	expected := initialContent + hostConfig
	assert.Equal(t, expected, string(finalContent))
}

func TestUpdateSSHConfigFile_AddsNewlineIfMissing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config file without trailing newline
	initialContent := "Host existing\n    User root"
	err := os.WriteFile(configPath, []byte(initialContent), 0o600)
	require.NoError(t, err)

	hostConfig := "\nHost new-host\n    User root\n"
	err = updateSSHConfigFile(configPath, hostConfig, "new-host")
	assert.NoError(t, err)

	// Check that newline was added before the new content
	finalContent, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	expected := initialContent + "\n" + hostConfig
	assert.Equal(t, expected, string(finalContent))
}

func TestUpdateSSHConfigFile_HandlesEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create empty config file
	err := os.WriteFile(configPath, []byte(""), 0o600)
	require.NoError(t, err)

	hostConfig := "Host new-host\n    User root\n"
	err = updateSSHConfigFile(configPath, hostConfig, "new-host")
	assert.NoError(t, err)

	// Check that content was added without extra newlines
	finalContent, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, hostConfig, string(finalContent))
}

func TestUpdateSSHConfigFile_HandlesReadError(t *testing.T) {
	configPath := "/nonexistent/file"
	hostConfig := "Host new-host\n"

	err := updateSSHConfigFile(configPath, hostConfig, "new-host")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read SSH config file")
}

func TestSetup_SuccessfulWithNewConfigFile(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
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

	// Check that config file was created
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	configStr := string(content)
	assert.Contains(t, configStr, "Host test-host")
	assert.Contains(t, configStr, "--cluster=cluster-123")
	assert.Contains(t, configStr, "--profile=test-profile")
}

func TestSetup_SuccessfulWithExistingConfigFile(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
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

	// Check that config file was updated and backup was created
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	configStr := string(content)
	assert.Contains(t, configStr, "# Existing SSH Config") // Original content preserved
	assert.Contains(t, configStr, "Host new-host")         // New content added
	assert.Contains(t, configStr, "--cluster=cluster-456")

	// Check backup was created
	backupPath := configPath + ".bak"
	backupContent, err := os.ReadFile(backupPath)
	assert.NoError(t, err)
	assert.Equal(t, existingContent, string(backupContent))
}

func TestSetup_DoesNotOverrideExistingHost(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ssh_config")

	// Create config file with existing host
	existingContent := "Host duplicate-host\n    User root\n"
	err := os.WriteFile(configPath, []byte(existingContent), 0o600)
	require.NoError(t, err)

	m := mocks.NewMockWorkspaceClient(t)
	clustersAPI := m.GetMockClustersAPI()

	clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
		DataSecurityMode: compute.DataSecurityModeSingleUser,
	}, nil)

	opts := SetupOptions{
		HostName:      "duplicate-host", // Same as existing
		ClusterID:     "cluster-123",
		SSHConfigPath: configPath,
		SSHKeysDir:    tmpDir,
		ShutdownDelay: 30 * time.Second,
	}

	err = Setup(ctx, m.WorkspaceClient, opts)
	assert.NoError(t, err)

	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, "Host duplicate-host\n    User root\n", string(content))
}
