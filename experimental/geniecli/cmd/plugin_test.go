package genieclicmd

import (
	"testing"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginRecorded(t *testing.T) {
	home := t.TempDir()
	ctx := env.Set(t.Context(), "HOME", home)
	// On Windows env.UserHomeDir reads USERPROFILE.
	ctx = env.Set(ctx, "USERPROFILE", home)

	// No state yet.
	got, err := pluginRecorded(ctx, "codex")
	require.NoError(t, err)
	assert.False(t, got)

	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)

	// State without the agent's plugin.
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{
		Plugins: map[string]installer.PluginRecord{"claude-code": {Plugin: "databricks"}},
	}))
	got, err = pluginRecorded(ctx, "codex")
	require.NoError(t, err)
	assert.False(t, got)

	// State with the agent's plugin recorded.
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{
		Plugins: map[string]installer.PluginRecord{"codex": {Plugin: "databricks"}},
	}))
	got, err = pluginRecorded(ctx, "codex")
	require.NoError(t, err)
	assert.True(t, got)
}

func TestEnsureDatabricksPluginSkipsNonPluginHarness(t *testing.T) {
	home := t.TempDir()
	ctx := env.Set(t.Context(), "HOME", home)
	ctx = env.Set(ctx, "USERPROFILE", home)

	// opencode is skills-only (no headless plugin) and gemini is unknown to the
	// aitools registry; both must be no-ops rather than errors or installs.
	require.NoError(t, ensureDatabricksPlugin(ctx, "opencode"))
	require.NoError(t, ensureDatabricksPlugin(ctx, "gemini"))
}
