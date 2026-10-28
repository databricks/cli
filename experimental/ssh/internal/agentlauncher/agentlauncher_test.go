package agentlauncher

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScript(t *testing.T) {
	got := script("/Workspace/Users/o'brien@example.com/databricks", "/Workspace/Users/o'brien@example.com", "codex")
	assert.Equal(t, `#!/usr/bin/env bash
# "codex" launcher from "databricks ssh connect"; delegates to "ssh agent-shim codex".
export DATABRICKS_WORKSPACE_HOME='/Workspace/Users/o'\''brien@example.com'
exec '/Workspace/Users/o'\''brien@example.com/databricks' ssh agent-shim codex "$@"
`, got)
}

func TestInstall(t *testing.T) {
	const wsHome = "/Workspace/Users/me@example.com"
	home := t.TempDir()
	t.Setenv(env.HomeEnvVar(), home)
	require.NoError(t, Install(t.Context(), "/old/databricks", wsHome))

	for _, agent := range SupportedAgents {
		path := filepath.Join(home, BinDir, agent.Name)
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, script("/old/databricks", wsHome, agent.Name), string(content))
		if runtime.GOOS != "windows" {
			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
		}
	}

	// A server restart from a new binary replaces the launchers in place.
	require.NoError(t, Install(t.Context(), "/new/databricks", wsHome))
	content, err := os.ReadFile(filepath.Join(home, BinDir, "claude"))
	require.NoError(t, err)
	assert.Equal(t, script("/new/databricks", wsHome, "claude"), string(content))
}

func TestInstallContinuesPastFailure(t *testing.T) {
	const wsHome = "/Workspace/Users/me@example.com"
	home := t.TempDir()
	t.Setenv(env.HomeEnvVar(), home)
	// A non-empty directory where the claude launcher goes can't be replaced by a file.
	require.NoError(t, os.MkdirAll(filepath.Join(home, BinDir, "claude", "blocker"), 0o755))

	err := Install(t.Context(), "/databricks", wsHome)
	assert.ErrorContains(t, err, "failed to write the claude launcher")
	assert.NotContains(t, err.Error(), "codex")

	content, err := os.ReadFile(filepath.Join(home, BinDir, "codex"))
	require.NoError(t, err)
	assert.Equal(t, script("/databricks", wsHome, "codex"), string(content))
}

func TestSupportedAgents(t *testing.T) {
	for _, name := range []string{"claude", "codex"} {
		a, ok := AgentByName(name)
		require.True(t, ok)
		assert.NotNil(t, a.ContextArg, "%s ContextArg", name)
	}
	assert.Len(t, SupportedAgents, 2)

	_, ok := AgentByName("not-an-agent")
	assert.False(t, ok)
}

func TestClaudeContextArg(t *testing.T) {
	home := t.TempDir()
	args, err := claudeContextArg(home, "the context")
	require.NoError(t, err)
	path := filepath.Join(home, RootDir, ContextFile)
	assert.Equal(t, []string{"--append-system-prompt-file", path}, args)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "the context", string(data))
}

func TestCodexContextArg(t *testing.T) {
	home := t.TempDir()
	const systemContext = "line \"one\"\nline two"
	args, err := codexContextArg(home, systemContext)
	require.NoError(t, err)
	require.Len(t, args, 2)
	assert.Equal(t, "-c", args[0])
	// codex splits the override on the first '=' and parses the rest as TOML
	var got map[string]string
	require.NoError(t, toml.Unmarshal([]byte(args[1]), &got))
	assert.Equal(t, map[string]string{"developer_instructions": systemContext}, got)
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
