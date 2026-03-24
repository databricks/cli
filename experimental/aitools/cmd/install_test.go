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
