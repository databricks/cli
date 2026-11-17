package aitools

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// autoInstallHome isolates HOME with Claude Code (its CLI on PATH) and Cursor
// detected.
func autoInstallHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	isolateAgents(t, "HOME", tmp, "USERPROFILE", tmp, "XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))
	fakeBinsOnPath(t, "claude")
	return tmp
}

func setAgentCLIsStoppable(t *testing.T, v bool) {
	t.Helper()
	orig := agentCLIsStoppable
	t.Cleanup(func() { agentCLIsStoppable = orig })
	agentCLIsStoppable = v
}

func planAgentNames(plan []agentPlanItem) []string {
	var names []string
	for _, it := range plan {
		names = append(names, it.agent.DisplayName)
	}
	return names
}

func TestPlanToolsRunsNoAgentCLIWhereTimeoutsCannotStopThem(t *testing.T) {
	autoInstallHome(t)

	setAgentCLIsStoppable(t, true)
	tools, err := planTools(t.Context(), autoinstall.State{})
	require.NoError(t, err)
	assert.Equal(t, []string{"Claude Code", "Cursor"}, planAgentNames(tools.plan))

	// The plugin install would run Claude Code's CLI.
	setAgentCLIsStoppable(t, false)
	tools, err = planTools(t.Context(), autoinstall.State{})
	require.NoError(t, err)
	assert.Equal(t, []string{"Cursor"}, planAgentNames(tools.plan))
	assert.Empty(t, tools.skip)

	// Updating the recorded plugins would run agent CLIs too.
	saveGlobalState(t, map[string]string{"databricks-sql": "0.1.0"})
	tools, err = planTools(t.Context(), autoinstall.State{})
	require.NoError(t, err)
	assert.Nil(t, tools.existing)
	assert.Equal(t, protos.AitoolsAutoInstallOutcomeSkippedPlatform, tools.skip)

	setAgentCLIsStoppable(t, true)
	tools, err = planTools(t.Context(), autoinstall.State{})
	require.NoError(t, err)
	assert.NotNil(t, tools.existing)
	assert.Empty(t, tools.skip)
}

func TestPlanToolsReportsWhyNothingIsInstalled(t *testing.T) {
	home := autoInstallHome(t)

	// Claude Code alone installs through its CLI, which is not run here.
	require.NoError(t, os.RemoveAll(filepath.Join(home, ".cursor")))
	setAgentCLIsStoppable(t, false)
	tools, err := planTools(t.Context(), autoinstall.State{})
	require.NoError(t, err)
	assert.Equal(t, protos.AitoolsAutoInstallOutcomeSkippedPlatform, tools.skip)

	require.NoError(t, os.RemoveAll(filepath.Join(home, ".claude")))
	tools, err = planTools(t.Context(), autoinstall.State{})
	require.NoError(t, err)
	assert.Equal(t, protos.AitoolsAutoInstallOutcomeSkippedNoAgents, tools.skip)
}

func TestPlanToolsKeepsSkillsWithoutInstallState(t *testing.T) {
	autoInstallHome(t)
	dir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	// No checksums record what the CLI wrote, so edits cannot be told apart.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "databricks-sql"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks-sql", "SKILL.md"), []byte("user edit"), 0o644))

	tools, err := planTools(t.Context(), autoinstall.State{})
	require.NoError(t, err)
	assert.Equal(t, protos.AitoolsAutoInstallOutcomeKept, tools.skip)
	assert.Empty(t, tools.plan)

	result, err := tools.install(t.Context())
	require.NoError(t, err)
	assert.Equal(t, protos.AitoolsAutoInstallOutcomeKept, result.outcome)
	assert.Empty(t, result.agents)
	data, err := os.ReadFile(filepath.Join(dir, "databricks-sql", "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, "user edit", string(data))

	// Skills that an unfinished install of auto-install left are its own, so
	// a later run finishes the install.
	tools, err = planTools(t.Context(), autoinstall.State{Installing: true})
	require.NoError(t, err)
	assert.Empty(t, tools.skip)
	assert.True(t, tools.fresh())
}

func TestChangedAgents(t *testing.T) {
	autoInstallHome(t)
	dir := saveGlobalState(t, map[string]string{"databricks-sql": "0.1.0"})
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "databricks-sql"), 0o755))
	before, err := installer.LoadState(dir)
	require.NoError(t, err)

	// A new release in which every outdated skill was kept changed nothing.
	after := *before
	after.Release = "v0.3.0"
	assert.Empty(t, changedAgents(t.Context(), before, &after))

	// Only the agent whose plugin was updated changed.
	after.Plugins = maps.Clone(before.Plugins)
	rec := after.Plugins[agents.NameClaudeCode]
	rec.Version = "0.3.0"
	after.Plugins[agents.NameClaudeCode] = rec
	assert.Equal(t, []string{"Claude Code"}, changedAgents(t.Context(), before, &after))
}

func TestSkipsAutoInstall(t *testing.T) {
	root := &cobra.Command{Use: "databricks"}
	add := func(parent *cobra.Command, name string) *cobra.Command {
		c := &cobra.Command{Use: name}
		parent.AddCommand(c)
		return c
	}
	jobsList := add(add(root, "jobs"), "list")
	auth := add(root, "auth")
	authToken := add(auth, "token")
	dockerToken := add(add(auth, "docker"), "token")
	authDescribe := add(auth, "describe")
	aitoolsList := add(add(root, "aitools"), "list")
	experimentalAitools := add(add(add(root, "experimental"), "aitools"), "tools")
	version := add(root, "version")
	completion := add(add(root, "completion"), "zsh")
	complete := add(root, cobra.ShellCompRequestCmd)

	for _, cmd := range []*cobra.Command{authToken, dockerToken, aitoolsList, experimentalAitools, version, completion, complete} {
		assert.True(t, skipsAutoInstall(cmd), cmd.CommandPath())
	}
	for _, cmd := range []*cobra.Command{jobsList, authDescribe} {
		assert.False(t, skipsAutoInstall(cmd), cmd.CommandPath())
	}
}
