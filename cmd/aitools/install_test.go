package aitools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	// Cursor is a skills-only agent (no headless plugin install).
	cursor := &agents.Agent{Name: agents.NameCursor, DisplayName: "Cursor", Binary: "cursor-agent", SupportsProjectScope: true}
	opencode := &agents.Agent{Name: agents.NameOpenCode, DisplayName: "OpenCode", Binary: "opencode"}
	targets := []*agents.Agent{claude, cursor, opencode}

	plan := buildPlan(targets, installer.ScopeGlobal, false, false)
	assert.Equal(t, deliveryPlugin, plan[0].delivery)
	assert.Equal(t, agentScopeUser, plan[0].scope)
	assert.Equal(t, deliverySkills, plan[1].delivery) // Cursor -> skills
	assert.Equal(t, deliverySkills, plan[2].delivery)

	// --skills-only forces skills for every agent.
	skillsPlan := buildPlan(targets, installer.ScopeGlobal, true, false)
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
	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	codex := testPluginAgent(agents.NameCodex, "Codex CLI", "codex")

	plan := buildPlan([]*agents.Agent{claude, codex}, installer.ScopeProject, false, false)
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

	plan := buildPlan([]*agents.Agent{opencode, projectSkills}, installer.ScopeProject, false, false)
	assert.Equal(t, deliverySkip, plan[0].delivery)
	assert.Contains(t, plan[0].reason, "project-scoped skills")
	assert.Equal(t, deliverySkills, plan[1].delivery)

	// --skills-only does not rescue a project-incompatible agent.
	skillsOnly := buildPlan([]*agents.Agent{opencode}, installer.ScopeProject, true, false)
	assert.Equal(t, deliverySkip, skillsOnly[0].delivery)

	// Under global scope the same agent gets skills.
	globalPlan := buildPlan([]*agents.Agent{opencode}, installer.ScopeGlobal, false, false)
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
	orig := installPluginForAgentFn
	origRec := recordPluginInstallsFn
	t.Cleanup(func() { installPluginForAgentFn = orig; recordPluginInstallsFn = origRec })
	installPluginForAgentFn = func(_ context.Context, a *agents.Agent, _, _ string) (installer.PluginRecord, error) {
		return installer.PluginRecord{}, &installer.BlockedError{Agent: a.Name, Reason: installer.ReasonCLINotOnPath}
	}
	recordPluginInstallsFn = func(context.Context, string, map[string]installer.PluginRecord, string) error { return nil }

	claude := testPluginAgent(agents.NameClaudeCode, "Claude Code", "claude")
	ctx := cmdio.MockDiscard(t.Context())

	// Non-explicit blocked install is a warning, not an error, but the agent's
	// failure is still recorded in its outcome.
	plan := buildPlan([]*agents.Agent{claude}, installer.ScopeGlobal, false, false)
	outcomes, err := executePlan(ctx, nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, false)
	require.NoError(t, err)
	require.Len(t, outcomes, 1)
	assert.Equal(t, outcomeFailed, outcomes[0].status)
	assert.Equal(t, protos.AitoolsErrorCategoryCLINotOnPath, outcomes[0].errorCategory)

	// Explicit (--agents) blocked install is an error.
	planExplicit := buildPlan([]*agents.Agent{claude}, installer.ScopeGlobal, false, true)
	_, err = executePlan(ctx, nil, planExplicit, installer.InstallOptions{Scope: installer.ScopeGlobal}, false)
	require.Error(t, err)
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
	plan := buildPlan([]*agents.Agent{claude}, installer.ScopeGlobal, false, true)
	outcomes, err := executePlan(cmdio.MockDiscard(t.Context()), nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}, false)
	require.NoError(t, err)
	require.Len(t, outcomes, 1)
	assert.Equal(t, outcomeInstalled, outcomes[0].status)
	assert.Equal(t, 1, lookups)
	// The marketplace picks the plugin version, so a failed lookup records it as unknown.
	assert.Empty(t, installRef)
	assert.Empty(t, recordRef)
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
	// Cursor is skills-only, so this fails in the skills-group path, which returns
	// before appending any per-agent outcome.
	cmd.SetArgs([]string{"--agents", "cursor", "--scope", "global", "--output", "json"})

	// A top-level failure has no per-agent entry, so it must still be represented
	// in the JSON (not just a bare non-zero exit with an empty agents array).
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

	require.NoError(t, cmd.Execute())
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
