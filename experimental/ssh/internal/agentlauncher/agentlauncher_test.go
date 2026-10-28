package agentlauncher

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
	// Each agent injects context by exactly one mechanism (flag OR home file);
	// the two are never both set.
	expect := map[string]struct{ flag, home string }{
		"claude": {flag: "--append-system-prompt-file"},
		"codex":  {home: ".codex/AGENTS.md"},
	}
	require.Len(t, SupportedAgents, len(expect))
	for name, want := range expect {
		a, ok := AgentByName(name)
		require.True(t, ok)
		assert.Equal(t, want.flag, a.ContextFlag, "%s ContextFlag", name)
		assert.Equal(t, want.home, a.ContextHomeFile, "%s ContextHomeFile", name)
	}

	_, ok := AgentByName("not-an-agent")
	assert.False(t, ok)
}
