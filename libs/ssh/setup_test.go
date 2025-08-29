package ssh

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateClusterAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("valid cluster with single user mode", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		clustersAPI := m.GetMockClustersAPI()

		clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
			DataSecurityMode: compute.DataSecurityModeSingleUser,
		}, nil)

		err := validateClusterAccess(ctx, m.WorkspaceClient, "cluster-123")
		assert.NoError(t, err)
	})

	t.Run("cluster with invalid access mode", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		clustersAPI := m.GetMockClustersAPI()

		clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "cluster-123"}).Return(&compute.ClusterDetails{
			DataSecurityMode: compute.DataSecurityModeUserIsolation,
		}, nil)

		err := validateClusterAccess(ctx, m.WorkspaceClient, "cluster-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not have dedicated access mode")
	})

	t.Run("cluster not found", func(t *testing.T) {
		m := mocks.NewMockWorkspaceClient(t)
		clustersAPI := m.GetMockClustersAPI()

		clustersAPI.EXPECT().Get(ctx, compute.GetClusterRequest{ClusterId: "nonexistent"}).Return(nil, errors.New("cluster not found"))

		err := validateClusterAccess(ctx, m.WorkspaceClient, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get cluster information for cluster ID 'nonexistent'")
	})
}

func TestGenerateHostConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	t.Run("valid host config generation", func(t *testing.T) {
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
		assert.Contains(t, result, fmt.Sprintf(`IdentityFile "%s"`, expectedKeyPath))
	})

	t.Run("host config without profile", func(t *testing.T) {
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
	})

	t.Run("path escaping with quotes", func(t *testing.T) {
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
		expectedEscapedPath := strings.ReplaceAll(filepath.Join(specialDir, "cluster-123"), `"`, `\"`)
		assert.Contains(t, result, fmt.Sprintf(`IdentityFile "%s"`, expectedEscapedPath))
	})
}

func TestEnsureSSHConfigExists(t *testing.T) {
	t.Run("creates config file when it doesn't exist", func(t *testing.T) {
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
	})
}

func TestCheckExistingHosts(t *testing.T) {
	t.Run("no existing host with same name", func(t *testing.T) {
		content := []byte(`Host other-host
    User root
    HostName example.com

Host another-host
    User admin
`)
		exists, err := checkExistingHosts(content, "test-host")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("host already exists", func(t *testing.T) {
		content := []byte(`Host test-host
    User root
    HostName example.com

Host another-host
    User admin
`)
		exists, err := checkExistingHosts(content, "another-host")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("empty content", func(t *testing.T) {
		content := []byte("")
		exists, err := checkExistingHosts(content, "test-host")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Host name with whitespaces around it", func(t *testing.T) {
		content := []byte(` Host  test-host  `)
		exists, err := checkExistingHosts(content, "test-host")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("partial name match", func(t *testing.T) {
		content := []byte(`Host test-host-long`)
		exists, err := checkExistingHosts(content, "test-host")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestCreateBackup(t *testing.T) {
	t.Run("creates backup file successfully", func(t *testing.T) {
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
	})

	t.Run("overwrites existing backup", func(t *testing.T) {
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
	})
}

func TestUpdateSSHConfigFile(t *testing.T) {
	t.Run("updates config file successfully", func(t *testing.T) {
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
	})

	t.Run("adds newline if file doesn't end with newline", func(t *testing.T) {
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
	})

	t.Run("handles empty file", func(t *testing.T) {
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
	})

	t.Run("handles read error", func(t *testing.T) {
		configPath := "/nonexistent/file"
		hostConfig := "Host new-host\n"

		err := updateSSHConfigFile(configPath, hostConfig, "new-host")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read SSH config file")
	})
}

func TestSetup(t *testing.T) {
	ctx := context.Background()

	t.Run("successful setup with new config file", func(t *testing.T) {
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
	})

	t.Run("successful setup with existing config file", func(t *testing.T) {
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
	})

	t.Run("doesn't override existing host", func(t *testing.T) {
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
	})
}
