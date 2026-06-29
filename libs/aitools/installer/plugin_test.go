package installer

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubAgentLookPath(t *testing.T, found bool) {
	t.Helper()
	orig := lookPath
	lookPath = func(name string) (string, error) {
		if found {
			return filepath.Join("/usr/bin", name), nil
		}
		return "", exec.ErrNotFound
	}
	t.Cleanup(func() { lookPath = orig })
}

func pluginAgent(name, display, binary string) *agents.Agent {
	return &agents.Agent{
		Name:        name,
		DisplayName: display,
		Binary:      binary,
		Plugin: &agents.PluginSpec{
			Marketplace: "databricks-agent-skills",
			ID:          "databricks",
			Source:      "databricks/databricks-agent-skills",
		},
	}
}

func claudeAgent() *agents.Agent { return pluginAgent(agents.NameClaudeCode, "Claude Code", "claude") }
func codexAgent() *agents.Agent  { return pluginAgent(agents.NameCodex, "Codex CLI", "codex") }

func noPluginAgent() *agents.Agent {
	return &agents.Agent{Name: agents.NameOpenCode, DisplayName: "OpenCode", Binary: "opencode"}
}

func TestInstallPluginForAgentClaudeSuccess(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	rec, err := InstallPluginForAgent(ctx, claudeAgent(), "user", "v0.2.6")
	require.NoError(t, err)
	assert.Equal(t, "databricks-agent-skills", rec.Marketplace)
	assert.Equal(t, "databricks", rec.Plugin)
	assert.Equal(t, "user", rec.Scope)
	assert.Equal(t, "0.2.6", rec.Version)
	assert.True(t, rec.InstalledMarketplace)

	cmds := stub.Commands()
	assert.Contains(t, cmds, "claude plugin --help")
	assert.Contains(t, cmds, "claude plugin marketplace add databricks/databricks-agent-skills")
	assert.Contains(t, cmds, "claude plugin install databricks@databricks-agent-skills --scope user")
}

func TestInstallPluginForAgentBuiltinMarketplace(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	// An agent whose plugin lives in a built-in marketplace (empty Source) like
	// Claude's claude-plugins-official: install from it, never register it.
	agent := &agents.Agent{
		Name:        agents.NameClaudeCode,
		DisplayName: "Claude Code",
		Binary:      "claude",
		Plugin:      &agents.PluginSpec{Marketplace: "claude-plugins-official", ID: "databricks", Source: ""},
	}

	rec, err := InstallPluginForAgent(ctx, agent, "user", "main")
	require.NoError(t, err)
	assert.Equal(t, "claude-plugins-official", rec.Marketplace)
	assert.False(t, rec.InstalledMarketplace, "a built-in marketplace is never registered by us")

	cmds := stub.Commands()
	for _, c := range cmds {
		assert.NotContains(t, c, "marketplace add", "must not register a built-in marketplace")
	}
	assert.Contains(t, cmds, "claude plugin install databricks@claude-plugins-official --scope user")
}

func TestInstallPluginForAgentCodexUsesAddNoScope(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	_, err := InstallPluginForAgent(ctx, codexAgent(), "user", "v0.2.6")
	require.NoError(t, err)

	cmds := stub.Commands()
	assert.Contains(t, cmds, "codex plugin add databricks@databricks-agent-skills")
	for _, c := range cmds {
		assert.NotContains(t, c, "--scope")
	}
}

func TestInstallPluginForAgentNoPlugin(t *testing.T) {
	_, err := InstallPluginForAgent(t.Context(), noPluginAgent(), "user", "v0.2.6")
	var be *BlockedError
	require.ErrorAs(t, err, &be)
	assert.Equal(t, ReasonNoPlugin, be.Reason)
}

func TestInstallPluginForAgentCLINotOnPath(t *testing.T) {
	stubAgentLookPath(t, false)
	ctx, stub := process.WithStub(t.Context())

	_, err := InstallPluginForAgent(ctx, claudeAgent(), "user", "v0.2.6")
	var be *BlockedError
	require.ErrorAs(t, err, &be)
	assert.Equal(t, ReasonCLINotOnPath, be.Reason)
	assert.Equal(t, 0, stub.Len(), "a missing CLI must not be executed")
}

func TestInstallPluginForAgentInstallFails(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	stub.WithStderrFor("claude plugin install", "you must run `copilot login`").
		WithFailureFor("claude plugin install", errors.New("exit status 1"))

	_, err := InstallPluginForAgent(ctx, claudeAgent(), "user", "v0.2.6")
	var be *BlockedError
	require.ErrorAs(t, err, &be)
	assert.Equal(t, ReasonInstallFailed, be.Reason)
	// The agent's own stderr is surfaced verbatim, not classified by us.
	assert.Equal(t, "you must run `copilot login`", be.Detail)
}

func TestInstallPluginForAgentMarketplaceAlreadyPresent(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	// The marketplace is already registered, so even though `add` succeeds we must
	// not claim ownership (and must not de-register it on uninstall).
	stub.WithStdoutFor("plugin marketplace list", "databricks-agent-skills\n")

	rec, err := InstallPluginForAgent(ctx, claudeAgent(), "user", "v0.2.6")
	require.NoError(t, err)
	assert.False(t, rec.InstalledMarketplace, "a pre-existing marketplace must not be recorded as ours")
}

func TestInstallPluginRollsBackMarketplaceOnInstallFailure(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	// Marketplace absent (empty list) so we add it; then the plugin install fails.
	stub.WithFailureFor("plugin install", errors.New("boom"))

	_, err := InstallPluginForAgent(ctx, claudeAgent(), "user", "v0.2.6")
	var be *BlockedError
	require.ErrorAs(t, err, &be)
	assert.Equal(t, ReasonInstallFailed, be.Reason)
	// We added the marketplace, so a failed install must de-register it again.
	assert.Contains(t, stub.Commands(), "claude plugin marketplace remove databricks-agent-skills")
}

func TestUpdatePluginForAgentCodexTwoStep(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	require.NoError(t, UpdatePluginForAgent(ctx, codexAgent()))
	cmds := stub.Commands()
	assert.Contains(t, cmds, "codex plugin marketplace upgrade")
	assert.Contains(t, cmds, "codex plugin add databricks@databricks-agent-skills")
}

func TestUpdatePluginForAgentClaudeOneStep(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	require.NoError(t, UpdatePluginForAgent(ctx, claudeAgent()))
	assert.Contains(t, stub.Commands(), "claude plugin update databricks@databricks-agent-skills")
}

func TestUninstallPluginDeregistersMarketplaceWhenInstalledByUs(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	rec := PluginRecord{Marketplace: "databricks-agent-skills", Plugin: "databricks", InstalledMarketplace: true}
	require.NoError(t, UninstallPluginForAgent(ctx, claudeAgent(), rec, false))

	cmds := stub.Commands()
	assert.Contains(t, cmds, "claude plugin uninstall databricks@databricks-agent-skills")
	assert.Contains(t, cmds, "claude plugin marketplace remove databricks-agent-skills")
}

func TestUninstallPluginKeepsSharedMarketplace(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	rec := PluginRecord{Marketplace: "databricks-agent-skills", Plugin: "databricks", InstalledMarketplace: false}
	require.NoError(t, UninstallPluginForAgent(ctx, claudeAgent(), rec, false))

	for _, c := range stub.Commands() {
		assert.NotContains(t, c, "marketplace remove")
	}
}

func TestUninstallSkillsOptsTearsDownPlugin(t *testing.T) {
	setupTestHome(t)
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	ctx, stderr := cmdio.NewTestContextWithStderr(ctx)

	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Release:       "v0.2.6",
		Plugins: map[string]PluginRecord{
			agents.NameCopilot: {Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user", InstalledMarketplace: true},
		},
	}))

	require.NoError(t, UninstallSkillsOpts(ctx, UninstallOptions{Scope: ScopeGlobal}))

	cmds := stub.Commands()
	assert.Contains(t, cmds, "copilot plugin uninstall databricks@databricks-agent-skills")
	assert.Contains(t, cmds, "copilot plugin marketplace remove databricks-agent-skills")
	assert.Contains(t, stderr.String(), "Uninstalled the plugin from 1 agent.")

	// State file removed since nothing remains.
	state, err := LoadState(dir)
	require.NoError(t, err)
	assert.Nil(t, state)
}

func TestUninstallNeverDeregistersBuiltinMarketplace(t *testing.T) {
	setupTestHome(t)
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	ctx = cmdio.MockDiscard(ctx)

	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	// Claude installs from its built-in claude-plugins-official marketplace; even a
	// stale InstalledMarketplace=true must never trigger a de-register.
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Plugins: map[string]PluginRecord{
			agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks", InstalledMarketplace: true},
		},
	}))

	require.NoError(t, UninstallSkillsOpts(ctx, UninstallOptions{Scope: ScopeGlobal}))

	cmds := stub.Commands()
	assert.Contains(t, cmds, "claude plugin uninstall databricks@claude-plugins-official")
	for _, c := range cmds {
		assert.NotContains(t, c, "marketplace remove")
	}
}

func TestUninstallKeepMarketplace(t *testing.T) {
	setupTestHome(t)
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	ctx = cmdio.MockDiscard(ctx)

	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Plugins: map[string]PluginRecord{
			agents.NameCopilot: {Marketplace: "databricks-agent-skills", Plugin: "databricks", InstalledMarketplace: true},
		},
	}))

	require.NoError(t, UninstallSkillsOpts(ctx, UninstallOptions{Scope: ScopeGlobal, KeepMarketplace: true}))

	for _, c := range stub.Commands() {
		assert.NotContains(t, c, "marketplace remove")
	}
}

func TestUninstallKeepsUnknownAgentRecord(t *testing.T) {
	setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())

	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Plugins: map[string]PluginRecord{
			"mystery-agent": {Marketplace: "databricks-agent-skills", Plugin: "databricks"},
		},
	}))

	// An unknown agent can't be torn down; its record (and the state file) must be
	// kept rather than silently dropped while the plugin may still exist.
	require.NoError(t, UninstallSkillsOpts(ctx, UninstallOptions{Scope: ScopeGlobal}))

	st, err := LoadState(dir)
	require.NoError(t, err)
	require.NotNil(t, st)
	assert.Contains(t, st.Plugins, "mystery-agent")
}

func TestUninstallClearsRecordWhenMarketplaceRemoveFails(t *testing.T) {
	setupTestHome(t)
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	// The plugin uninstall succeeds; only the marketplace de-register fails.
	stub.WithFailureFor("plugin marketplace remove", errors.New("boom"))
	ctx = cmdio.MockDiscard(ctx)

	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Plugins: map[string]PluginRecord{
			agents.NameCopilot: {Marketplace: "databricks-agent-skills", Plugin: "databricks", InstalledMarketplace: true},
		},
	}))

	require.NoError(t, UninstallSkillsOpts(ctx, UninstallOptions{Scope: ScopeGlobal}))

	// The plugin is gone, so the record is cleared even though the marketplace
	// de-register failed (otherwise a retry would be stuck).
	st, err := LoadState(dir)
	require.NoError(t, err)
	assert.Nil(t, st)
}

func TestUpdateInstalledPlugins(t *testing.T) {
	setupTestHome(t)
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Release:       "v0.2.6",
		Plugins: map[string]PluginRecord{
			agents.NameCopilot: {Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user", Version: "0.2.6"},
		},
	}))

	updated, err := UpdateInstalledPlugins(ctx, ScopeGlobal, "v0.2.7")
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Equal(t, "GitHub Copilot", updated[0].Agent)
	assert.Equal(t, "0.2.7", updated[0].Version)
	assert.Contains(t, stub.Commands(), "copilot plugin update databricks@databricks-agent-skills")

	state, err := LoadState(dir)
	require.NoError(t, err)
	assert.Equal(t, "0.2.7", state.Plugins[agents.NameCopilot].Version)
}

func TestUninstallPluginKeepMarketplaceFlag(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	rec := PluginRecord{Marketplace: "databricks-agent-skills", Plugin: "databricks", InstalledMarketplace: true}
	require.NoError(t, UninstallPluginForAgent(ctx, claudeAgent(), rec, true))

	for _, c := range stub.Commands() {
		assert.NotContains(t, c, "marketplace remove")
	}
}

func TestRecordPluginInstallsThenSkillsInstallNoPanic(t *testing.T) {
	tmp := setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)

	// A pure-plugin install creates state with no skills.
	require.NoError(t, RecordPluginInstalls(ctx, ScopeGlobal, map[string]PluginRecord{
		agents.NameClaudeCode: {Marketplace: "databricks-agent-skills", Plugin: "databricks", Version: "0.2.6"},
	}, "v0.2.6"))

	// A later raw-skills install over that plugin-only state must not panic on a
	// nil Skills map.
	require.NoError(t, InstallSkillsForAgents(ctx, &mockManifestSource{manifest: testManifest()}, []*agents.Agent{testAgent(tmp)}, InstallOptions{}))

	st, err := LoadState(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, st.Skills)
	assert.Contains(t, st.Plugins, agents.NameClaudeCode)
}
