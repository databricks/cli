package aitools

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
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
func setupTestAgents(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
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

	// Project scope: only Claude (plugin) and Cursor (skills) support it; the
	// user-only plugin agents and files-only agents are not offered as choices.
	choices := agentChoices(ctx, installer.ScopeProject, false)
	var names []string
	for _, c := range choices {
		names = append(names, c.agent.Name)
	}
	assert.Contains(t, names, agents.NameClaudeCode)
	assert.Contains(t, names, agents.NameCursor)
	assert.NotContains(t, names, agents.NameCodex)
	assert.NotContains(t, names, agents.NameOpenCode)
	assert.NotContains(t, names, agents.NameCopilot)
	assert.NotContains(t, names, agents.NameAntigravity)
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

	// Non-explicit blocked install is a warning, not an error.
	plan := buildPlan([]*agents.Agent{claude}, installer.ScopeGlobal, false, false)
	require.NoError(t, executePlan(ctx, nil, plan, installer.InstallOptions{Scope: installer.ScopeGlobal}))

	// Explicit (--agents) blocked install is an error.
	planExplicit := buildPlan([]*agents.Agent{claude}, installer.ScopeGlobal, false, true)
	require.Error(t, executePlan(ctx, nil, planExplicit, installer.InstallOptions{Scope: installer.ScopeGlobal}))
}

// --- RunE: skills-only path (config-dir detection, no plugin) ---

func TestInstallSkillsOnlyAllAgents(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
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

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills-only", "--skills", "databricks,databricks-apps"})

	require.NoError(t, cmd.Execute())
	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"databricks", "databricks-apps"}, (*calls)[0].opts.SpecificSkills)
}

func TestInstallSkillsOnlyExperimental(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
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

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
	cmd.SetContext(ctx)

	require.NoError(t, cmd.Execute())
	require.Len(t, *plugins, 1)
	assert.Equal(t, agents.NameClaudeCode, (*plugins)[0].agent)
	assert.Equal(t, agentScopeUser, (*plugins)[0].scope)
	// Claude (a real plugin agent) must not get raw skills, but Cursor (manual-only
	// plugin) does, plus a plugin recommendation.
	require.Len(t, *skills, 1)
	assert.Equal(t, []string{agents.NameCursor}, (*skills)[0].agents)
}

func TestInstallInteractivePickerAndConfirm(t *testing.T) {
	setupTestAgents(t)
	fakeBinsOnPath(t, "claude")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	plugins := setupPluginMock(t)
	setupScopeMock(t, installer.ScopeGlobal)

	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })
	pickerCalled := false
	promptAgentSelection = func(_ context.Context, choices []agentChoice) ([]*agents.Agent, error) {
		pickerCalled = true
		for _, c := range choices {
			if c.agent.Name == agents.NameClaudeCode {
				return []*agents.Agent{c.agent}, nil
			}
		}
		return nil, nil
	}

	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)

	cmd := NewInstallCmd()
	cmd.SetContext(ctx)

	errc := make(chan error, 1)
	go func() { errc <- cmd.RunE(cmd, nil) }()

	_, err := test.Stdin.WriteString("y\n")
	require.NoError(t, err)
	require.NoError(t, test.Stdin.Flush())

	require.NoError(t, <-errc)
	assert.True(t, pickerCalled)
	require.Len(t, *plugins, 1)
	assert.Equal(t, agents.NameClaudeCode, (*plugins)[0].agent)
}

func TestInstallExplicitAgentWorksUndetected(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	fakeBinsOnPath(t, "codex")
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	plugins := setupPluginMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--agents", "codex"})

	require.NoError(t, cmd.Execute())
	require.Len(t, *plugins, 1)
	assert.Equal(t, agents.NameCodex, (*plugins)[0].agent)
}

func TestInstallUnknownAgentErrors(t *testing.T) {
	setupTestAgents(t)
	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
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
	fakeBinsOnPath(t) // no agent binaries
	plugins := setupPluginMock(t)
	skills := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
	cmd.SetContext(ctx)

	require.NoError(t, cmd.Execute())
	assert.Empty(t, *plugins)
	assert.Empty(t, *skills)
}

func TestInstallSkillsRequiresSkillsOnlyOrPath(t *testing.T) {
	setupTestAgents(t)
	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills", "databricks"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--skills requires --skills-only or --path")
}

func TestInstallInteractivePickerErrorPropagates(t *testing.T) {
	setupTestAgents(t)
	setupScopeMock(t, installer.ScopeGlobal)

	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })
	promptAgentSelection = func(_ context.Context, _ []agentChoice) ([]*agents.Agent, error) {
		return nil, errors.New("at least one agent must be selected")
	}

	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)

	cmd := NewInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one agent")
}

func TestInstallPathConflictsWithSkillsOnly(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills-only", "--path", "./out"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use --skills-only with --path")
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

			ctx := cmdio.MockDiscard(t.Context())
			cmd := NewInstallCmd()
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
	cmd := NewInstallCmd()
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

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills-only"})

	require.NoError(t, cmd.Execute())
	require.Len(t, *calls, 1)
	assert.Equal(t, installer.ScopeGlobal, (*calls)[0].opts.Scope)
}

// --- Positional-arg rejections ---

func TestInstallRejectsPositionalArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewInstallCmd()
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
