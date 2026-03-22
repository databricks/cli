package aitools

import (
	"context"
	"bufio"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
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

func TestInstallCommandsDelegateToSkillsInstall(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	tests := []struct {
		name       string
		newCmd     func() *cobra.Command
		args       []string
		wantAgents int
		wantSkills []string
	}{
		{
			name:       "skills install installs all skills for all agents",
			newCmd:     newSkillsInstallCmd,
			wantAgents: 2,
		},
		{
			name:       "skills install forwards skill name",
			newCmd:     newSkillsInstallCmd,
			args:       []string{"bundle/review"},
			wantAgents: 2,
			wantSkills: []string{"bundle/review"},
		},
		{
			name:       "top level install installs all skills",
			newCmd:     newInstallCmd,
			wantAgents: 2,
		},
		{
			name:       "top level install forwards skill name",
			newCmd:     newInstallCmd,
			args:       []string{"bundle/review"},
			wantAgents: 2,
			wantSkills: []string{"bundle/review"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*calls = nil

			ctx := cmdio.MockDiscard(t.Context())
			cmd := tt.newCmd()
			cmd.SetContext(ctx)

			err := cmd.RunE(cmd, tt.args)
			require.NoError(t, err)

			require.Len(t, *calls, 1)
			assert.Len(t, (*calls)[0].agents, tt.wantAgents)
			assert.Equal(t, tt.wantSkills, (*calls)[0].opts.SpecificSkills)
		})
	}
}

func TestRunSkillsInstallInteractivePrompt(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })

	promptCalled := false
	promptAgentSelection = func(_ context.Context, detected []*agents.Agent) ([]*agents.Agent, error) {
		promptCalled = true
		// Return only the first agent.
		return detected[:1], nil
	}

	// Use SetupTest with PromptSupported=true to simulate interactive terminal.
	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()

	// Drain both pipes in background to prevent blocking.
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

	err := runSkillsInstall(ctx, nil)
	require.NoError(t, err)

	assert.True(t, promptCalled, "prompt should be called when 2+ agents detected and interactive")
	require.Len(t, *calls, 1)
	assert.Len(t, (*calls)[0].agents, 1, "only the selected agent should be passed")
}

func TestRunSkillsInstallNonInteractiveUsesAllAgents(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })

	promptCalled := false
	promptAgentSelection = func(_ context.Context, detected []*agents.Agent) ([]*agents.Agent, error) {
		promptCalled = true
		return detected, nil
	}

	// MockDiscard gives a non-interactive context.
	ctx := cmdio.MockDiscard(t.Context())

	err := runSkillsInstall(ctx, nil)
	require.NoError(t, err)

	assert.False(t, promptCalled, "prompt should not be called in non-interactive mode")
	require.Len(t, *calls, 1)
	assert.Len(t, (*calls)[0].agents, 2, "all detected agents should be used")
}

func TestRunSkillsInstallNoAgents(t *testing.T) {
	// Set HOME to empty dir so no agents are detected.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	calls := setupInstallMock(t)
	ctx := cmdio.MockDiscard(t.Context())

	err := runSkillsInstall(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, *calls, "install should not be called when no agents detected")
}
