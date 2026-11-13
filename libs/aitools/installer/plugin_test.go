package installer

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureWarnings routes warnings and errors logged through ctx into a buffer.
func captureWarnings(ctx context.Context) (context.Context, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	return log.NewContext(ctx, logger), &buf
}

const localCopyWarning = "plugin came from its local copy and may be out of date"

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
	assert.Contains(t, cmds, "claude plugin marketplace update databricks-agent-skills")
	assert.Contains(t, cmds, "claude plugin install databricks@databricks-agent-skills --scope user")
}

func TestInstallPluginForAgentSharedMarketplace(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	// Claude installs from its official, Shared marketplace (claude-plugins-official).
	// It is not reliably registered locally, so the CLI must add it from its source
	// before refreshing and installing. Use the real registry spec.
	agent := agents.ByName(agents.NameClaudeCode)

	rec, err := InstallPluginForAgent(ctx, agent, "user", "main")
	require.NoError(t, err)
	assert.Equal(t, "claude-plugins-official", rec.Marketplace)
	assert.True(t, rec.InstalledMarketplace, "an absent shared marketplace is added by us")

	cmds := stub.Commands()
	assert.Contains(t, cmds, "claude plugin marketplace add anthropics/claude-plugins-official")
	assert.Contains(t, cmds, "claude plugin marketplace update claude-plugins-official")
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

func TestInstallPluginContinuesWhenRefreshFails(t *testing.T) {
	// The plugin's own marketplace can't be refreshed (e.g. it is unreachable),
	// but the local copy still lists the plugin, so the install goes on.
	for _, tc := range []struct {
		name string
		// listed is the `plugin marketplace list` output before the install.
		listed            string
		wantInstalledByUs bool
		wantVersion       string
		wantWarning       bool
	}{
		{
			// We just added the marketplace, so its local copy is fresh.
			name:              "just added",
			wantInstalledByUs: true,
			wantVersion:       "0.2.6",
		},
		{
			// Registered before, so the local copy may be old, and the
			// marketplace is not ours to remove or record.
			name:        "already registered",
			listed:      "databricks-agent-skills\n",
			wantVersion: "",
			wantWarning: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stubAgentLookPath(t, true)
			ctx, stub := process.WithStub(t.Context())
			ctx, logs := captureWarnings(ctx)
			stub.WithCallback(func(*exec.Cmd) error { return nil })
			stub.WithStdoutFor("plugin marketplace list", tc.listed)
			stub.WithStderrFor("plugin marketplace update", "could not reach github.com").
				WithFailureFor("plugin marketplace update", errors.New("exit status 1"))

			rec, err := InstallPluginForAgent(ctx, claudeAgent(), "user", "v0.2.6")
			require.NoError(t, err)
			assert.Equal(t, tc.wantInstalledByUs, rec.InstalledMarketplace)
			assert.Equal(t, tc.wantVersion, rec.Version)
			assert.Contains(t, stub.Commands(), "claude plugin install databricks@databricks-agent-skills --scope user")
			assert.NotContains(t, stub.Commands(), "claude plugin marketplace remove databricks-agent-skills")
			if tc.wantWarning {
				assert.Contains(t, logs.String(), localCopyWarning)
				assert.Contains(t, logs.String(), "could not reach github.com")
			} else {
				assert.NotContains(t, logs.String(), localCopyWarning)
			}
		})
	}
}

func TestInstallPluginRollsBackMarketplaceWhenRefreshAndInstallFail(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	// Marketplace absent (empty list) so we add it; then refresh and install fail.
	stub.WithFailureFor("plugin marketplace update", errors.New("refresh boom"))
	stub.WithFailureFor("plugin install", errors.New("install boom"))
	ctx, logs := captureWarnings(ctx)

	_, err := InstallPluginForAgent(ctx, claudeAgent(), "user", "v0.2.6")
	var be *BlockedError
	require.ErrorAs(t, err, &be)
	assert.Equal(t, ReasonInstallFailed, be.Reason)
	assert.Contains(t, be.Detail, "install boom (refreshing the marketplace also failed: ")
	assert.Contains(t, be.Detail, "refresh boom)")
	// The refresh error is in the detail; the "local copy" advice would be wrong.
	assert.NotContains(t, logs.String(), localCopyWarning)
	// We added the marketplace, so a failed install must de-register it again.
	assert.Contains(t, stub.Commands(), "claude plugin marketplace remove databricks-agent-skills")
}

func TestUpdatePluginForAgentCodexTwoStep(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	refreshErr, err := UpdatePluginForAgent(ctx, codexAgent(), PluginRecord{Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user"})
	require.NoError(t, err)
	assert.NoError(t, refreshErr)
	cmds := stub.Commands()
	assert.Contains(t, cmds, "codex plugin marketplace upgrade databricks-agent-skills")
	assert.Contains(t, cmds, "codex plugin add databricks@databricks-agent-skills")
}

func TestUpdatePluginForAgentClaudeOneStep(t *testing.T) {
	for _, tc := range []struct {
		name string
		rec  PluginRecord
	}{
		{"recorded marketplace", PluginRecord{Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user"}},
		// A record without a marketplace falls back to the agent's current one.
		{"record without marketplace", PluginRecord{Plugin: "databricks", Scope: "user"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stubAgentLookPath(t, true)
			ctx, stub := process.WithStub(t.Context())
			stub.WithCallback(func(*exec.Cmd) error { return nil })

			refreshErr, err := UpdatePluginForAgent(ctx, claudeAgent(), tc.rec)
			require.NoError(t, err)
			assert.NoError(t, refreshErr)
			assert.Equal(t, []string{
				"claude plugin marketplace update databricks-agent-skills",
				"claude plugin update databricks@databricks-agent-skills --scope user",
			}, stub.Commands())
		})
	}
}

func TestUpdatePluginForAgentContinuesWhenRefreshFails(t *testing.T) {
	for _, tc := range []struct {
		agent   *agents.Agent
		refresh string
		update  string
	}{
		{claudeAgent(), "claude plugin marketplace update databricks-agent-skills", "claude plugin update databricks@databricks-agent-skills --scope user"},
		{codexAgent(), "codex plugin marketplace upgrade databricks-agent-skills", "codex plugin add databricks@databricks-agent-skills"},
	} {
		t.Run(tc.agent.Name, func(t *testing.T) {
			stubAgentLookPath(t, true)
			ctx, stub := process.WithStub(t.Context())
			stub.WithCallback(func(*exec.Cmd) error { return nil })
			stub.WithStderrFor("plugin marketplace", "could not reach github.com").
				WithFailureFor("plugin marketplace", errors.New("exit status 1"))

			refreshErr, err := UpdatePluginForAgent(ctx, tc.agent, PluginRecord{Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user"})
			require.NoError(t, err)
			require.Error(t, refreshErr)
			assert.Equal(t, "could not reach github.com", stderrOf(refreshErr))
			assert.Equal(t, []string{tc.refresh, tc.update}, stub.Commands())
		})
	}
}

func TestUpdatePluginForAgentReportsRefreshWhenUpdateFails(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	stub.WithFailureFor("plugin marketplace update", errors.New("refresh boom"))
	stub.WithFailureFor("plugin update", errors.New("update boom"))
	ctx, logs := captureWarnings(ctx)

	_, err := UpdatePluginForAgent(ctx, claudeAgent(), PluginRecord{Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user"})
	var be *BlockedError
	require.ErrorAs(t, err, &be)
	assert.Equal(t, ReasonInstallFailed, be.Reason)
	assert.Contains(t, be.Detail, "update boom (refreshing the marketplace also failed: ")
	assert.Contains(t, be.Detail, "refresh boom)")
	assert.NotContains(t, logs.String(), localCopyWarning)
}

func TestUninstallPluginClaudeAlreadyDisabledContinues(t *testing.T) {
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	stub.WithStderrFor("plugin disable", "Plugin databricks is already disabled in user scope").
		WithFailureFor("plugin disable", errors.New("exit status 1"))

	rec := PluginRecord{Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user"}
	require.NoError(t, UninstallPluginForAgent(ctx, claudeAgent(), rec, false))

	cmds := stub.Commands()
	assert.Contains(t, cmds, "claude plugin disable databricks@databricks-agent-skills --scope user")
	assert.Contains(t, cmds, "claude plugin uninstall databricks@databricks-agent-skills --scope user")
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

func TestUninstallSkillsWithResultTearsDownPlugin(t *testing.T) {
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

	_, err = UninstallSkillsWithResult(ctx, UninstallOptions{Scope: ScopeGlobal})
	require.NoError(t, err)

	cmds := stub.Commands()
	assert.Contains(t, cmds, "copilot plugin uninstall databricks@databricks-agent-skills")
	assert.Contains(t, cmds, "copilot plugin marketplace remove databricks-agent-skills")
	assert.Contains(t, stderr.String(), "Uninstalled the plugin from 1 agent.")

	// State file removed since nothing remains.
	state, err := LoadState(dir)
	require.NoError(t, err)
	assert.Nil(t, state)
}

func TestUninstallSkillsWithResultTargetsPluginAgents(t *testing.T) {
	setupTestHome(t)
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	ctx = cmdio.MockDiscard(ctx)

	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Release:       "v0.2.6",
		Plugins: map[string]PluginRecord{
			agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks", Scope: "user"},
			agents.NameCopilot:    {Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user"},
		},
	}))

	_, err = UninstallSkillsWithResult(ctx, UninstallOptions{Scope: ScopeGlobal, Agents: []string{agents.NameClaudeCode}})
	require.NoError(t, err)

	cmds := stub.Commands()
	assert.Contains(t, cmds, "claude plugin uninstall databricks@claude-plugins-official --scope user")
	for _, cmd := range cmds {
		assert.NotContains(t, cmd, "copilot plugin uninstall")
	}

	state, err := LoadState(dir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.NotContains(t, state.Plugins, agents.NameClaudeCode)
	assert.Contains(t, state.Plugins, agents.NameCopilot)
}

func TestUninstallNeverDeregistersSharedMarketplace(t *testing.T) {
	setupTestHome(t)
	stubAgentLookPath(t, true)
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })
	ctx, stderr := cmdio.NewTestContextWithStderr(ctx)

	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	// Claude's claude-plugins-official is a Shared marketplace; even a stale
	// InstalledMarketplace=true must never trigger a de-register.
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Plugins: map[string]PluginRecord{
			agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks", InstalledMarketplace: true},
		},
	}))

	_, err = UninstallSkillsWithResult(ctx, UninstallOptions{Scope: ScopeGlobal})
	require.NoError(t, err)

	cmds := stub.Commands()
	assert.Contains(t, cmds, "claude plugin uninstall databricks@claude-plugins-official")
	for _, c := range cmds {
		assert.NotContains(t, c, "marketplace remove")
	}
	// The message must not claim to have removed the shared marketplace.
	assert.Contains(t, stderr.String(), "removed databricks plugin")
	assert.NotContains(t, stderr.String(), "+ marketplace")
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

	_, err = UninstallSkillsWithResult(ctx, UninstallOptions{Scope: ScopeGlobal, KeepMarketplace: true})
	require.NoError(t, err)

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
	_, err = UninstallSkillsWithResult(ctx, UninstallOptions{Scope: ScopeGlobal})
	require.NoError(t, err)

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

	_, err = UninstallSkillsWithResult(ctx, UninstallOptions{Scope: ScopeGlobal})
	require.NoError(t, err)

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

func TestUpdateInstalledPluginsRecordsUnknownVersionWhenRefreshFails(t *testing.T) {
	for _, tc := range []struct {
		name        string
		recorded    string
		want        string
		wantWarning bool
	}{
		// The update ran from a local copy of unknown age.
		{"older release recorded", "0.2.6", "", true},
		// The record already holds this release, and the local copy can only
		// have moved forward since the refresh that wrote it, so there is
		// nothing to warn about.
		{"current release recorded", "0.2.7", "0.2.7", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupTestHome(t)
			stubAgentLookPath(t, true)
			ctx, stub := process.WithStub(t.Context())
			ctx, logs := captureWarnings(ctx)
			stub.WithCallback(func(*exec.Cmd) error { return nil })
			stub.WithStderrFor("plugin marketplace update", "could not reach github.com").
				WithFailureFor("plugin marketplace update", errors.New("exit status 1"))

			dir, err := GlobalSkillsDir(ctx)
			require.NoError(t, err)
			require.NoError(t, SaveState(dir, &InstallState{
				SchemaVersion: schemaVersionV2,
				Release:       "v" + tc.recorded,
				Plugins: map[string]PluginRecord{
					agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks", Scope: "user", Version: tc.recorded},
				},
			}))

			updated, err := UpdateInstalledPlugins(ctx, ScopeGlobal, "v0.2.7")
			require.NoError(t, err)
			require.Len(t, updated, 1)
			require.NoError(t, updated[0].Err)
			assert.Equal(t, tc.want, updated[0].Version)
			assert.Contains(t, stub.Commands(), "claude plugin update databricks@claude-plugins-official --scope user")

			state, err := LoadState(dir)
			require.NoError(t, err)
			assert.Equal(t, tc.want, state.Plugins[agents.NameClaudeCode].Version)
			if tc.wantWarning {
				assert.Contains(t, logs.String(), localCopyWarning)
				assert.Contains(t, logs.String(), "rerun `databricks aitools update` later")
				assert.Contains(t, logs.String(), "could not reach github.com")
			} else {
				assert.NotContains(t, logs.String(), localCopyWarning)
			}
		})
	}
}

func TestUpdateInstalledPluginsUsesRecordedClaudeScope(t *testing.T) {
	for _, scope := range []string{"project", "user", "local"} {
		t.Run(scope, func(t *testing.T) {
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
					agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks", Scope: scope, Version: "0.2.6"},
				},
			}))

			updated, err := UpdateInstalledPlugins(ctx, ScopeGlobal, "v0.2.7")
			require.NoError(t, err)
			require.Len(t, updated, 1)

			cmds := stub.Commands()
			assert.Contains(t, cmds, "claude plugin marketplace update claude-plugins-official")
			assert.Contains(t, cmds, "claude plugin update databricks@claude-plugins-official --scope "+scope)
		})
	}
}

// A plugin whose CLI is missing is skipped with a warning (no error), and
// reported so telemetry can tell a run that did nothing from a success.
func TestPluginOperationsReportFailures(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		ctx, dir := saveCopilotPlugin(t)
		results, err := UpdateInstalledPlugins(ctx, ScopeGlobal, "v0.2.7")
		require.NoError(t, err)
		require.Len(t, results, 1)
		blocked, ok := errors.AsType[*BlockedError](results[0].Err)
		require.True(t, ok, "got %v", results[0].Err)
		assert.Equal(t, ReasonCLINotOnPath, blocked.Reason)

		state, err := LoadState(dir)
		require.NoError(t, err)
		assert.Equal(t, "0.2.6", state.Plugins[agents.NameCopilot].Version)
	})

	t.Run("uninstall", func(t *testing.T) {
		ctx, dir := saveCopilotPlugin(t)
		result, err := UninstallSkillsWithResult(ctx, UninstallOptions{Scope: ScopeGlobal})
		require.NoError(t, err)
		assert.Equal(t, 0, result.PluginsRemoved)
		assert.Equal(t, 0, result.SkillsChanged)
		require.Len(t, result.PluginFailures, 1)
		blocked, ok := errors.AsType[*BlockedError](result.PluginFailures[0])
		require.True(t, ok, "got %v", result.PluginFailures[0])
		assert.Equal(t, ReasonCLINotOnPath, blocked.Reason)

		state, err := LoadState(dir)
		require.NoError(t, err)
		assert.Contains(t, state.Plugins, agents.NameCopilot)
	})
}

// A plugin record for an agent this CLI does not know (e.g. written by a newer
// CLI) is kept and reported as a failure, so a run that acts on nothing else
// is not a success.
func TestPluginOperationsReportUnknownAgent(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		ctx, dir := saveUnknownAgentPlugin(t)
		results, err := UpdateInstalledPlugins(ctx, ScopeGlobal, "v0.2.7")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "future-agent", results[0].Agent)
		assert.EqualError(t, results[0].Err, `unknown agent "future-agent" in state`)
		assert.ErrorIs(t, results[0].Err, ErrUnknownAgent)

		state, err := LoadState(dir)
		require.NoError(t, err)
		assert.Equal(t, "0.2.6", state.Plugins["future-agent"].Version)
	})

	t.Run("uninstall", func(t *testing.T) {
		ctx, dir := saveUnknownAgentPlugin(t)
		result, err := UninstallSkillsWithResult(ctx, UninstallOptions{Scope: ScopeGlobal})
		require.NoError(t, err)
		assert.Equal(t, 0, result.PluginsRemoved)
		require.Len(t, result.PluginFailures, 1)
		assert.EqualError(t, result.PluginFailures[0], `unknown agent "future-agent" in state`)
		assert.ErrorIs(t, result.PluginFailures[0], ErrUnknownAgent)

		state, err := LoadState(dir)
		require.NoError(t, err)
		assert.Contains(t, state.Plugins, "future-agent")
	})
}

// saveUnknownAgentPlugin records a plugin for an agent this CLI does not know
// in the global state, and returns the context and the state dir.
func saveUnknownAgentPlugin(t *testing.T) (context.Context, string) {
	t.Helper()
	setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Release:       "v0.2.6",
		Plugins: map[string]PluginRecord{
			"future-agent": {Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user", Version: "0.2.6"},
		},
	}))
	return ctx, dir
}

// saveCopilotPlugin records a Copilot plugin in the global state while the
// Copilot CLI is not on PATH, and returns the context and the state dir.
func saveCopilotPlugin(t *testing.T) (context.Context, string) {
	t.Helper()
	setupTestHome(t)
	stubAgentLookPath(t, false)
	ctx := cmdio.MockDiscard(t.Context())
	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Release:       "v0.2.6",
		Plugins: map[string]PluginRecord{
			agents.NameCopilot: {Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: "user", Version: "0.2.6"},
		},
	}))
	return ctx, dir
}

func TestUninstallSkillsWithResultUsesRecordedClaudeScope(t *testing.T) {
	for _, scope := range []string{"project", "user", "local"} {
		t.Run(scope, func(t *testing.T) {
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
					agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks", Scope: scope},
				},
			}))

			_, err = UninstallSkillsWithResult(ctx, UninstallOptions{Scope: ScopeGlobal})
			require.NoError(t, err)

			cmds := stub.Commands()
			assert.Contains(t, cmds, "claude plugin disable databricks@claude-plugins-official --scope "+scope)
			assert.Contains(t, cmds, "claude plugin uninstall databricks@claude-plugins-official --scope "+scope)
		})
	}
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

func TestRecordPluginInstallsUnresolvedRefKeepsRelease(t *testing.T) {
	setupTestHome(t)
	ctx := cmdio.MockDiscard(t.Context())
	dir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)

	require.NoError(t, RecordPluginInstalls(ctx, ScopeGlobal, map[string]PluginRecord{
		agents.NameClaudeCode: {Marketplace: "databricks-agent-skills", Plugin: "databricks", Version: "0.2.6"},
	}, "v0.2.6"))
	require.NoError(t, RecordPluginInstalls(ctx, ScopeGlobal, map[string]PluginRecord{
		agents.NameCodex: {Marketplace: "databricks-agent-skills", Plugin: "databricks"},
	}, ""))

	st, err := LoadState(dir)
	require.NoError(t, err)
	assert.Equal(t, "v0.2.6", st.Release)
	assert.Contains(t, st.Plugins, agents.NameCodex)
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

func TestRunAgentCmdTimeoutKeepsPartialStderr(t *testing.T) {
	// What the CLI wrote before it was killed often says why it hung.
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(cmd *exec.Cmd) error {
		_, _ = cmd.Stderr.Write([]byte("Waiting for login...\n"))
		time.Sleep(50 * time.Millisecond)
		return errors.New("signal: killed")
	})

	_, err := runAgentCmd(ctx, time.Millisecond, []string{"claude", "plugin", "--help"})
	require.Error(t, err)
	blocked := newBlockedError(claudeAgent(), ReasonCLINotOnPath, err)
	assert.True(t, blocked.TimedOut)
	assert.Equal(t, "timed out after 1ms: Waiting for login...", blocked.Detail)
}

func TestRunAgentCmdReportsTimeout(t *testing.T) {
	// A slow agent CLI used to be reported as "not on PATH" with whatever the
	// killed process left on stderr.
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error {
		time.Sleep(50 * time.Millisecond)
		return errors.New("signal: killed")
	})

	_, err := runAgentCmd(ctx, time.Millisecond, []string{"claude", "plugin", "--help"})
	require.Error(t, err)
	blocked := newBlockedError(claudeAgent(), ReasonCLINotOnPath, err)
	assert.True(t, blocked.TimedOut)
	assert.Equal(t, ReasonCLINotOnPath, blocked.Reason)
	assert.Equal(t, "timed out after 1ms", blocked.Detail)
}

func TestManualPluginInstallCommands(t *testing.T) {
	assert.Equal(t, []string{
		"claude plugin marketplace add databricks/databricks-agent-skills",
		"claude plugin marketplace update databricks-agent-skills",
		"claude plugin install databricks@databricks-agent-skills --scope user",
	}, ManualPluginInstallCommands(claudeAgent(), "user"))
	assert.Equal(t, []string{
		"codex plugin marketplace add databricks/databricks-agent-skills",
		"codex plugin add databricks@databricks-agent-skills",
	}, ManualPluginInstallCommands(codexAgent(), ""))
	assert.Nil(t, ManualPluginInstallCommands(noPluginAgent(), ""))
}

// copilotPluginHelp is the commander-style shape of `copilot plugin --help`.
const copilotPluginHelp = "Usage: copilot plugin [options] [command]\n\nManage plugins\n\nCommands:\n  install <plugin>     Install a plugin\n  uninstall <plugin>   Uninstall a plugin\n  marketplace          Manage marketplaces\n"

// codexPluginHelp is the clap-style shape of `codex plugin --help`, which
// lists add rather than install.
const codexPluginHelp = "Manage plugins\n\nUsage: codex plugin [OPTIONS] <COMMAND>\n\nCommands:\n  add          Install a plugin\n  marketplace  Manage plugin marketplaces\n  help         Print this message or the help of the given subcommand(s)\n\nOptions:\n  -h, --help  Print help\n"

func TestProbeWindowsShimPrintedHelp(t *testing.T) {
	copilot := pluginAgent(agents.NameCopilot, "GitHub Copilot", "copilot")
	tests := []struct {
		name   string
		agent  *agents.Agent
		bin    string
		stdout string
		stderr string
		shim   bool
	}{
		{name: "windows .bat shim printed help and exited 1", agent: copilot, bin: `C:\npm\copilot.bat`, stdout: copilotPluginHelp, shim: true},
		{name: "windows .cmd shim printed help and exited 1", agent: copilot, bin: `C:\npm\copilot.cmd`, stdout: copilotPluginHelp, shim: true},
		{name: "windows shim under a path with a space", agent: copilot, bin: `C:\Users\Jane Doe\AppData\Roaming\npm\copilot.cmd`, stdout: copilotPluginHelp, shim: true},
		{name: "windows shim that printed an error on stderr", agent: copilot, bin: `C:\npm\copilot.cmd`, stdout: copilotPluginHelp, stderr: "  node: bad option\n", shim: true},
		{name: "help on stderr is not a shim that printed help", agent: copilot, bin: `C:\npm\copilot.cmd`, stderr: copilotPluginHelp},
		{name: "windows codex .cmd shim printed clap help and exited 1", agent: codexAgent(), bin: `C:\npm\codex.cmd`, stdout: codexPluginHelp, shim: true},
		{name: "windows shim without a plugin command", agent: codexAgent(), bin: `C:\npm\codex.cmd`, stdout: "error: unknown command 'plugin'\n"},
		{name: "unix binary is not a shim", agent: copilot, bin: "/usr/bin/copilot", stdout: copilotPluginHelp},
		{name: "windows .exe is not a shim", agent: copilot, bin: `C:\bin\copilot.exe`, stdout: copilotPluginHelp},
		{name: "error that mentions both words", agent: copilot, bin: `C:\npm\copilot.bat`, stdout: "Error: cannot install plugins: marketplace disabled by policy\n"},
		{name: "usage line without a command list", agent: copilot, bin: `C:\npm\copilot.bat`, stdout: "Usage: copilot plugin\nerror: install and marketplace are not available\n"},
		{name: "unindented words after blank lines", agent: copilot, bin: `C:\npm\copilot.bat`, stdout: "Usage: copilot plugin [options]\n\nmarketplace: disabled by policy\n\ninstall is not permitted\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origLook := lookPath
			t.Cleanup(func() { lookPath = origLook })
			lookPath = func(string) (string, error) { return tc.bin, nil }

			ctx, stub := process.WithStub(t.Context())
			stub.WithCallback(func(*exec.Cmd) error { return nil })
			stub.WithStdoutFor("plugin --help", tc.stdout)
			stub.WithStderrFor("plugin --help", tc.stderr)
			stub.WithFailureFor("plugin --help", errors.New("exit status 1"))

			_, err := InstallPluginForAgent(ctx, tc.agent, "", "v0.2.6")
			var be *BlockedError
			require.ErrorAs(t, err, &be)
			if tc.shim {
				assert.Equal(t, ReasonShimExitedWithError, be.Reason)
				assert.False(t, be.TimedOut)
				wantDetail := "printed its plugin help but exited with an error (exit status 1)"
				if tc.stderr != "" {
					wantDetail += ": " + strings.TrimSpace(tc.stderr)
				}
				wantDetail += "; to see the error and exit code, run `& \"" + tc.bin + "\" plugin --help; $LASTEXITCODE` in PowerShell or `\"" + tc.bin + "\" plugin --help` and then `echo %ERRORLEVEL%` in Command Prompt, and re-run this command once it exits with 0"
				assert.Equal(t, wantDetail, be.Detail)
			} else {
				assert.Equal(t, ReasonCLINotOnPath, be.Reason)
			}
			assert.Len(t, stub.Commands(), 1, "nothing runs after a failed probe")
		})
	}
}

func TestWindowsShimPrintedHelpRejectsTimeout(t *testing.T) {
	// A shim that printed the help and then hung until its deadline reports the
	// timeout, even though the same output without the timeout is a shim error.
	perr := &process.ProcessError{Command: "copilot.bat plugin --help", Err: errors.New("exit status 1"), Stdout: copilotPluginHelp}
	bin := `C:\npm\copilot.bat`
	got, ok := windowsShimPrintedHelp(bin, perr)
	assert.True(t, ok)
	assert.Same(t, perr, got)
	_, ok = windowsShimPrintedHelp(bin, &agentCmdTimeoutError{timeout: time.Second, err: perr})
	assert.False(t, ok)
}
