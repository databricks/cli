package aitools

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

type installCall struct {
	agents []string
	opts   installer.InstallOptions
}

func setupTestAgents(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// Create config dirs for two agents.
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))
	return tmp
}

func TestInstallAllSkillsForAllAgents(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Len(t, (*calls)[0].agents, 2)
	assert.Nil(t, (*calls)[0].opts.SpecificSkills)
}

func TestInstallSpecificSkills(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills", "databricks,databricks-apps"})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"databricks", "databricks-apps"}, (*calls)[0].opts.SpecificSkills)
}

func TestInstallSingleSkill(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--skills", "databricks"})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"databricks"}, (*calls)[0].opts.SpecificSkills)
}

func TestInstallSpecificAgents(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--agents", "claude-code"})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"claude-code"}, (*calls)[0].agents)
}

func TestInstallUnknownAgentErrors(t *testing.T) {
	setupTestAgents(t)
	setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--agents", "invalid-agent"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent")
	assert.Contains(t, err.Error(), "Available agents:")
}

func TestInstallIncludeExperimental(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--experimental"})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.True(t, (*calls)[0].opts.IncludeExperimental)
}

func TestInstallInteractivePrompt(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)
	setupScopeMock(t, installer.ScopeGlobal)

	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })

	promptCalled := false
	promptAgentSelection = func(_ context.Context, detected []*agents.Agent) ([]*agents.Agent, error) {
		promptCalled = true
		return detected[:1], nil
	}

	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()

	drain := func(r *bufio.Reader) {
		buf := make([]byte, 4096)
		for {
			_, err := r.Read(buf)
			if err != nil {
				return
			}
		}
	}
	go drain(test.Stdout)
	go drain(test.Stderr)

	cmd := newInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)

	assert.True(t, promptCalled, "prompt should be called when 2+ agents detected and interactive")
	require.Len(t, *calls, 1)
	assert.Len(t, (*calls)[0].agents, 1, "only the selected agent should be passed")
}

func TestInstallNonInteractiveUsesAllAgents(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })

	promptCalled := false
	promptAgentSelection = func(_ context.Context, detected []*agents.Agent) ([]*agents.Agent, error) {
		promptCalled = true
		return detected, nil
	}

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)

	assert.False(t, promptCalled, "prompt should not be called in non-interactive mode")
	require.Len(t, *calls, 1)
	assert.Len(t, (*calls)[0].agents, 2, "all detected agents should be used")
}

func TestInstallNoAgentsDetected(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	calls := setupInstallMock(t)
	ctx := cmdio.MockDiscard(t.Context())

	cmd := newInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	assert.Empty(t, *calls, "install should not be called when no agents detected")
}

func TestInstallAgentsFlagSkipsPrompt(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })

	promptCalled := false
	promptAgentSelection = func(_ context.Context, detected []*agents.Agent) ([]*agents.Agent, error) {
		promptCalled = true
		return detected, nil
	}

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--agents", "claude-code,cursor"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.False(t, promptCalled, "prompt should not be called when --agents is specified")
	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"claude-code", "cursor"}, (*calls)[0].agents)
}

func TestSkillsInstallDelegatesToInstall(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newSkillsInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Len(t, (*calls)[0].agents, 2)
}

func TestSkillsInstallForwardsSkillName(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newSkillsInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"databricks"})
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"databricks"}, (*calls)[0].opts.SpecificSkills)
}

func TestSkillsInstallExecuteNoArgs(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newSkillsInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Len(t, (*calls)[0].agents, 2)
	assert.Nil(t, (*calls)[0].opts.SpecificSkills)
}

func TestSkillsInstallExecuteWithSkillName(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newSkillsInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"databricks"})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"databricks"}, (*calls)[0].opts.SpecificSkills)
}

func TestSkillsInstallForwardsExperimental(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newSkillsInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--experimental"})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.True(t, (*calls)[0].opts.IncludeExperimental, "--experimental should be forwarded")
}

func TestSkillsInstallExecuteRejectsTwoArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newSkillsInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"a", "b"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts at most 1 arg")
}

func TestInstallRejectsPositionalArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"databricks"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestUpdateRejectsPositionalArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newUpdateCmd()
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
	cmd := newUninstallCmd()
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
	cmd := newListCmd()
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
	cmd := newVersionCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"databricks"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

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

// --- Scope flag tests ---

func TestInstallProjectFlag(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--project"})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, installer.ScopeProject, (*calls)[0].opts.Scope)
}

func TestInstallGlobalFlag(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--global"})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, installer.ScopeGlobal, (*calls)[0].opts.Scope)
}

func TestInstallGlobalAndProjectErrors(t *testing.T) {
	setupTestAgents(t)
	setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newInstallCmd()
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
	cmd := newInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, installer.ScopeGlobal, (*calls)[0].opts.Scope)
}

func TestInstallNoFlagInteractiveShowsScopePrompt(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)
	scopePromptCalled := setupScopeMock(t, installer.ScopeProject)

	// Also mock agent prompt since interactive mode triggers it.
	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })
	promptAgentSelection = func(_ context.Context, detected []*agents.Agent) ([]*agents.Agent, error) {
		return detected, nil
	}

	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()

	drain := func(r *bufio.Reader) {
		buf := make([]byte, 4096)
		for {
			_, err := r.Read(buf)
			if err != nil {
				return
			}
		}
	}
	go drain(test.Stdout)
	go drain(test.Stderr)

	cmd := newInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)

	assert.True(t, *scopePromptCalled, "scope prompt should be called in interactive mode")
	require.Len(t, *calls, 1)
	assert.Equal(t, installer.ScopeProject, (*calls)[0].opts.Scope)
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
