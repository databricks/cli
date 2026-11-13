package aitools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func drainReader(r *bufio.Reader) {
	buf := make([]byte, 4096)
	for {
		if _, err := r.Read(buf); err != nil {
			return
		}
	}
}

// --- Test helpers ---

// newTestInstallCmd builds the install command with the pieces the root command
// supplies in production: the persistent --output flag (so cobra can parse
// `--output` on this detached command; install reads it via installOutputIsJSON)
// and silenced cobra error/usage output (root sets SilenceErrors and prints
// errors itself). Without the latter, a detached command prints cobra's own
// "Error:"/usage to the captured buffers.
func newTestInstallCmd() *cobra.Command {
	cmd := NewInstallCmd()
	output := flags.OutputText
	cmd.PersistentFlags().VarP(&output, "output", "o", "output type: text or json")
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

type installCall struct {
	agents []string
	opts   installer.InstallOptions
}

func setupInstallMock(t *testing.T) *[]installCall {
	t.Helper()
	orig := installSkillsForAgentsFn
	t.Cleanup(func() { installSkillsForAgentsFn = orig })

	var calls []installCall
	installSkillsForAgentsFn = func(_ context.Context, _ installer.ManifestSource, targetAgents []*agents.Agent, opts installer.InstallOptions) error {
		names := make([]string, len(targetAgents))
		for i, a := range targetAgents {
			names[i] = a.Name
		}
		calls = append(calls, installCall{agents: names, opts: opts})
		return nil
	}
	return &calls
}

type pluginCall struct {
	agent string
	scope string
}

func setupPluginMock(t *testing.T) *[]pluginCall {
	t.Helper()
	origInstall := installPluginForAgentFn
	origRecord := recordPluginInstallsFn
	t.Cleanup(func() {
		installPluginForAgentFn = origInstall
		recordPluginInstallsFn = origRecord
	})

	var calls []pluginCall
	installPluginForAgentFn = func(_ context.Context, a *agents.Agent, scope, ref string) (installer.PluginRecord, error) {
		calls = append(calls, pluginCall{agent: a.Name, scope: scope})
		return installer.PluginRecord{Marketplace: "databricks-agent-skills", Plugin: "databricks", Scope: scope, Version: "0.2.6"}, nil
	}
	recordPluginInstallsFn = func(context.Context, string, map[string]installer.PluginRecord, string) error { return nil }
	return &calls
}

// stubPluginAlreadyInstalled pins whether an agent counts as already having the
// plugin; the real check reads the agent's manifest under the real HOME.
func stubPluginAlreadyInstalled(t *testing.T, installed bool) {
	t.Helper()
	orig := pluginAlreadyInstalledFn
	t.Cleanup(func() { pluginAlreadyInstalledFn = orig })
	pluginAlreadyInstalledFn = func(context.Context, *agents.Agent, string) bool { return installed }
}

// stubLegacyCleanup skips the raw-skill cleanup that follows a plugin install;
// the test agents have no config dir for it to inspect.
func stubLegacyCleanup(t *testing.T) {
	t.Helper()
	orig := cleanupLegacyFn
	t.Cleanup(func() { cleanupLegacyFn = orig })
	cleanupLegacyFn = func(context.Context, *agents.Agent, string) error { return nil }
}

func setupScopeMock(t *testing.T, scope string) *bool {
	t.Helper()
	orig := promptScopeSelection
	t.Cleanup(func() { promptScopeSelection = orig })

	called := false
	promptScopeSelection = func(_ context.Context) (string, error) {
		called = true
		return scope, nil
	}
	return &called
}

// setupTestAgents creates config dirs for Claude and Cursor under a temp HOME.
// XDG_CONFIG_HOME must be neutralized too: OpenCode resolves its config dir
// from it, so a developer machine with a real ~/.config/opencode (or an XDG
// override pointing at one) would otherwise detect OpenCode as a third agent.
func setupTestAgents(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))
	return tmp
}

// fakeBinsOnPath replaces PATH with a temp dir holding stub executables for the
// named agent binaries, so HasBinary detects exactly those agents.
func fakeBinsOnPath(t *testing.T, binaries ...string) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range binaries {
		p := filepath.Join(dir, name)
		if runtime.GOOS == "windows" {
			p += ".exe"
		}
		require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755))
	}
	t.Setenv("PATH", dir)
}

// writeClaudeManifest writes Claude Code's installed_plugins.json under home
// with one databricks plugin record of the given scope.
func writeClaudeManifest(t *testing.T, home, scope, projectPath string) {
	t.Helper()
	manifest, err := json.Marshal(map[string]any{"plugins": map[string]any{
		"databricks@claude-plugins-official": []map[string]string{{"scope": scope, "projectPath": projectPath, "version": "0.2.9"}},
	}})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude", "plugins", "installed_plugins.json"), manifest, 0o600))
}

func testPluginAgent(name, display, binary string) *agents.Agent {
	spec := &agents.PluginSpec{Marketplace: "databricks-agent-skills", ID: "databricks", Source: "databricks/databricks-agent-skills"}
	a := &agents.Agent{Name: name, DisplayName: display, Binary: binary, Plugin: spec}
	if name == agents.NameClaudeCode {
		a.SupportsProjectScope = true
	}
	return a
}

// --- buildPlan / mapAgentScope / executePlan unit tests (no detection) ---

func TestBuildPlanDeliveries(t *testing.T) {
	fakeBinsOnPath(t, "claude")
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	// Cursor is a skills-only agent (no headless plugin install).
	cursor := &agents.Agent{Name: agents.NameCursor, DisplayName: "Cursor", Binary: "cursor-agent", SupportsProjectScope: true}
	opencode := &agents.Agent{Name: agents.NameOpenCode, DisplayName: "OpenCode", Binary: "opencode"}
	targets := []*agents.Agent{claude, cursor, opencode}

	plan := buildPlan(t.Context(), targets, installer.ScopeGlobal, false, false)
	assert.Equal(t, deliveryPlugin, plan[0].delivery)
	assert.Equal(t, agentScopeUser, plan[0].scope)
	assert.Equal(t, deliverySkills, plan[1].delivery) // Cursor -> skills
	assert.Equal(t, deliverySkills, plan[2].delivery)

	// --skills-only forces skills for every agent.
	skillsPlan := buildPlan(t.Context(), targets, installer.ScopeGlobal, true, false)
	for _, it := range skillsPlan {
		assert.Equal(t, deliverySkills, it.delivery)
	}
}

func TestAgentChoicesOnlyOffersActionableAgents(t *testing.T) {
	setupTestAgents(t)
	fakeBinsOnPath(t, "claude")
	ctx := cmdio.MockDiscard(t.Context())

	// Project scope: agents that support project-scoped skills are offered (Claude
	// via plugin; Pi/Gemini/Goose via skills). User-only plugin agents and
	// global-only files agents are not.
	choices := agentChoices(ctx, installer.ScopeProject, false)
	var names []string
	for _, c := range choices {
		names = append(names, c.agent.Name)
	}
	assert.Contains(t, names, agents.NameClaudeCode)
	assert.Contains(t, names, agents.NamePi)
	assert.Contains(t, names, agents.NameGemini)
	assert.Contains(t, names, agents.NameGoose)
	assert.Contains(t, names, agents.NameKiro)
	assert.NotContains(t, names, agents.NameCursor)
	assert.NotContains(t, names, agents.NameCodex)
	assert.NotContains(t, names, agents.NameOpenCode)
	assert.NotContains(t, names, agents.NameCopilot)
	assert.NotContains(t, names, agents.NameAntigravity)
}

// choiceLabels returns the picker label of each offered agent, by agent name.
func choiceLabels(ctx context.Context, scope string, skillsOnly bool) map[string]string {
	m := map[string]string{}
	for _, c := range agentChoices(ctx, scope, skillsOnly) {
		m[c.agent.Name] = c.label
	}
	return m
}

func TestAgentChoicesSkillsOnlyLabelsPluginAgentsSkills(t *testing.T) {
	setupTestAgents(t)
	fakeBinsOnPath(t, "claude")
	ctx := cmdio.MockDiscard(t.Context())

	assert.Equal(t, "Claude Code  plugin", choiceLabels(ctx, installer.ScopeGlobal, false)[agents.NameClaudeCode])
	// --skills-only installs raw skills for every agent, plugin agents included.
	assert.Equal(t, "Claude Code  skills", choiceLabels(ctx, installer.ScopeGlobal, true)[agents.NameClaudeCode])
	assert.Equal(t, "Claude Code  skills", choiceLabels(ctx, installer.ScopeProject, true)[agents.NameClaudeCode])
}

func TestAgentChoicesSkillsOnlyLabelsSkippedPluginAgentSkills(t *testing.T) {
	setupTestAgents(t)
	fakeBinsOnPath(t, "codex")
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())

	// Codex has no project-scoped skills, so it is skipped and only listed.
	agentChoices(ctx, installer.ScopeProject, true)
	assert.Regexp(t, `Codex CLI +skills · does not support project-scoped skills`, stderr.String())
	assert.NotContains(t, stderr.String(), "plugin")
}

func TestAgentChoicesSkillsOnlyPreselectsByConfigDir(t *testing.T) {
	setupTestAgents(t) // ~/.claude and ~/.cursor
	fakeBinsOnPath(t)  // no claude CLI on PATH
	ctx := cmdio.MockDiscard(t.Context())

	// Plugin delivery needs the CLI, so it starts unchecked.
	assert.False(t, choicePreselected(t, agentChoices(ctx, installer.ScopeGlobal, false), agents.NameClaudeCode))
	// Skills delivery only needs the config dir, as in a non-interactive run.
	assert.True(t, choicePreselected(t, agentChoices(ctx, installer.ScopeGlobal, true), agents.NameClaudeCode))
}

// choicePreselected reports whether the named agent's picker row starts checked.
func choicePreselected(t *testing.T, choices []agentChoice, name string) bool {
	t.Helper()
	for _, c := range choices {
		if c.agent.Name == name {
			return c.preselect
		}
	}
	t.Fatalf("%s not offered", name)
	return false
}

func TestBuildPlanProjectScopeSkipsUserOnlyAgent(t *testing.T) {
	fakeBinsOnPath(t, "claude", "codex")
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	codex := testPluginAgent(agents.NameCodex, "Codex CLI", "codex")

	plan := buildPlan(t.Context(), []*agents.Agent{claude, codex}, installer.ScopeProject, false, false)
	assert.Equal(t, deliveryPlugin, plan[0].delivery)
	assert.Equal(t, agentScopeProject, plan[0].scope)
	assert.Equal(t, deliverySkip, plan[1].delivery)
	assert.Contains(t, plan[1].reason, "user-only")
}

func TestBuildPlanProjectScopeSkipsFilesOnlyAgent(t *testing.T) {
	// A files-only agent that does not support project scope is skipped up front,
	// so the picker never offers an option that fails at install time.
	opencode := &agents.Agent{Name: agents.NameOpenCode, DisplayName: "OpenCode", Binary: "opencode"}
	projectSkills := &agents.Agent{Name: "proj-skills", DisplayName: "Proj", Binary: "proj", SupportsProjectScope: true}

	plan := buildPlan(t.Context(), []*agents.Agent{opencode, projectSkills}, installer.ScopeProject, false, false)
	assert.Equal(t, deliverySkip, plan[0].delivery)
	assert.Contains(t, plan[0].reason, "project-scoped skills")
	assert.Equal(t, deliverySkills, plan[1].delivery)

	// --skills-only does not rescue a project-incompatible agent.
	skillsOnly := buildPlan(t.Context(), []*agents.Agent{opencode}, installer.ScopeProject, true, false)
	assert.Equal(t, deliverySkip, skillsOnly[0].delivery)

	// Under global scope the same agent gets skills.
	globalPlan := buildPlan(t.Context(), []*agents.Agent{opencode}, installer.ScopeGlobal, false, false)
	assert.Equal(t, deliverySkills, globalPlan[0].delivery)
}

func TestMapAgentScope(t *testing.T) {
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	codex := testPluginAgent(agents.NameCodex, "Codex CLI", "codex")

	scope, ok, _ := mapAgentScope(claude, installer.ScopeGlobal)
	assert.True(t, ok)
	assert.Equal(t, agentScopeUser, scope)

	scope, ok, _ = mapAgentScope(claude, installer.ScopeProject)
	assert.True(t, ok)
	assert.Equal(t, agentScopeProject, scope)

	_, ok, reason := mapAgentScope(codex, installer.ScopeProject)
	assert.False(t, ok)
	assert.Contains(t, reason, "user-only")
}

func TestExecutePlanSkipBlockedPluginExit0(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	fakeBinsOnPath(t, "claude", "codex")
	orig := installPluginForAgentFn
	origRec := recordPluginInstallsFn
	t.Cleanup(func() { installPluginForAgentFn = orig; recordPluginInstallsFn = origRec })
	installPluginForAgentFn = func(_ context.Context, a *agents.Agent, _, _ string) (installer.PluginRecord, error) {
		if a.Name == agents.NameClaudeCode {
			return installer.PluginRecord{}, &installer.BlockedError{Agent: a.Name, Reason: installer.ReasonInstallFailed}
		}
		return installer.PluginRecord{Version: "0.2.6"}, nil
	}
	recordPluginInstallsFn = func(context.Context, string, map[string]installer.PluginRecord, string) error { return nil }
	origCleanup := cleanupLegacyFn
	t.Cleanup(func() { cleanupLegacyFn = origCleanup })
	cleanupLegacyFn = func(context.Context, *agents.Agent, string) error { return nil }

	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	codex := testPluginAgent(agents.NameCodex, "Codex CLI", "codex")
	ctx := cmdio.MockDiscard(t.Context())

	// Non-explicit blocked install is a warning, not an error, as long as another
	// agent was set up; the agent's failure is still recorded in its outcome.
	plan := buildPlan(t.Context(), []*agents.Agent{claude, codex}, installer.ScopeGlobal, false, false)
	outcomes, err := executePlan(ctx, nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, installInvocation{scope: installer.ScopeGlobal}, false)
	require.NoError(t, err)
	require.Len(t, outcomes, 2)
	assert.Equal(t, outcomeFailed, outcomes[0].status)
	assert.Equal(t, protos.AitoolsErrorCategoryPluginInstallFailed, outcomes[0].errorCategory)
	assert.Equal(t, outcomeInstalled, outcomes[1].status)

	// Explicit (--agents) blocked install is an error, even when another named
	// agent was set up.
	planExplicit := buildPlan(t.Context(), []*agents.Agent{claude, codex}, installer.ScopeGlobal, false, true)
	_, err = executePlan(ctx, nil, planExplicit, installer.InstallOptions{Scope: installer.ScopeGlobal}, installInvocation{scope: installer.ScopeGlobal}, false)
	require.Error(t, err)
	assert.NotErrorIs(t, err, errNoAgentSetUp)
}

func TestExecutePlanNothingInstalledIsAnError(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	fakeBinsOnPath(t, "claude")
	orig := installPluginForAgentFn
	t.Cleanup(func() { installPluginForAgentFn = orig })
	installPluginForAgentFn = func(_ context.Context, a *agents.Agent, _, _ string) (installer.PluginRecord, error) {
		return installer.PluginRecord{}, &installer.BlockedError{Agent: a.Name, Reason: installer.ReasonInstallFailed, Detail: "fatal: could not read from remote repository"}
	}

	// Nothing named via --agents and the only agent's plugin install fails. This
	// used to exit 0 with only a warning, so an agent caller believed it worked.
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	plan := buildPlan(t.Context(), []*agents.Agent{claude}, installer.ScopeGlobal, false, false)
	outcomes, err := executePlan(ctx, nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, installInvocation{scope: installer.ScopeGlobal}, false)

	require.ErrorIs(t, err, errNoAgentSetUp)
	// The failure is reported per agent, so it must not also be a top-level one.
	assert.NoError(t, topLevelFailure(err))
	require.Len(t, outcomes, 1)
	assert.Equal(t, "Run 'claude plugin marketplace add databricks/databricks-agent-skills', then 'claude plugin marketplace update databricks-agent-skills', then 'claude plugin install databricks@databricks-agent-skills --scope user' to see the full error. Once they succeed, re-run 'databricks aitools install --scope=global --agents=claude-code' so the CLI records the plugin. Or install skill files instead: databricks aitools install --scope=global --agents=claude-code --skills-only", outcomes[0].recovery)
	assert.Contains(t, stderr.String(), "Skipped Claude Code: 'claude plugin' failed: fatal: could not read from remote repository")
	assert.Contains(t, stderr.String(), "  Run 'claude plugin marketplace add")
}

func TestExecutePlanPluginWithUnresolvedSkillsRelease(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("DATABRICKS_SKILLS_REF", "")
	t.Setenv("DATABRICKS_FORCE_EMBEDDED_COMPAT", "false")
	t.Setenv("DATABRICKS_CACHE_ENABLED", "true")
	t.Setenv("DATABRICKS_CACHE_DIR", tmp)
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "compat-manifest.json"), []byte(`{"0.0.0":{"appkit":"0.84.0","skills":"latest"}}`), 0o600))
	lookups := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		lookups++
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATABRICKS_CLI_GITHUB_API_URL", srv.URL)

	installRef, recordRef := "unset", "unset"
	orig := installPluginForAgentFn
	origRec := recordPluginInstallsFn
	origCleanup := cleanupLegacyFn
	t.Cleanup(func() {
		installPluginForAgentFn = orig
		recordPluginInstallsFn = origRec
		cleanupLegacyFn = origCleanup
	})
	cleanupLegacyFn = func(context.Context, *agents.Agent, string) error { return nil }
	installPluginForAgentFn = func(_ context.Context, _ *agents.Agent, scope, ref string) (installer.PluginRecord, error) {
		installRef = ref
		return installer.PluginRecord{Plugin: "databricks", Scope: scope, Version: installer.DisplaySkillsVersion(ref)}, nil
	}
	recordPluginInstallsFn = func(_ context.Context, _ string, _ map[string]installer.PluginRecord, ref string) error {
		recordRef = ref
		return nil
	}

	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	plan := buildPlan(t.Context(), []*agents.Agent{claude}, installer.ScopeGlobal, false, true)
	outcomes, err := executePlan(cmdio.MockDiscard(t.Context()), nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, installInvocation{scope: installer.ScopeGlobal}, false)
	require.NoError(t, err)
	require.Len(t, outcomes, 1)
	assert.Equal(t, outcomeInstalled, outcomes[0].status)
	assert.Equal(t, 1, lookups)
	// The marketplace picks the plugin version, so a failed lookup records it as unknown.
	assert.Empty(t, installRef)
	assert.Empty(t, recordRef)
}

func TestExecutePlanNoSkillsToInstall(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	fakeBinsOnPath(t, "claude")
	plugins := setupPluginMock(t)
	origSkills := installSkillsForAgentsFn
	origCleanup := cleanupLegacyFn
	t.Cleanup(func() { installSkillsForAgentsFn = origSkills; cleanupLegacyFn = origCleanup })
	cleanupLegacyFn = func(context.Context, *agents.Agent, string) error { return nil }
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		return installer.ErrNoSkillsResolved
	}
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	cursor := &agents.Agent{Name: agents.NameCursor, DisplayName: "Cursor"}
	inv := installInvocation{scope: installer.ScopeGlobal}

	// Only a skills agent: the run sets up nothing and fails, like an empty
	// --path dump.
	plan := buildPlan(t.Context(), []*agents.Agent{cursor}, installer.ScopeGlobal, true, false)
	_, err := executePlan(cmdio.MockDiscard(t.Context()), nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, inv, false)
	require.ErrorIs(t, err, installer.ErrNoSkillsResolved)
	assert.Error(t, topLevelFailure(err))

	// A plugin was installed too: the run succeeds and Cursor is skipped with
	// the step.
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	plan = buildPlan(t.Context(), []*agents.Agent{cursor, claude}, installer.ScopeGlobal, false, false)
	outcomes, err := executePlan(ctx, nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, inv, false)
	require.NoError(t, err)
	require.Len(t, *plugins, 1)
	require.Len(t, outcomes, 2)
	assert.Equal(t, outcomeInstalled, outcomes[0].status)
	assert.Equal(t, agents.NameCursor, outcomes[1].agent.Name)
	assert.Equal(t, outcomeSkipped, outcomes[1].status)
	// Same category the run's error gets when no plugin was installed.
	assert.Equal(t, classifyInstallError(installer.ErrNoSkillsResolved), outcomes[1].errorCategory)
	assert.NotEmpty(t, outcomes[1].errorCategory)
	assert.Contains(t, stderr.String(), "Skipped Cursor: no skills to install")
	assert.Contains(t, stderr.String(), "  No skill in this release can be installed by this CLI.")

	// A kept plugin counts like an installed one: the run succeeds too.
	*plugins = nil
	stubPluginAlreadyInstalled(t, true)
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude"), 0o755))
	desktopClaude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude-desktop")
	desktopClaude.ConfigDir = func(context.Context) (string, error) { return filepath.Join(home, ".claude"), nil }
	plan = buildPlan(t.Context(), []*agents.Agent{cursor, desktopClaude}, installer.ScopeGlobal, false, false)
	outcomes, err = executePlan(cmdio.MockDiscard(t.Context()), nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, inv, false)
	require.NoError(t, err)
	assert.Empty(t, *plugins)
	require.Len(t, outcomes, 2)
	assert.Equal(t, outcomeKept, outcomes[0].status)
	assert.Equal(t, outcomeSkipped, outcomes[1].status)

	// Named via --agents, the skipped skills agent is an error even so.
	plan = buildPlan(t.Context(), []*agents.Agent{cursor, claude}, installer.ScopeGlobal, false, true)
	_, err = executePlan(cmdio.MockDiscard(t.Context()), nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, inv, false)
	require.ErrorIs(t, err, installer.ErrNoSkillsResolved)
	assert.NoError(t, topLevelFailure(err))
}

func TestExecutePlanNoSkillsToInstallWithPluginFailures(t *testing.T) {
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	fakeBinsOnPath(t, "claude")
	setupPluginMock(t)
	origSkills := installSkillsForAgentsFn
	t.Cleanup(func() { installSkillsForAgentsFn = origSkills })
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		return installer.ErrNoSkillsResolved
	}
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	cursor := &agents.Agent{Name: agents.NameCursor, DisplayName: "Cursor"}
	inv := installInvocation{scope: installer.ScopeGlobal}
	plan := buildPlan(t.Context(), []*agents.Agent{cursor, claude}, installer.ScopeGlobal, false, false)

	// The plugin install failed: the run failed to set anything up, which is a
	// FAILURE with the plugin's category, not "nothing to do".
	installPluginForAgentFn = func(_ context.Context, a *agents.Agent, _, _ string) (installer.PluginRecord, error) {
		return installer.PluginRecord{}, &installer.BlockedError{Agent: a.Name, Reason: installer.ReasonInstallFailed}
	}
	outcomes, err := executePlan(cmdio.MockDiscard(t.Context()), nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, inv, false)
	require.ErrorIs(t, err, errNoAgentSetUp)
	assert.NotErrorIs(t, err, installer.ErrNoSkillsResolved)
	assert.Equal(t, protos.AitoolsErrorCategoryPluginInstallFailed, firstFailureCategory(outcomes))

	// The plugin installed but its record could not be saved: the record
	// failure is the error, without the no-skills sentinel.
	installPluginForAgentFn = func(_ context.Context, _ *agents.Agent, scope, _ string) (installer.PluginRecord, error) {
		return installer.PluginRecord{Plugin: "databricks", Scope: scope, Version: "0.2.6"}, nil
	}
	origCleanup := cleanupLegacyFn
	t.Cleanup(func() { cleanupLegacyFn = origCleanup })
	cleanupLegacyFn = func(context.Context, *agents.Agent, string) error { return nil }
	stateErr := &installer.StateError{Path: ".state.json", Err: fs.ErrPermission}
	recordPluginInstallsFn = func(context.Context, string, map[string]installer.PluginRecord, string) error { return stateErr }
	_, err = executePlan(cmdio.MockDiscard(t.Context()), nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, inv, false)
	require.ErrorIs(t, err, stateErr)
	assert.NotErrorIs(t, err, installer.ErrNoSkillsResolved)
}

// A detected plugin agent whose CLI is not on PATH (a desktop or IDE install:
// ~/.claude exists, `claude` does not) gets skill files in the same skills
// batch as the other agents, or keeps the plugin it already has. This used to
// print "Skipped Claude Code" and leave the agent with nothing.
func TestExecutePlanWhenTheCLIIsMissing(t *testing.T) {
	cases := []struct {
		name            string
		pluginInstalled bool
		wantBatches     [][]string
		wantOutcomes    []string
		projectPlugin   bool
		manifestPlugin  bool // Claude Code's own manifest shows the plugin
		wantCleanup     []string
		wantNotice      string
	}{
		{
			name:        "skill files instead of the plugin",
			wantBatches: [][]string{{agents.NameCursor, agents.NameClaudeCode}},
			// The plugin failure stays visible to telemetry next to the skills
			// success.
			wantOutcomes: []string{"claude-code plugin failed CLI_NOT_ON_PATH", "cursor skills installed ", "claude-code skills installed "},
			wantNotice:   "Skipped the plugin for Claude Code: 'claude' is not on PATH. Installing skill files instead; to switch to the plugin, install the Claude Code CLI so 'claude' is on PATH and run 'databricks aitools install --scope=global --agents=claude-code'.",
		},
		{
			name:          "skill files instead, next to a project plugin",
			projectPlugin: true,
			wantBatches:   [][]string{{agents.NameCursor, agents.NameClaudeCode}},
			wantOutcomes:  []string{"claude-code plugin failed CLI_NOT_ON_PATH", "cursor skills installed ", "claude-code skills installed "},
			wantNotice:    "Installing skill files instead; to switch to the plugin, install the Claude Code CLI so 'claude' is on PATH and run 'databricks aitools install --scope=global --agents=claude-code'.\nClaude Code has the databricks plugin for this project, so this project will see both the plugin and the global skill files.\n",
		},
		{
			name:            "plugin already installed is kept",
			pluginInstalled: true,
			manifestPlugin:  true,
			wantBatches:     [][]string{{agents.NameCursor}},
			wantOutcomes:    []string{"claude-code plugin kept CLI_NOT_ON_PATH", "cursor skills installed "},
			// Raw skills from an earlier fallback would duplicate the plugin's.
			wantCleanup: []string{agents.NameClaudeCode},
			wantNotice:  "Claude Code already has the databricks plugin; it was not refreshed because 'claude' is not on PATH. To refresh it, install the Claude Code CLI so 'claude' is on PATH and run 'databricks aitools install --scope=global --agents=claude-code'.",
		},
		{
			// Only the CLI's record says so (as for Codex and Copilot), and it
			// outlives a removal with the agent's own CLI: the raw skills stay.
			name:            "plugin kept on the CLI's record alone",
			pluginInstalled: true,
			wantBatches:     [][]string{{agents.NameCursor}},
			wantOutcomes:    []string{"claude-code plugin kept CLI_NOT_ON_PATH", "cursor skills installed "},
			wantNotice:      "Claude Code already has the databricks plugin",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// First: it resets the environment set below.
			isolateAgents(t)
			t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
			fakeBinsOnPath(t)
			// The fallback rewrites the install state under HOME.
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			stubPluginAlreadyInstalled(t, tc.pluginInstalled)
			skills := setupInstallMock(t)
			plugins := setupPluginMock(t)
			var cleaned []string
			orig := cleanupLegacyFn
			t.Cleanup(func() { cleanupLegacyFn = orig })
			cleanupLegacyFn = func(_ context.Context, a *agents.Agent, _ string) error {
				cleaned = append(cleaned, a.Name)
				return nil
			}

			// Detected by its config dir (a desktop or IDE install).
			t.Chdir(t.TempDir())
			claude := agents.ByName(agents.NameClaudeCode)
			require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "plugins"), 0o755))
			if tc.manifestPlugin {
				writeClaudeManifest(t, home, "user", "")
			}
			cursor := &agents.Agent{Name: agents.NameCursor, DisplayName: "Cursor"}
			ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
			plan := buildPlan(t.Context(), []*agents.Agent{cursor, claude}, installer.ScopeGlobal, false, false)
			// planItemFor reads the project plugin from Claude Code's manifest
			// (see TestPluginAlreadyInstalledRespectsTheScope); set it here.
			plan[1].projectPlugin = tc.projectPlugin
			outcomes, err := executePlan(ctx, nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, installInvocation{scope: installer.ScopeGlobal}, false)

			require.NoError(t, err)
			assert.Empty(t, *plugins, "no plugin install without the CLI")
			var batches [][]string
			for _, c := range *skills {
				batches = append(batches, c.agents)
			}
			assert.Equal(t, tc.wantBatches, batches)
			var got []string
			for _, o := range outcomes {
				got = append(got, fmt.Sprintf("%s %s %s %s", o.agent.Name, o.delivery, o.status, o.errorCategory))
			}
			assert.Equal(t, tc.wantOutcomes, got)
			assert.Equal(t, tc.wantCleanup, cleaned)
			assert.Contains(t, stderr.String(), tc.wantNotice)
		})
	}
}

// After a picker selection the recovery re-run names the agents, and a named
// agent never falls back, so a failed skills batch that included an agent
// whose CLI is missing is re-run with --skills-only.
func TestExecutePlanSkillsFailureWithAFallbackAgentSuggestsSkillsOnly(t *testing.T) {
	isolateAgents(t)
	home := setupTestAgents(t) // ~/.claude without `claude`: a desktop install
	fakeBinsOnPath(t)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	t.Chdir(t.TempDir())
	stubPluginAlreadyInstalled(t, false)
	origSkills := installSkillsForAgentsFn
	t.Cleanup(func() { installSkillsForAgentsFn = origSkills })
	fetchErr := &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: 503, Err: errors.New("failed to fetch manifest: HTTP 503")}
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		return fetchErr
	}
	require.DirExists(t, filepath.Join(home, ".claude"))

	claude := agents.ByName(agents.NameClaudeCode)
	cursor := agents.ByName(agents.NameCursor)
	plan := buildPlan(t.Context(), []*agents.Agent{cursor, claude}, installer.ScopeGlobal, false, false)
	require.True(t, plan[1].cliMissing)

	picked := installInvocation{scope: installer.ScopeGlobal, agents: "cursor,claude-code"}
	_, err := executePlan(cmdio.MockDiscard(t.Context()), nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, picked, true)
	require.ErrorIs(t, err, fetchErr)
	err = withRecovery(t.Context(), picked, err)
	// The value's quoting depends on the shell, so check its parts.
	assert.Contains(t, err.Error(), "cursor,claude-code")
	assert.Contains(t, err.Error(), " --skills-only")

	// Without a picker selection the re-run detects the same agents, which
	// fall back again, so it stays as it is.
	_, err = executePlan(cmdio.MockDiscard(t.Context()), nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, installInvocation{scope: installer.ScopeGlobal}, true)
	err = withRecovery(t.Context(), installInvocation{scope: installer.ScopeGlobal}, err)
	assert.NotContains(t, err.Error(), "--skills-only")
}

// A release with no skill this CLI can install skips the skills agents when a
// plugin was installed. After a picker selection that named a fallback agent,
// the skip's step re-runs the skills agents with --skills-only, as a failed
// skills batch does: re-run by name, the agent would try its plugin.
func TestExecutePlanNoSkillsWithAFallbackAgentSuggestsSkillsOnly(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t) // ~/.claude without `claude`: a desktop install
	fakeBinsOnPath(t, "codex")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	t.Chdir(t.TempDir())
	stubPluginAlreadyInstalled(t, false)
	stubLegacyCleanup(t)
	plugins := setupPluginMock(t)
	origSkills := installSkillsForAgentsFn
	t.Cleanup(func() { installSkillsForAgentsFn = origSkills })
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		return installer.ErrNoSkillsResolved
	}

	claude := agents.ByName(agents.NameClaudeCode)
	codex := testPluginAgent(agents.NameCodex, "Codex CLI", "codex")
	plan := buildPlan(t.Context(), []*agents.Agent{claude, codex}, installer.ScopeGlobal, false, false)
	require.True(t, plan[0].cliMissing)

	picked := installInvocation{scope: installer.ScopeGlobal, agents: "claude-code,codex"}
	outcomes, err := executePlan(cmdio.MockDiscard(t.Context()), nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, picked, true)
	require.NoError(t, err)
	require.Len(t, *plugins, 1)
	var skipped *agentOutcome
	for i := range outcomes {
		if outcomes[i].delivery == deliverySkills {
			skipped = &outcomes[i]
		}
	}
	require.NotNil(t, skipped)
	assert.Equal(t, agents.NameClaudeCode, skipped.agent.Name)
	assert.Contains(t, skipped.recovery, "claude-code")
	assert.Contains(t, skipped.recovery, " --skills-only")
	assert.NotContains(t, skipped.recovery, "codex")
}

// The plan decides the fallback, so the picker label, the plan summary and the
// outcomes agree on it. It applies only to a detected agent whose CLI is not on
// PATH: a named agent keeps failing with the commands to run.
func TestPlanItemForAgentWithoutCLI(t *testing.T) {
	cases := []struct {
		name            string
		agent           string
		scope           string
		onPath          bool
		explicit        bool
		pluginInstalled bool
		wantDelivery    delivery
		wantCLIMissing  bool
		wantLabel       string
	}{
		{name: "CLI on PATH", agent: agents.NameClaudeCode, scope: installer.ScopeGlobal, onPath: true, wantDelivery: deliveryPlugin, wantLabel: "plugin"},
		{name: "CLI missing", agent: agents.NameClaudeCode, scope: installer.ScopeGlobal, wantDelivery: deliverySkills, wantCLIMissing: true, wantLabel: "skills · CLI not found"},
		{name: "CLI missing, plugin installed", agent: agents.NameClaudeCode, scope: installer.ScopeGlobal, pluginInstalled: true, wantDelivery: deliveryPlugin, wantCLIMissing: true, wantLabel: "plugin installed · CLI not found"},
		{name: "CLI missing, named via --agents", agent: agents.NameClaudeCode, scope: installer.ScopeGlobal, explicit: true, wantDelivery: deliveryPlugin, wantLabel: "plugin · CLI not found"},
		{name: "CLI missing, scope unsupported", agent: agents.NameCodex, scope: installer.ScopeProject, wantDelivery: deliverySkip, wantLabel: "plugin · CLI not found · Codex CLI is user-only; project scope is not supported"},
		// Neither its config dir nor its CLI exists: the picker still lists it,
		// but picking it must not write skill files for an agent that is not
		// installed.
		{name: "not installed", agent: agents.NameCopilot, scope: installer.ScopeGlobal, wantDelivery: deliveryPlugin, wantLabel: "not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := setupTestAgents(t)
			require.NoError(t, os.MkdirAll(filepath.Join(home, ".codex"), 0o755))
			if tc.onPath {
				fakeBinsOnPath(t, "claude", "codex")
			} else {
				fakeBinsOnPath(t)
			}
			stubPluginAlreadyInstalled(t, tc.pluginInstalled)
			a := agents.ByName(tc.agent)

			item := planItemFor(t.Context(), a, tc.scope, false, tc.explicit)
			assert.Equal(t, tc.wantDelivery, item.delivery)
			assert.Equal(t, tc.wantCLIMissing, item.cliMissing)
			assert.Equal(t, tc.wantLabel, agentChoiceLabel(t.Context(), a, item, false))
		})
	}
}

func TestExecutePlanSkillsFailureStillInstallsPlugins(t *testing.T) {
	setupTestAgents(t) // temp HOME for the real state file below
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	fakeBinsOnPath(t, "claude")
	plugins := setupPluginMock(t)
	recordPluginInstallsFn = installer.RecordPluginInstalls
	origSkills := installSkillsForAgentsFn
	origCleanup := cleanupLegacyFn
	t.Cleanup(func() { installSkillsForAgentsFn = origSkills; cleanupLegacyFn = origCleanup })
	cleanupLegacyFn = func(context.Context, *agents.Agent, string) error { return nil }
	fetchErr := &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: 503, Err: errors.New("failed to fetch manifest: HTTP 503")}
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		return fetchErr
	}

	ctx := cmdio.MockDiscard(t.Context())
	// Cursor's skills are at v0.2.5 from an earlier install.
	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{
		SchemaVersion: 2,
		Release:       "v0.2.5",
		Skills:        map[string]string{"databricks": "0.2.5"},
	}))

	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	cursor := &agents.Agent{Name: agents.NameCursor, DisplayName: "Cursor"}
	plan := buildPlan(t.Context(), []*agents.Agent{cursor, claude}, installer.ScopeGlobal, false, false)
	outcomes, err := executePlan(ctx, nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, installInvocation{scope: installer.ScopeGlobal}, false)

	// The skills failure is still the command's error, but it no longer stops
	// Claude Code's plugin install.
	require.ErrorIs(t, err, fetchErr)
	require.Len(t, *plugins, 1)
	require.Len(t, outcomes, 1)
	assert.Equal(t, agents.NameClaudeCode, outcomes[0].agent.Name)
	assert.Equal(t, outcomeInstalled, outcomes[0].status)

	// The plugin is recorded, but the release stays at the skills' v0.2.5 so
	// version and update don't report the old skills as up to date.
	state, err := installer.LoadState(dir)
	require.NoError(t, err)
	assert.Contains(t, state.Plugins, agents.NameClaudeCode)
	assert.Equal(t, "v0.2.5", state.Release)
}

func TestExecutePlanSkillsFailureKeepsUntrackedSkillsVisible(t *testing.T) {
	setupTestAgents(t)
	fakeBinsOnPath(t, "claude")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	plugins := setupPluginMock(t)
	recordPluginInstallsFn = installer.RecordPluginInstalls
	origSkills := installSkillsForAgentsFn
	origCleanup := cleanupLegacyFn
	t.Cleanup(func() { installSkillsForAgentsFn = origSkills; cleanupLegacyFn = origCleanup })
	cleanupLegacyFn = func(context.Context, *agents.Agent, string) error { return nil }
	fetchErr := &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: 503, Err: errors.New("failed to fetch manifest: HTTP 503")}
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		return fetchErr
	}

	ctx := cmdio.MockDiscard(t.Context())
	// Skills from an install that predates state tracking: on disk, no state file.
	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "databricks"), 0o755))
	require.True(t, installer.HasUntrackedSkills(ctx, installer.ScopeGlobal))

	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	cursor := &agents.Agent{Name: agents.NameCursor, DisplayName: "Cursor"}
	plan := buildPlan(t.Context(), []*agents.Agent{cursor, claude}, installer.ScopeGlobal, false, false)
	outcomes, err := executePlan(ctx, nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, installInvocation{scope: installer.ScopeGlobal}, false)

	require.ErrorIs(t, err, fetchErr)
	require.Len(t, *plugins, 1)
	require.Len(t, outcomes, 1)
	assert.Equal(t, outcomeInstalled, outcomes[0].status)

	// No plugin-only state file is created, so update still reports the
	// untracked skills instead of treating the scope as plugin-only.
	state, err := installer.LoadState(dir)
	require.NoError(t, err)
	assert.Nil(t, state)
	assert.True(t, installer.HasUntrackedSkills(ctx, installer.ScopeGlobal))
}

func TestExecutePlanSkillsAndStateWriteFailuresAreJoined(t *testing.T) {
	setupTestAgents(t) // temp HOME: the skills failure path reads the global skills dir
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	fakeBinsOnPath(t, "claude")
	setupPluginMock(t)
	origSkills := installSkillsForAgentsFn
	origCleanup := cleanupLegacyFn
	t.Cleanup(func() { installSkillsForAgentsFn = origSkills; cleanupLegacyFn = origCleanup })
	cleanupLegacyFn = func(context.Context, *agents.Agent, string) error { return nil }
	skillsErr := errors.New("failed to write skill files")
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		return skillsErr
	}
	stateErr := &installer.StateError{Path: ".state.json", Err: errors.New("permission denied")}
	recordPluginInstallsFn = func(context.Context, string, map[string]installer.PluginRecord, string) error {
		return stateErr
	}

	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	cursor := &agents.Agent{Name: agents.NameCursor, DisplayName: "Cursor"}
	ctx := cmdio.MockDiscard(t.Context())
	plan := buildPlan(t.Context(), []*agents.Agent{cursor, claude}, installer.ScopeGlobal, false, false)
	_, err := executePlan(ctx, nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, installInvocation{scope: installer.ScopeGlobal}, false)

	// Both failures reach the user, the skills one first.
	require.ErrorIs(t, err, skillsErr)
	require.ErrorIs(t, err, stateErr)
	assert.Equal(t, skillsErr.Error()+"\n"+stateErr.Error(), err.Error())
	// The category is picked by error type, not by order: the typed state
	// error outranks the untyped skills error.
	assert.Equal(t, protos.AitoolsErrorCategoryStateInvalid, classifyInstallError(err))
}

// --- RunE: skills-only path (config-dir detection, no plugin) ---

func TestInstallSkillsOnlyAllAgents(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills-only"})

	require.NoError(t, cmd.Execute())
	require.Len(t, *calls, 1)
	assert.Len(t, (*calls)[0].agents, 2)
	assert.Nil(t, (*calls)[0].opts.SpecificSkills)
}

func TestInstallSkillsOnlySpecificSkills(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills-only", "--skills", "databricks,databricks-apps"})

	require.NoError(t, cmd.Execute())
	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"databricks", "databricks-apps"}, (*calls)[0].opts.SpecificSkills)
}

func TestInstallSkillsOnlyExperimental(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills-only", "--experimental"})

	require.NoError(t, cmd.Execute())
	require.Len(t, *calls, 1)
	assert.True(t, (*calls)[0].opts.IncludeExperimental)
}

// --- RunE: plugin-first default path ---

func TestInstallPluginFirstDefault(t *testing.T) {
	setupTestAgents(t) // ~/.claude exists
	fakeBinsOnPath(t, "claude")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	plugins := setupPluginMock(t)
	skills := setupInstallMock(t)

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	cmd := newTestInstallCmd()
	cmd.SetContext(telemetry.WithNewLogger(ctx))

	require.NoError(t, cmd.Execute())
	require.Len(t, *plugins, 1)
	assert.Equal(t, agents.NameClaudeCode, (*plugins)[0].agent)
	assert.Equal(t, agentScopeUser, (*plugins)[0].scope)
	// Claude (a real plugin agent) must not get raw skills, but Cursor (manual-only
	// plugin) does, plus a plugin recommendation.
	require.Len(t, *skills, 1)
	assert.Equal(t, []string{agents.NameCursor}, (*skills)[0].agents)
	assert.Contains(t, stderr.String(), "Installing databricks plugin for Claude Code...")
}

func TestInstallInteractivePickerInstallsPickedAgent(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	fakeBinsOnPath(t, "claude")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	plugins := setupPluginMock(t)
	// Project, not the global default, so the scope assertions below can fail.
	setupScopeMock(t, installer.ScopeProject)

	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })
	pickerScope := ""
	promptAgentSelection = func(_ context.Context, scope string, choices []agentChoice) ([]*agents.Agent, error) {
		pickerScope = scope
		for _, c := range choices {
			if c.agent.Name == agents.NameClaudeCode {
				return []*agents.Agent{c.agent}, nil
			}
		}
		return nil, nil
	}

	_, stderr := runInstallWithStderr(t, true)
	// The picker gets the resolved scope for its title, and the scope line
	// follows the picker.
	assert.Equal(t, installer.ScopeProject, pickerScope)
	assert.Contains(t, stderr, "Install scope: project")
	require.Len(t, *plugins, 1)
	assert.Equal(t, agents.NameClaudeCode, (*plugins)[0].agent)
}

func TestAgentPickerTitleNamesScope(t *testing.T) {
	assert.Equal(t, "Select agents to set up (global scope)", agentPickerTitle(installer.ScopeGlobal))
	assert.Equal(t, "Select agents to set up (project scope)", agentPickerTitle(installer.ScopeProject))
}

func TestPrintInstallScope(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	printInstallScope(ctx, installer.ScopeProject)
	assert.Equal(t, "\nInstall scope: project\n\n", stderr.String())
}

func TestInstallExplicitAgentWorksUndetected(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	fakeBinsOnPath(t, "codex")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	plugins := setupPluginMock(t)

	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--agents", "codex"})

	require.NoError(t, cmd.Execute())
	require.Len(t, *plugins, 1)
	assert.Equal(t, agents.NameCodex, (*plugins)[0].agent)
}

func TestInstallOutputJSONReportsErrorCategories(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	fakeBinsOnPath(t, "codex")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")

	origInstall := installPluginForAgentFn
	origRecord := recordPluginInstallsFn
	t.Cleanup(func() { installPluginForAgentFn = origInstall; recordPluginInstallsFn = origRecord })
	installPluginForAgentFn = func(_ context.Context, a *agents.Agent, _, _ string) (installer.PluginRecord, error) {
		return installer.PluginRecord{}, &installer.BlockedError{Agent: a.Name, Reason: installer.ReasonInstallFailed, Detail: "boom"}
	}
	recordPluginInstallsFn = func(context.Context, string, map[string]installer.PluginRecord, string) error { return nil }

	var out bytes.Buffer
	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--agents", "codex", "--scope", "global", "--output", "json"})

	// Explicit --agents makes a blocked install a hard error, but the JSON result
	// is still emitted for the extension to consume. The command returns
	// ErrAlreadyPrinted so root exits non-zero without printing a duplicate
	// "Error:" line over the JSON (exercised end-to-end in
	// TestInstallOutputJSONThroughRoot).
	err := cmd.Execute()
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var got installOutput
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got.Agents, 1)
	assert.Equal(t, agents.NameCodex, got.Agents[0].Name)
	assert.Equal(t, deliveryPlugin.String(), got.Agents[0].Delivery)
	assert.Equal(t, string(outcomeFailed), got.Agents[0].Status)
	assert.Equal(t, string(protos.AitoolsErrorCategoryPluginInstallFailed), got.Agents[0].ErrorCategory)
	// A per-agent failure stays in the agent entry; it is not repeated in the
	// top-level error fields.
	assert.Empty(t, got.Error)
	assert.Empty(t, got.ErrorCategory)
}

// TestInstallOutputJSONThroughRoot runs a failing `install --output json` through
// the real root command, where the "Error:" line is actually printed (root does
// it, not cobra). It guards the contract that a failed JSON run writes only the
// JSON to stdout and no text error to stderr.
func TestInstallOutputJSONThroughRoot(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	fakeBinsOnPath(t, "codex")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")

	origInstall := installPluginForAgentFn
	origRecord := recordPluginInstallsFn
	t.Cleanup(func() { installPluginForAgentFn = origInstall; recordPluginInstallsFn = origRecord })
	installPluginForAgentFn = func(_ context.Context, a *agents.Agent, _, _ string) (installer.PluginRecord, error) {
		return installer.PluginRecord{}, &installer.BlockedError{Agent: a.Name, Reason: installer.ReasonInstallFailed, Detail: "boom"}
	}
	recordPluginInstallsFn = func(context.Context, string, map[string]installer.PluginRecord, string) error { return nil }

	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	cli := root.New(ctx)
	cli.AddCommand(NewInstallCmd())
	var out, errOut bytes.Buffer
	cli.SetOut(&out)
	cli.SetErr(&errOut)
	cli.SetArgs([]string{"install", "--agents", "codex", "--scope", "global", "--output", "json"})

	err := root.Execute(ctx, cli)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.NotContains(t, errOut.String(), "Error:")

	var got installOutput
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got.Agents, 1)
	assert.Equal(t, string(outcomeFailed), got.Agents[0].Status)
}

func TestInstallOutputJSONTopLevelFailure(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	orig := installSkillsForAgentsFn
	t.Cleanup(func() { installSkillsForAgentsFn = orig })
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		return &installer.SkillError{Skill: "databricks", Reason: installer.ReasonSkillNotFound, Detail: "not found"}
	}

	var out bytes.Buffer
	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetOut(&out)
	// Cursor is skills-only, so this fails in the skills-group path, which records
	// no per-agent outcome.
	cmd.SetArgs([]string{"--agents", "cursor", "--scope", "global", "--output", "json"})

	// A top-level failure has no per-agent entry, so it must still be represented
	// in the JSON (not just a bare non-zero exit). With no plugin agent in the
	// plan, the agents array is empty here.
	err := cmd.Execute()
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var got installOutput
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Empty(t, got.Agents)
	assert.Contains(t, got.Error, "databricks")
	assert.Equal(t, string(protos.AitoolsErrorCategorySkillNotFound), got.ErrorCategory)
}

func TestInstallOutputJSONRequiresNonInteractiveFlags(t *testing.T) {
	setupTestAgents(t)

	cases := []struct {
		name string
		args []string
		want []string // substrings the error must name
	}{
		{
			name: "no scope or agents",
			args: []string{"--output", "json"},
			want: []string{"--scope", "--agents"},
		},
		{
			name: "agents without scope",
			args: []string{"--agents", "claude-code", "--output", "json"},
			want: []string{"--scope"},
		},
		{
			name: "scope without agents",
			args: []string{"--scope", "global", "--output", "json"},
			want: []string{"--agents"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
			cmd := newTestInstallCmd()
			cmd.SetContext(ctx)
			cmd.SetOut(&out)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			require.Error(t, err)
			for _, w := range tc.want {
				assert.Contains(t, err.Error(), w)
			}
			// The command errors before rendering, so no JSON is emitted.
			assert.Empty(t, out.String())
		})
	}
}

func TestInstallUnknownAgentErrors(t *testing.T) {
	setupTestAgents(t)
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--agents", "invalid-agent"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent")
	assert.Contains(t, err.Error(), "Available agents:")
}

func TestInstallNoAgentsDetected(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	// Neutralize XDG_CONFIG_HOME so a real opencode config on the dev machine
	// is not picked up through it.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	fakeBinsOnPath(t) // no agent binaries
	plugins := setupPluginMock(t)
	skills := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)

	// Nothing was set up, so the run fails; the message above it says what to do,
	// so no generic recovery step is appended.
	err := cmd.Execute()
	require.ErrorIs(t, err, errNoAgentSetUp)
	assert.Equal(t, "nothing was installed; follow the steps above", err.Error())
	assert.Empty(t, *plugins)
	assert.Empty(t, *skills)
}

func TestInstallSkillsRequiresSkillsOnlyOrPath(t *testing.T) {
	setupTestAgents(t)
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills", "databricks"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--skills requires --skills-only or --path")
}

func TestInstallInteractivePickerErrorPropagates(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	setupScopeMock(t, installer.ScopeGlobal)

	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })
	promptAgentSelection = func(context.Context, string, []agentChoice) ([]*agents.Agent, error) {
		return nil, errors.New("at least one agent must be selected")
	}

	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)

	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one agent")
}

func TestInstallPathConflictsWithSkillsOnly(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills-only", "--path", "./out"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use --skills-only with --path")
}

func TestInstallOutputJSONConflictsWithPath(t *testing.T) {
	var out bytes.Buffer
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--path", "./out", "--output", "json"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use --output json with --path")
	// The command errors before dumping, so nothing is emitted.
	assert.Empty(t, out.String())
}

// --- Scope flag parsing (exercised via the skills path so opts.Scope is observable) ---

func TestInstallScopeFlag(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantScope string
		wantErr   string
	}{
		{name: "scope project", args: []string{"--skills-only", "--scope", "project"}, wantScope: installer.ScopeProject},
		{name: "scope global", args: []string{"--skills-only", "--scope", "global"}, wantScope: installer.ScopeGlobal},
		{name: "scope both rejected", args: []string{"--scope", "both"}, wantErr: "--scope=both is not supported"},
		{name: "scope invalid value", args: []string{"--scope", "all"}, wantErr: `invalid --scope "all"`},
		{name: "scope conflicts with legacy", args: []string{"--scope", "global", "--project"}, wantErr: "cannot use --scope with --project or --global"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestAgents(t)
			calls := setupInstallMock(t)

			ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
			cmd := newTestInstallCmd()
			cmd.SetContext(ctx)
			cmd.SetArgs(tt.args)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			err := cmd.Execute()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, *calls, 1)
			assert.Equal(t, tt.wantScope, (*calls)[0].opts.Scope)
		})
	}
}

func TestInstallGlobalAndProjectErrors(t *testing.T) {
	setupTestAgents(t)
	setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--global", "--project"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use --global and --project together")
}

func TestInstallNoFlagNonInteractiveUsesGlobal(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills-only"})

	require.NoError(t, cmd.Execute())
	require.Len(t, *calls, 1)
	assert.Equal(t, installer.ScopeGlobal, (*calls)[0].opts.Scope)
}

// --- Positional-arg rejections ---

func TestInstallRejectsPositionalArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"databricks-jobs"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestUpdateRejectsPositionalArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewUpdateCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"databricks"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestUninstallRejectsPositionalArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewUninstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"databricks"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestListRejectsPositionalArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewListCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"databricks"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestVersionRejectsPositionalArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewVersionCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"databricks"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

// --- resolveAgentNames ---

func TestResolveAgentNamesValid(t *testing.T) {
	ctx := t.Context()
	result, err := resolveAgentNames(ctx, "claude-code,cursor")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "claude-code", result[0].Name)
	assert.Equal(t, "cursor", result[1].Name)
}

func TestResolveAgentNamesUnknown(t *testing.T) {
	ctx := t.Context()
	_, err := resolveAgentNames(ctx, "claude-code,invalid-agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent")
	assert.Contains(t, err.Error(), "invalid-agent")
}

func TestResolveAgentNamesEmpty(t *testing.T) {
	ctx := t.Context()
	_, err := resolveAgentNames(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no agents specified")
}

func TestResolveAgentNamesDuplicatesDeduplicates(t *testing.T) {
	ctx := t.Context()
	result, err := resolveAgentNames(ctx, "claude-code,claude-code")
	require.NoError(t, err)
	assert.Len(t, result, 1, "duplicate agent names should be deduplicated")
	assert.Equal(t, "claude-code", result[0].Name)
}

func TestResolveScopeValidation(t *testing.T) {
	tests := []struct {
		name    string
		project bool
		global  bool
		want    string
		wantErr string
	}{
		{name: "neither", want: installer.ScopeGlobal},
		{name: "global only", global: true, want: installer.ScopeGlobal},
		{name: "project only", project: true, want: installer.ScopeProject},
		{name: "both", project: true, global: true, wantErr: "cannot use --global and --project together"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveScope(tc.project, tc.global)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

// setupAgentCallerInstall sets up an install run by the agent caller that
// agentEnv names (key/value pairs), with Claude Code and Cursor detected. Every
// prompt fails the test if shown. isolateAgents clears the environment, so it
// runs first and the HOME, PATH and skills ref set below survive.
func setupAgentCallerInstall(t *testing.T, agentEnv ...string) (plugins *[]pluginCall, skills *[]installCall, scopePrompted *bool) {
	t.Helper()
	isolateAgents(t, agentEnv...)
	setupTestAgents(t) // ~/.claude and ~/.cursor
	fakeBinsOnPath(t, "claude")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	plugins = setupPluginMock(t)
	skills = setupInstallMock(t)
	scopePrompted = setupScopeMock(t, installer.ScopeProject)
	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })
	promptAgentSelection = func(context.Context, string, []agentChoice) ([]*agents.Agent, error) {
		t.Fatal("agent picker shown to an agent caller")
		return nil, nil
	}
	return plugins, skills, scopePrompted
}

// runInstallWithStderr runs install with args and returns the telemetry
// context and everything written to stderr.
func runInstallWithStderr(t *testing.T, promptSupported bool, args ...string) (context.Context, string) {
	t.Helper()
	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: promptSupported})
	go drainReader(test.Stdout)
	// Buffered so the reader never blocks if the test stops before receiving.
	stderr := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(test.Stderr)
		stderr <- b
	}()
	ctx = telemetry.WithNewLogger(ctx)
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs(args)

	err := cmd.Execute()
	test.Done()
	out := string(<-stderr)
	require.NoError(t, err, "stderr:\n%s", out)
	return ctx, out
}

func TestInstallAgentCallerInPTYNeverPrompts(t *testing.T) {
	// Cursor and Copilot run agent commands in a pseudo-terminal, so stdin is a
	// TTY, but no human answers the prompts. The run must use the
	// non-interactive defaults instead of blocking on the scope select.
	plugins, skills, scopePrompted := setupAgentCallerInstall(t, "CURSOR_AGENT", "1")

	ctx, stderr := runInstallWithStderr(t, true)
	assert.Contains(t, stderr, "No prompts shown because a coding agent (cursor) was detected. Pass --scope and --agents to databricks aitools install to choose what to install.")
	events := funnelEvents(ctx)
	require.NotEmpty(t, events)
	assert.False(t, events[0].Interactive)
	assert.False(t, *scopePrompted)
	require.Len(t, *plugins, 1)
	assert.Equal(t, agents.NameClaudeCode, (*plugins)[0].agent)
	require.Len(t, *skills, 1)
	assert.Equal(t, installer.ScopeGlobal, (*skills)[0].opts.Scope)
}

func TestInstallAgentCallerNoteNamesOnlyMissingFlags(t *testing.T) {
	setupAgentCallerInstall(t, "CURSOR_AGENT", "1")

	_, stderr := runInstallWithStderr(t, true, "--scope", "global")
	assert.Contains(t, stderr, "No prompts shown because a coding agent (cursor) was detected. Pass --agents to databricks aitools install to choose what to install.")
}

func TestInstallAgentCallerNoteNamesAgent(t *testing.T) {
	cases := []struct {
		name string
		env  []string
		want string
	}{
		// The SDK reports copilot-cli; the note shows the --agents value.
		{name: "registry alias", env: []string{"COPILOT_CLI", "1"}, want: "because a coding agent (copilot) was detected."},
		// vscode-agent has no registry entry, so no name is shown.
		{name: "no registry entry", env: []string{"VSCODE_AGENT", "1"}, want: "because a coding agent was detected."},
		{name: "nested agents", env: []string{"CURSOR_AGENT", "1", "CLAUDECODE", "1"}, want: "because more than one coding agent was detected."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plugins, _, _ := setupAgentCallerInstall(t, tc.env...)

			_, stderr := runInstallWithStderr(t, true)
			assert.Contains(t, stderr, tc.want)
			require.Len(t, *plugins, 1)
		})
	}
}

func TestInstallAgentCallerNoteSilent(t *testing.T) {
	cases := []struct {
		name            string
		promptSupported bool
		args            []string
	}{
		// Both flags given: no prompt would have run, so there is nothing to explain.
		{name: "scope and agents passed", promptSupported: true, args: []string{"--scope", "global", "--agents", "claude-code"}},
		// No terminal: prompts are skipped for every caller, as before.
		{name: "no terminal", promptSupported: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plugins, _, _ := setupAgentCallerInstall(t, "CURSOR_AGENT", "1")

			_, stderr := runInstallWithStderr(t, tc.promptSupported, tc.args...)
			assert.NotContains(t, stderr, "No prompts shown")
			require.Len(t, *plugins, 1)
		})
	}
}

func TestInstallPickerAbortSuggestsFlags(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	setupScopeMock(t, installer.ScopeGlobal)
	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })
	promptAgentSelection = func(context.Context, string, []agentChoice) ([]*agents.Agent, error) {
		return nil, huh.ErrUserAborted
	}

	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.ErrorIs(t, err, huh.ErrUserAborted)
	assert.Equal(t, "user aborted\nTo install without prompts, name the scope and agents, for example: databricks aitools install --scope=global --agents=<name>", err.Error())
}

// The skills fallback is skipped only when the plugin already serves the
// install's scope: a global install serves every project, so only a
// machine-wide plugin covers it, and a project or local plugin counts only for
// a project-scope install inside its own project.
func TestPluginAlreadyInstalledRespectsTheScope(t *testing.T) {
	project, other := t.TempDir(), t.TempDir()
	// A project install is shared through the repository, so only a
	// project-scope plugin of this project covers it; a user or local plugin is
	// personal. A fallback warns when this directory sees a plugin that does not
	// cover the install next to the skill files: a project or local plugin for a
	// global install, a user or local plugin for a project install.
	cases := []struct {
		name                   string
		scope, projectPath     string
		wantGlobal, wantInProj bool
		wantWarning            bool
		wantProjectWarning     bool
	}{
		{name: "user scope", scope: "user", wantGlobal: true, wantProjectWarning: true},
		{name: "project scope of this project", scope: "project", projectPath: project, wantInProj: true, wantWarning: true},
		{name: "local scope of this project", scope: "local", projectPath: project, wantWarning: true, wantProjectWarning: true},
		{name: "project scope of another project", scope: "project", projectPath: other},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateAgents(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			writeClaudeManifest(t, home, tc.scope, tc.projectPath)
			t.Chdir(project)

			claude := agents.ByName(agents.NameClaudeCode)
			assert.Equal(t, tc.wantGlobal, pluginAlreadyInstalled(t.Context(), claude, installer.ScopeGlobal), "global install")
			assert.Equal(t, tc.wantInProj, pluginAlreadyInstalled(t.Context(), claude, installer.ScopeProject), "project install")

			// Without the CLI, a global install that falls back to skill files
			// flags a project or local plugin of this project: it sees both.
			fakeBinsOnPath(t)
			item := planItemFor(t.Context(), claude, installer.ScopeGlobal, false, false)
			assert.Equal(t, tc.wantWarning, item.projectPlugin, "project plugin next to a global fallback")
			item = planItemFor(t.Context(), claude, installer.ScopeProject, false, false)
			assert.Equal(t, tc.wantProjectWarning, item.projectPlugin, "personal plugin next to a project fallback")
		})
	}
}

// Removing the plugin with Claude Code's own CLI leaves the CLI's state record
// behind; Claude's manifest is authoritative, so the record must not keep the
// skills fallback from running.
func TestPluginAlreadyInstalledTrustsClaudesManifestOverTheState(t *testing.T) {
	isolateAgents(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(t.TempDir())
	require.NoError(t, installer.RecordPluginInstalls(t.Context(), installer.ScopeGlobal, map[string]installer.PluginRecord{
		agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks"},
	}, "v0.2.6"))
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude", "plugins", "installed_plugins.json"), []byte(`{"plugins":{}}`), 0o600))

	claude := agents.ByName(agents.NameClaudeCode)
	assert.False(t, pluginAlreadyInstalled(t.Context(), claude, installer.ScopeGlobal))

	// The fallback install drops the stale record, so list stops reporting the
	// removed plugin and update reconciles Claude Code's skill files again.
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	fakeBinsOnPath(t)
	setupInstallMock(t)
	ctx := cmdio.MockDiscard(t.Context())
	plan := buildPlan(t.Context(), []*agents.Agent{claude}, installer.ScopeGlobal, false, false)
	_, err := executePlan(ctx, nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, installInvocation{scope: installer.ScopeGlobal}, true)
	require.NoError(t, err)

	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)
	state, err := installer.LoadState(dir)
	require.NoError(t, err)
	assert.NotContains(t, state.Plugins, agents.NameClaudeCode)
	fileAgents, _ := excludePluginAgents(ctx, installer.ScopeGlobal, []*agents.Agent{claude}, state)
	assert.Equal(t, []*agents.Agent{claude}, fileAgents)
	for _, e := range buildAgentEntries(ctx, map[string]*installer.InstallState{installer.ScopeGlobal: state}) {
		if e.Name == agents.NameClaudeCode {
			assert.NotEqual(t, deliveryPlugin.String(), e.Installed[installer.ScopeGlobal].Delivery)
		}
	}
}

// A manifest that exists but cannot be parsed says nothing about the plugin,
// so the CLI's record decides instead of dropping it.
func TestPluginAlreadyInstalledUsesTheStateWhenTheManifestIsUnreadable(t *testing.T) {
	isolateAgents(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(t.TempDir())
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude", "plugins", "installed_plugins.json"), []byte(`{not json`), 0o600))

	claude := agents.ByName(agents.NameClaudeCode)
	assert.False(t, pluginAlreadyInstalled(t.Context(), claude, installer.ScopeGlobal))
	require.NoError(t, installer.RecordPluginInstalls(t.Context(), installer.ScopeGlobal, map[string]installer.PluginRecord{
		agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks"},
	}, "v0.2.6"))
	assert.True(t, pluginAlreadyInstalled(t.Context(), claude, installer.ScopeGlobal))
}

func TestPluginAlreadyInstalledReadsTheScopeState(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	claude.ConfigDir = func(context.Context) (string, error) { return filepath.Join(tmp, ".claude"), nil }
	ctx := t.Context()
	assert.False(t, pluginAlreadyInstalled(ctx, claude, installer.ScopeGlobal))

	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, installer.RecordPluginInstalls(ctx, installer.ScopeGlobal, map[string]installer.PluginRecord{
		agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks"},
	}, "v0.2.6"))
	require.FileExists(t, filepath.Join(dir, ".state.json"))
	assert.True(t, pluginAlreadyInstalled(ctx, claude, installer.ScopeGlobal))
}

func TestInstallPathFailureHasRecoveryForThePathRun(t *testing.T) {
	setupTestAgents(t)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", "http://127.0.0.1:1")
	// A relative destination keeps the printed command independent of how the
	// host spells its temp dir; the space makes it need quoting in every shell.
	t.Chdir(t.TempDir())
	const out = "my skills"

	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	cmd := newTestInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--path", out})
	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "\nCould not download Databricks skills from 127.0.0.1:1.")
	// The POSIX-quoted command contains single quotes, so backticks delimit it.
	want := "`databricks aitools install --path='my skills'`"
	if runtime.GOOS == "windows" {
		want = `'databricks aitools install --path="my skills"'`
	}
	assert.Contains(t, err.Error(), "then re-run "+want)
}

func TestInstallFailureAfterPickerNamesTheSelectedAgents(t *testing.T) {
	isolateAgents(t)
	setupTestAgents(t)
	setupScopeMock(t, installer.ScopeGlobal)
	t.Setenv("DATABRICKS_SKILLS_BASE_URL", "")
	origPicker := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPicker })
	// Claude Code and Cursor are detected; the user picks only Cursor.
	promptAgentSelection = func(context.Context, string, []agentChoice) ([]*agents.Agent, error) {
		return []*agents.Agent{agents.ByName(agents.NameCursor)}, nil
	}
	origInstall := installSkillsForAgentsFn
	t.Cleanup(func() { installSkillsForAgentsFn = origInstall })
	installSkillsForAgentsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.InstallOptions) error {
		return &installer.FetchError{Stage: installer.FetchStageManifest, StatusCode: http.StatusBadGateway, Err: errors.New("failed to fetch manifest: HTTP 502")}
	}

	ioCtx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)
	cmd := newTestInstallCmd()
	cmd.SetContext(telemetry.WithNewLogger(ioCtx))
	require.NoError(t, cmd.Flags().Set("skills-only", "true"))

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "re-run 'databricks aitools install --scope=global --agents=cursor --skills-only'")
}

func TestNoAgentsMessageKeepsTheRunsScopeAndMode(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	printNoAgentsMessage(ctx, installInvocation{scope: installer.ScopeProject, skillsOnly: true})
	assert.Contains(t, stderr.String(), "Install one, then re-run 'databricks aitools install --scope=project --skills-only'. To set up an agent that was not detected, name it: databricks aitools install --scope=project --skills-only --agents=<name>")
}

// --skills-only detects agents by their config dir, so the message must not
// blame PATH.
func TestNoAgentsMessageMatchesTheDetection(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	printNoAgentsMessage(ctx, installInvocation{scope: installer.ScopeGlobal, skillsOnly: true})
	assert.Contains(t, stderr.String(), "No supported coding agent config directories found.")
	assert.NotContains(t, stderr.String(), "on PATH")

	ctx, stderr = cmdio.NewTestContextWithStderr(t.Context())
	printNoAgentsMessage(ctx, installInvocation{scope: installer.ScopeGlobal})
	assert.Contains(t, stderr.String(), "No supported coding agents found on PATH.")
}
