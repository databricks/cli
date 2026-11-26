package aitools

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
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

// Install state that an unfinished first install of auto-install saved is
// finished as a first install. Install state that anything else wrote, such
// as a later `aitools install --skills-only`, is updated: a first install
// would put Claude Code on the plugin and drop the user's choices.
func TestPlanToolsUpdatesInstallStateAnUnfinishedInstallDidNotWrite(t *testing.T) {
	autoInstallHome(t)
	dir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	saved := &installer.InstallState{
		SchemaVersion: installer.CurrentSchemaVersion,
		Release:       "v0.2.6",
		Skills:        map[string]string{"databricks-sql": "1.0.0"},
		Scope:         installer.ScopeGlobal,
	}
	require.NoError(t, installer.SaveState(dir, saved))
	loaded, err := installer.LoadState(dir)
	require.NoError(t, err)
	digest, err := installStateDigest(loaded)
	require.NoError(t, err)
	prior := autoinstall.State{Installing: true, InstallStateDigest: digest}

	tools, err := planTools(t.Context(), prior)
	require.NoError(t, err)
	assert.True(t, tools.fresh(), "the install state is the unfinished install's own")

	// The MCP section is written by the mcp commands and does not count.
	require.NoError(t, installer.SaveMCPState(dir, &installer.MCPState{Servers: []installer.MCPEntry{{Name: "databricks"}}}))
	tools, err = planTools(t.Context(), prior)
	require.NoError(t, err)
	assert.True(t, tools.fresh(), "an MCP record keeps the install state the unfinished install's own")

	saved.SkillsOnlyAgents = []string{agents.NameClaudeCode}
	require.NoError(t, installer.SaveState(dir, saved))
	tools, err = planTools(t.Context(), prior)
	require.NoError(t, err)
	assert.False(t, tools.fresh())
	require.NotNil(t, tools.existing, "the user's install state is updated")
	assert.Equal(t, []string{agents.NameClaudeCode}, tools.existing.SkillsOnlyAgents)

	// Install state saved while Installing is set, by a CLI that recorded no
	// digest, is not known to be the unfinished install's own either.
	tools, err = planTools(t.Context(), autoinstall.State{Installing: true})
	require.NoError(t, err)
	require.NotNil(t, tools.existing)
}

func TestPlanToolsKeepsPluginInstalledWithAgentCLI(t *testing.T) {
	// The records Codex CLI 0.171.0 and Copilot CLI 1.0.x write for
	// `codex plugin add` and `copilot plugin install`. Each records the
	// plugin both in its config and in its plugin cache. {{home}} stands for
	// the home directory.
	tests := []struct {
		name  string
		file  string
		body  string
		cache string
	}{
		{
			name: "codex",
			file: filepath.Join(".codex", "config.toml"),
			body: `[plugins."databricks@databricks-agent-skills"]
enabled = true
`,
			cache: filepath.Join(".codex", "plugins", "cache", "databricks-agent-skills", "databricks", "0.2.50"),
		},
		{
			name: "copilot",
			file: filepath.Join(".copilot", "config.json"),
			body: `// This file is managed automatically.
{"installedPlugins": [{"name": "databricks", "marketplace": "databricks-agent-skills", "version": "0.2.50", "cache_path": "{{home}}/.copilot/installed-plugins/databricks-agent-skills/databricks"}]}
`,
			cache: filepath.Join(".copilot", "installed-plugins", "databricks-agent-skills", "databricks"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := autoInstallHome(t)
			require.NoError(t, os.RemoveAll(filepath.Join(home, ".cursor")))
			require.NoError(t, os.MkdirAll(filepath.Join(home, filepath.Dir(tt.file)), 0o755))
			body := strings.ReplaceAll(tt.body, "{{home}}", filepath.ToSlash(home))
			require.NoError(t, os.WriteFile(filepath.Join(home, tt.file), []byte(body), 0o644))
			require.NoError(t, os.MkdirAll(filepath.Join(home, tt.cache), 0o755))
			// Claude Code installs through its CLI, which is not run here, so
			// the plan is the agent whose CLI is not on PATH.
			setAgentCLIsStoppable(t, false)

			tools, err := planTools(t.Context(), autoinstall.State{})
			require.NoError(t, err)
			require.Len(t, tools.plan, 1)
			assert.Equal(t, deliveryPlugin, tools.plan[0].delivery)
			assert.True(t, tools.plan[0].cliMissing)
			assert.Empty(t, tools.newAgents())

			forbidSkillsInstall(t)
			// Raw skills an earlier run gave the agent would show every skill
			// twice next to the plugin, so they are removed.
			var cleaned []string
			origCleanup := cleanupLegacyFn
			t.Cleanup(func() { cleanupLegacyFn = origCleanup })
			cleanupLegacyFn = func(_ context.Context, a *agents.Agent, scope string) error {
				cleaned = append(cleaned, a.Name+" "+scope)
				return nil
			}
			result, err := tools.install(t.Context())
			require.NoError(t, err)
			assert.Equal(t, protos.AitoolsAutoInstallOutcomeInstalled, result.outcome)
			assert.Empty(t, result.agents)
			assert.Equal(t, []string{tools.plan[0].agent.Name + " " + installer.ScopeGlobal}, cleaned)
		})
	}
}

// A plugin record that outlived the agent's plugin cache is not an installed
// plugin: the agent loads none, so it gets raw skills. Codex keeps a
// [plugins."databricks@..."] table, Copilot an installedPlugins entry whose
// cache_path is gone (github/copilot-cli#4039).
func TestPlanToolsGivesSkillsForPluginRecordWithoutCache(t *testing.T) {
	tests := []struct {
		agent string
		file  string
		body  string
	}{
		{
			agent: agents.NameCodex,
			file:  filepath.Join(".codex", "config.toml"),
			body: `[plugins."databricks@databricks-agent-skills"]
enabled = true
`,
		},
		{
			agent: agents.NameCopilot,
			file:  filepath.Join(".copilot", "config.json"),
			body: `// This file is managed automatically.
{"installedPlugins": [{"name": "databricks", "marketplace": "databricks-agent-skills", "version": "0.2.50", "cache_path": "{{home}}/.copilot/installed-plugins/databricks-agent-skills/databricks"}]}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			home := autoInstallHome(t)
			require.NoError(t, os.RemoveAll(filepath.Join(home, ".cursor")))
			config := filepath.Join(home, tt.file)
			require.NoError(t, os.MkdirAll(filepath.Dir(config), 0o755))
			body := strings.ReplaceAll(tt.body, "{{home}}", filepath.ToSlash(home))
			require.NoError(t, os.WriteFile(config, []byte(body), 0o644))
			setAgentCLIsStoppable(t, false)

			tools, err := planTools(t.Context(), autoinstall.State{})
			require.NoError(t, err)
			require.Len(t, tools.plan, 1)
			assert.Equal(t, tt.agent, tools.plan[0].agent.Name)
			assert.Equal(t, deliverySkills, tools.plan[0].delivery)
		})
	}
}

// A first install keeps a skill that the user put into a skills agent's own
// directory: the installer refuses the skills, which the run reports as kept,
// unless a plugin agent got its plugin. Either way the install is unfinished
// (see TestAutoInstallFinishesAMixedInstallRefusedTheSkills for the run after
// it).
func TestInstallKeepsHandPlacedSkills(t *testing.T) {
	home := autoInstallHome(t)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.1.0")
	setAgentCLIsStoppable(t, true)
	plugins := setupPluginMock(t)
	orig := installSkillsForAgentsFn
	t.Cleanup(func() { installSkillsForAgentsFn = orig })
	installSkillsForAgentsFn = func(_ context.Context, _ installer.ManifestSource, _ []*agents.Agent, opts installer.InstallOptions) error {
		assert.True(t, opts.KeepHandPlacedSkills)
		return &installer.RefusedError{Err: errors.New("kept the skill")}
	}

	tools, err := planTools(t.Context(), autoinstall.State{})
	require.NoError(t, err)
	require.Equal(t, []string{"Claude Code", "Cursor"}, planAgentNames(tools.plan))
	result, err := tools.install(t.Context())
	require.NoError(t, err)
	assert.Equal(t, protos.AitoolsAutoInstallOutcomeInstalled, result.outcome)
	assert.Equal(t, []string{"Claude Code"}, result.agents)
	assert.True(t, result.unfinished, "Cursor did not get the skills")
	assert.Len(t, *plugins, 1)

	require.NoError(t, os.RemoveAll(filepath.Join(home, ".claude")))
	tools, err = planTools(t.Context(), autoinstall.State{})
	require.NoError(t, err)
	require.Equal(t, []string{"Cursor"}, planAgentNames(tools.plan))
	_, err = tools.install(t.Context())
	assert.ErrorAs(t, err, new(*installer.RefusedError))
}

// An unfinished first install is planned again without the plugin agents that
// got their plugin: their CLI is not run again, and the agents are not named
// as installed again. A first install that is not unfinished keeps them.
func TestPlanToolsLeavesOutPluginsAnUnfinishedInstallGave(t *testing.T) {
	autoInstallHome(t)
	setAgentCLIsStoppable(t, true)
	stubPluginAlreadyInstalled(t, true)

	tools, err := planTools(t.Context(), autoinstall.State{Installing: true})
	require.NoError(t, err)
	assert.Equal(t, []string{"Cursor"}, planAgentNames(tools.plan))
	assert.Equal(t, []string{"Cursor"}, tools.newAgents())

	tools, err = planTools(t.Context(), autoinstall.State{})
	require.NoError(t, err)
	assert.Equal(t, []string{"Claude Code", "Cursor"}, planAgentNames(tools.plan))
}

// When every agent left in an unfinished first install got its plugin, there
// is nothing to finish.
func TestPlanToolsFinishesAnUnfinishedInstallThatGaveEveryPlugin(t *testing.T) {
	home := autoInstallHome(t)
	require.NoError(t, os.RemoveAll(filepath.Join(home, ".cursor")))
	setAgentCLIsStoppable(t, true)
	stubPluginAlreadyInstalled(t, true)

	tools, err := planTools(t.Context(), autoinstall.State{Installing: true})
	require.NoError(t, err)
	assert.Equal(t, protos.AitoolsAutoInstallOutcomeUpToDate, tools.skip)
	assert.False(t, tools.fresh())
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

// The update runs the plugin updates before it refuses the raw skills, so a
// refusal still reports the plugins it updated.
func TestUpdateToolsReportsPluginsUpdatedBeforeARefusal(t *testing.T) {
	autoInstallHome(t)
	noTelemetryAuth(t)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.3.0")
	resetUpdatePluginSeams(t)
	dir := saveGlobalState(t, map[string]string{"databricks-sql": "0.1.0"})
	before, err := installer.LoadState(dir)
	require.NoError(t, err)

	origUpdateSkills := updateSkillsFn
	t.Cleanup(func() { updateSkillsFn = origUpdateSkills })
	updateSkillsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.UpdateOptions) (*installer.UpdateResult, error) {
		return nil, &installer.RefusedError{Err: errors.New("keeping the installed skills")}
	}
	pluginUpdated := false
	updatePluginsFn = func(context.Context, string, string) ([]installer.PluginUpdate, error) {
		if !pluginUpdated {
			return nil, nil
		}
		state, err := installer.LoadState(dir)
		require.NoError(t, err)
		rec := state.Plugins[agents.NameClaudeCode]
		rec.Version = "0.3.0"
		state.Plugins[agents.NameClaudeCode] = rec
		require.NoError(t, installer.SaveState(dir, state))
		return []installer.PluginUpdate{{Agent: "Claude Code", Version: "0.3.0"}}, nil
	}
	ctx := cmdio.MockDiscard(t.Context())

	// Nothing changed: the refusal is returned, which auto-install reports
	// as kept.
	_, err = updateTools(ctx, dir, before)
	assert.ErrorAs(t, err, new(*installer.RefusedError))

	pluginUpdated = true
	result, err := updateTools(ctx, dir, before)
	require.NoError(t, err)
	assert.Equal(t, protos.AitoolsAutoInstallOutcomeInstalled, result.outcome)
	assert.Equal(t, []string{"Claude Code"}, result.agents)
}
