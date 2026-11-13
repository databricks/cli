package sandbox_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteHostConfigPreservesUnrelatedConfig(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	mainConfig := filepath.Join(home, ".ssh", "config")
	require.NoError(t, os.MkdirAll(filepath.Dir(mainConfig), 0o700))
	original := "Host existing\n    HostName existing.test\n"
	require.NoError(t, os.WriteFile(mainConfig, []byte(original), 0o600))
	identity := &sandbox.Identity{KeyPath: filepath.Join(home, "private key"), KnownHostsPath: filepath.Join(home, "known hosts"), HostKeyAlias: "sandbox-identity"}
	filename, err := sandbox.WriteHostConfig(ctx, "my-sandbox", mainConfig, "databricks sandbox ssh sandboxes/test --proxy", identity, false)
	require.NoError(t, err)
	config, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Contains(t, string(config), "Host my-sandbox")
	assert.Contains(t, string(config), "User sandbox-agent")
	assert.Contains(t, string(config), "ConnectTimeout 900")
	assert.Contains(t, string(config), "HostKeyAlias sandbox-identity")
	assert.Contains(t, string(config), "StrictHostKeyChecking yes")
	assert.Contains(t, string(config), "ProxyCommand databricks sandbox ssh sandboxes/test --proxy")
	_, err = sandbox.WriteHostConfig(ctx, "my-sandbox", mainConfig, "databricks sandbox ssh sandboxes/test --proxy", identity, false)
	require.NoError(t, err)
	main, err := os.ReadFile(mainConfig)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(main), "Include "))
	assert.Contains(t, string(main), original)
	_, err = sandbox.WriteHostConfig(ctx, "my-sandbox", mainConfig, "different proxy", identity, false)
	require.ErrorContains(t, err, "already exists")
	unchanged, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Equal(t, config, unchanged)
	_, err = sandbox.WriteHostConfig(ctx, "my-sandbox", mainConfig, "different proxy", identity, true)
	require.NoError(t, err)
	changed, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Contains(t, string(changed), "ProxyCommand different proxy")
	if ssh, err := exec.LookPath("ssh"); err == nil {
		if runtime.GOOS == "windows" {
			// Git Bash's ssh does not resolve Windows drive paths in Include directives.
			ssh, err = exec.LookPath(filepath.Join(os.Getenv("SystemRoot"), "System32", "OpenSSH", "ssh.exe"))
			require.NoError(t, err)
		}
		output, err := exec.CommandContext(t.Context(), ssh, "-G", "-F", mainConfig, "my-sandbox").CombinedOutput()
		require.NoError(t, err, string(output))
		assert.Contains(t, string(output), "user sandbox-agent")
		assert.Contains(t, string(output), "proxycommand different proxy")
	}
}

func TestInvalidHostAliasesCannotWriteConfig(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	for _, alias := range []string{"", "../escape", "a/b", "-flag", "host *", "host\nHost *", "CON", "com1.txt", "trailing.", "a%h"} {
		_, err := sandbox.WriteHostConfig(ctx, alias, "", "proxy", &sandbox.Identity{}, true)
		require.Error(t, err, alias)
	}
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
