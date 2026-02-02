package sshconfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	assert.NoError(t, err)
	assert.Contains(t, dir, filepath.Join(".databricks", "ssh-tunnel-configs"))
}

func TestGetMainConfigPath(t *testing.T) {
	path, err := GetMainConfigPath()
	assert.NoError(t, err)
	assert.Contains(t, path, filepath.Join(".ssh", "config"))
}

func TestGetMainConfigPathOrDefault(t *testing.T) {
	path, err := GetMainConfigPathOrDefault("/custom/path")
	assert.NoError(t, err)
	assert.Equal(t, "/custom/path", path)

	path, err = GetMainConfigPathOrDefault("")
	assert.NoError(t, err)
	assert.Contains(t, path, filepath.Join(".ssh", "config"))
}

func TestEnsureMainConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".ssh", "config")

	err := EnsureMainConfigExists(configPath)
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Dir(configPath))
	assert.NoError(t, err)

	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Empty(t, content)
}

func TestEnsureIncludeDirective_NewConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".ssh", "config")

	// Set home directory for test
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	err := EnsureIncludeDirective(configPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	configStr := string(content)
	assert.Contains(t, configStr, "Include")
	// SSH config uses forward slashes on all platforms
	assert.Contains(t, configStr, ".databricks/ssh-tunnel-configs/*")
}

func TestEnsureIncludeDirective_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configPath := filepath.Join(tmpDir, ".ssh", "config")

	configDir, err := GetConfigDir()
	require.NoError(t, err)

	// Use forward slashes as that's what SSH config uses
	configDirUnix := filepath.ToSlash(configDir)
	existingContent := "Include " + configDirUnix + "/*\n\nHost example\n    User test\n"
	err = os.MkdirAll(filepath.Dir(configPath), 0o700)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte(existingContent), 0o600)
	require.NoError(t, err)

	err = EnsureIncludeDirective(configPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, existingContent, string(content))
}

func TestEnsureIncludeDirective_PrependsToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".ssh", "config")

	// Set home directory for test
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	existingContent := "Host example\n    User test\n"
	err := os.MkdirAll(filepath.Dir(configPath), 0o700)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte(existingContent), 0o600)
	require.NoError(t, err)

	err = EnsureIncludeDirective(configPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	configStr := string(content)
	assert.Contains(t, configStr, "Include")
	// SSH config uses forward slashes on all platforms
	assert.Contains(t, configStr, ".databricks/ssh-tunnel-configs/*")
	assert.Contains(t, configStr, "Host example")

	includeIndex := len("Include")
	hostIndex := len(configStr) - len(existingContent)
	assert.Less(t, includeIndex, hostIndex, "Include directive should come before existing content")
}

func TestGetHostConfigPath(t *testing.T) {
	path, err := GetHostConfigPath("test-host")
	assert.NoError(t, err)
	assert.Contains(t, path, filepath.Join(".databricks", "ssh-tunnel-configs", "test-host"))
}

func TestHostConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	exists, err := HostConfigExists("nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)

	configDir := filepath.Join(tmpDir, ConfigDirName)
	err = os.MkdirAll(configDir, 0o700)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(configDir, "existing-host"), []byte("config"), 0o600)
	require.NoError(t, err)

	exists, err = HostConfigExists("existing-host")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCreateOrUpdateHostConfig_NewConfig(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	hostConfig := "Host test\n    User root\n"
	created, err := CreateOrUpdateHostConfig(ctx, "test-host", hostConfig, false)
	assert.NoError(t, err)
	assert.True(t, created)

	configPath, err := GetHostConfigPath("test-host")
	require.NoError(t, err)
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, hostConfig, string(content))
}

func TestCreateOrUpdateHostConfig_ExistingConfigNoRecreate(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configDir := filepath.Join(tmpDir, ConfigDirName)
	err := os.MkdirAll(configDir, 0o700)
	require.NoError(t, err)
	existingConfig := "Host test\n    User admin\n"
	err = os.WriteFile(filepath.Join(configDir, "test-host"), []byte(existingConfig), 0o600)
	require.NoError(t, err)

	newConfig := "Host test\n    User root\n"
	created, err := CreateOrUpdateHostConfig(ctx, "test-host", newConfig, false)
	assert.NoError(t, err)
	assert.False(t, created)

	configPath, err := GetHostConfigPath("test-host")
	require.NoError(t, err)
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, existingConfig, string(content))
}

func TestCreateOrUpdateHostConfig_ExistingConfigWithRecreate(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configDir := filepath.Join(tmpDir, ConfigDirName)
	err := os.MkdirAll(configDir, 0o700)
	require.NoError(t, err)
	existingConfig := "Host test\n    User admin\n"
	err = os.WriteFile(filepath.Join(configDir, "test-host"), []byte(existingConfig), 0o600)
	require.NoError(t, err)

	newConfig := "Host test\n    User root\n"
	created, err := CreateOrUpdateHostConfig(ctx, "test-host", newConfig, true)
	assert.NoError(t, err)
	assert.True(t, created)

	configPath, err := GetHostConfigPath("test-host")
	require.NoError(t, err)
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, newConfig, string(content))
}
