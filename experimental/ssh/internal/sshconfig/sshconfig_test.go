package sshconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir(t.Context())
	assert.NoError(t, err)
	assert.Contains(t, dir, filepath.Join(".databricks", "ssh-tunnel-configs"))
}

func TestGetMainConfigPath(t *testing.T) {
	path, err := GetMainConfigPath(t.Context())
	assert.NoError(t, err)
	assert.Contains(t, path, filepath.Join(".ssh", "config"))
}

func TestGetMainConfigPathOrDefault(t *testing.T) {
	path, err := GetMainConfigPathOrDefault(t.Context(), "/custom/path")
	assert.NoError(t, err)
	assert.Equal(t, "/custom/path", path)

	path, err = GetMainConfigPathOrDefault(t.Context(), "")
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

	err := EnsureIncludeDirective(t.Context(), configPath)
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

	configDir, err := GetConfigDir(t.Context())
	require.NoError(t, err)

	// Use forward slashes and quotes as that's what SSH config uses
	configDirUnix := filepath.ToSlash(configDir)
	existingContent := `Include "` + configDirUnix + `/*"` + "\n\nHost example\n    User test\n"
	err = os.MkdirAll(filepath.Dir(configPath), 0o700)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte(existingContent), 0o600)
	require.NoError(t, err)

	err = EnsureIncludeDirective(t.Context(), configPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, existingContent, string(content))
}

func TestEnsureIncludeDirective_MigratesOldUnquotedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configPath := filepath.Join(tmpDir, ".ssh", "config")

	configDir, err := GetConfigDir(t.Context())
	require.NoError(t, err)

	configDirUnix := filepath.ToSlash(configDir)
	oldContent := "Include " + configDirUnix + "/*\n\nHost example\n    User test\n"
	err = os.MkdirAll(filepath.Dir(configPath), 0o700)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte(oldContent), 0o600)
	require.NoError(t, err)

	err = EnsureIncludeDirective(t.Context(), configPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	configStr := string(content)

	assert.Contains(t, configStr, `Include "`+configDirUnix+`/*"`)
	assert.NotContains(t, configStr, "Include "+configDirUnix+"/*\n")
	assert.Contains(t, configStr, "Host example")
}

func TestEnsureIncludeDirective_NotFooledBySubstring(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configPath := filepath.Join(tmpDir, ".ssh", "config")

	configDir, err := GetConfigDir(t.Context())
	require.NoError(t, err)

	configDirUnix := filepath.ToSlash(configDir)
	// The include path appears only inside a comment, not as a standalone directive.
	existingContent := `# Include "` + configDirUnix + `/*"` + "\nHost example\n    User test\n"
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o700))
	require.NoError(t, os.WriteFile(configPath, []byte(existingContent), 0o600))

	err = EnsureIncludeDirective(t.Context(), configPath)
	require.NoError(t, err)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), `Include "`+configDirUnix+`/*"`)
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

	err = EnsureIncludeDirective(t.Context(), configPath)
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

func TestContainsLine(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		line  string
		found bool
	}{
		{"exact match", `Include "/path/*"` + "\nHost example\n", `Include "/path/*"`, true},
		{"not present", "Host example\n", `Include "/path/*"`, false},
		{"substring only", `# Include "/path/*"`, `Include "/path/*"`, false},
		{"commented line", `# Include "/path/*"` + "\n" + `Include "/path/*"` + "\n", `Include "/path/*"`, true},
		{"windows line ending", `Include "/path/*"` + "\r\nHost example\r\n", `Include "/path/*"`, true},
		{"empty data", "", `Include "/path/*"`, false},
		{"indented with spaces", "  " + `Include "/path/*"` + "\nHost example\n", `Include "/path/*"`, true},
		{"indented with tab", "\t" + `Include "/path/*"` + "\nHost example\n", `Include "/path/*"`, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.found, containsLine([]byte(tc.data), tc.line))
		})
	}
}

func TestReplaceLine(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		old      string
		new      string
		expected string
	}{
		{
			"exact match",
			`Include "/p/*"` + "\nHost x\n",
			`Include "/p/*"`, `Include "/p/*" NEW`,
			`Include "/p/*" NEW` + "\nHost x\n",
		},
		{
			"indented match",
			"  " + `Include "/p/*"` + "\nHost x\n",
			`Include "/p/*"`, `Include "/p/*" NEW`,
			`Include "/p/*" NEW` + "\nHost x\n",
		},
		{
			"no match",
			"Host x\n",
			`Include "/p/*"`, `Include "/p/*" NEW`,
			"Host x\n",
		},
		{
			"substring in comment — must not be replaced",
			`# Include "/p/*"` + "\nHost x\n",
			`Include "/p/*"`, `Include "/p/*" NEW`,
			`# Include "/p/*"` + "\nHost x\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := replaceLine([]byte(tc.data), tc.old, tc.new)
			assert.Equal(t, tc.expected, string(got))
		})
	}
}

func TestEnsureIncludeDirective_MigratesIndentedOldFormat(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configPath := filepath.Join(tmpDir, ".ssh", "config")

	configDir, err := GetConfigDir(t.Context())
	require.NoError(t, err)

	configDirUnix := filepath.ToSlash(configDir)
	// Old format with leading whitespace — should still be detected and migrated.
	oldContent := "  Include " + configDirUnix + "/*\n\nHost example\n    User test\n"
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o700))
	require.NoError(t, os.WriteFile(configPath, []byte(oldContent), 0o600))

	err = EnsureIncludeDirective(t.Context(), configPath)
	require.NoError(t, err)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	configStr := string(content)

	assert.Contains(t, configStr, `Include "`+configDirUnix+`/*"`)
	assert.NotContains(t, configStr, "  Include "+configDirUnix+"/*")
	assert.Contains(t, configStr, "Host example")
}

func TestEnsureIncludeDirective_NotFooledByOldFormatSubstring(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configPath := filepath.Join(tmpDir, ".ssh", "config")

	configDir, err := GetConfigDir(t.Context())
	require.NoError(t, err)

	configDirUnix := filepath.ToSlash(configDir)
	// Old unquoted form appears only inside a comment — must not be migrated.
	existingContent := "# Include " + configDirUnix + "/*\nHost example\n    User test\n"
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o700))
	require.NoError(t, os.WriteFile(configPath, []byte(existingContent), 0o600))

	err = EnsureIncludeDirective(t.Context(), configPath)
	require.NoError(t, err)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	configStr := string(content)

	// New quoted directive should have been prepended (not a migration).
	assert.Contains(t, configStr, `Include "`+configDirUnix+`/*"`)
	// Comment line must be preserved unchanged.
	assert.Contains(t, configStr, "# Include "+configDirUnix+"/*")
}

func TestGetHostConfigPath(t *testing.T) {
	path, err := GetHostConfigPath(t.Context(), "test-host")
	assert.NoError(t, err)
	assert.Contains(t, path, filepath.Join(".databricks", "ssh-tunnel-configs", "test-host"))
}

func TestHostConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	exists, err := HostConfigExists(t.Context(), "nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)

	configDir := filepath.Join(tmpDir, configDirName)
	err = os.MkdirAll(configDir, 0o700)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(configDir, "existing-host"), []byte("config"), 0o600)
	require.NoError(t, err)

	exists, err = HostConfigExists(t.Context(), "existing-host")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCreateOrUpdateHostConfig_NewConfig(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	hostConfig := "Host test\n    User root\n"
	created, err := CreateOrUpdateHostConfig(ctx, "test-host", hostConfig, false)
	assert.NoError(t, err)
	assert.True(t, created)

	configPath, err := GetHostConfigPath(ctx, "test-host")
	require.NoError(t, err)
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, hostConfig, string(content))
}

func TestCreateOrUpdateHostConfig_ExistingConfigNoRecreate(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configDir := filepath.Join(tmpDir, configDirName)
	err := os.MkdirAll(configDir, 0o700)
	require.NoError(t, err)
	existingConfig := "Host test\n    User admin\n"
	err = os.WriteFile(filepath.Join(configDir, "test-host"), []byte(existingConfig), 0o600)
	require.NoError(t, err)

	newConfig := "Host test\n    User root\n"
	created, err := CreateOrUpdateHostConfig(ctx, "test-host", newConfig, false)
	assert.NoError(t, err)
	assert.False(t, created)

	configPath, err := GetHostConfigPath(ctx, "test-host")
	require.NoError(t, err)
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, existingConfig, string(content))
}

func TestCreateOrUpdateHostConfig_ExistingConfigWithRecreate(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	configDir := filepath.Join(tmpDir, configDirName)
	err := os.MkdirAll(configDir, 0o700)
	require.NoError(t, err)
	existingConfig := "Host test\n    User admin\n"
	err = os.WriteFile(filepath.Join(configDir, "test-host"), []byte(existingConfig), 0o600)
	require.NoError(t, err)

	newConfig := "Host test\n    User root\n"
	created, err := CreateOrUpdateHostConfig(ctx, "test-host", newConfig, true)
	assert.NoError(t, err)
	assert.True(t, created)

	configPath, err := GetHostConfigPath(ctx, "test-host")
	require.NoError(t, err)
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, newConfig, string(content))
}

func TestGetSocketsDir(t *testing.T) {
	dir, err := GetSocketsDir(t.Context())
	assert.NoError(t, err)
	assert.Contains(t, dir, filepath.Join(".databricks", "ssh-sockets"))
}

func TestEnsureSocketsDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	err := EnsureSocketsDir(t.Context())
	require.NoError(t, err)

	socketsDir := filepath.Join(tmpDir, socketsDirName)
	info, err := os.Stat(socketsDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestGenerateHostConfig_Basic(t *testing.T) {
	config := GenerateHostConfig(HostConfigOptions{
		HostName:     "my-cluster",
		UserName:     "root",
		IdentityFile: "/home/user/.databricks/ssh-tunnel-keys/abc-123",
		ProxyCommand: `"/usr/local/bin/databricks" ssh connect --proxy --cluster=abc-123`,
	})

	assert.Contains(t, config, "Host my-cluster")
	assert.Contains(t, config, "User root")
	assert.Contains(t, config, "ConnectTimeout 360")
	assert.Contains(t, config, "StrictHostKeyChecking accept-new")
	assert.Contains(t, config, "IdentitiesOnly yes")
	assert.Contains(t, config, `IdentityFile "/home/user/.databricks/ssh-tunnel-keys/abc-123"`)
	assert.Contains(t, config, `ProxyCommand "/usr/local/bin/databricks" ssh connect --proxy --cluster=abc-123`)
	assert.NotContains(t, config, "ControlMaster")
	assert.NotContains(t, config, "ControlPath")
	assert.NotContains(t, config, "ControlPersist")
}

func TestGenerateHostConfig_WithControlMaster(t *testing.T) {
	config := GenerateHostConfig(HostConfigOptions{
		HostName:     "my-cluster",
		UserName:     "root",
		IdentityFile: "/home/user/.databricks/ssh-tunnel-keys/abc-123",
		ProxyCommand: `"/usr/local/bin/databricks" ssh connect --proxy --cluster=abc-123`,
		ControlPath:  "~/.databricks/ssh-sockets/%h",
	})

	assert.Contains(t, config, "Host my-cluster")
	assert.Contains(t, config, "User root")
	assert.Contains(t, config, `ProxyCommand "/usr/local/bin/databricks" ssh connect --proxy --cluster=abc-123`)

	if runtime.GOOS == "windows" {
		assert.NotContains(t, config, "ControlMaster")
		assert.NotContains(t, config, "ControlPath")
		assert.NotContains(t, config, "ControlPersist")
	} else {
		assert.Contains(t, config, "ControlMaster auto")
		assert.Contains(t, config, "ControlPath ~/.databricks/ssh-sockets/%h")
		assert.Contains(t, config, "ControlPersist 10m")
	}
}
