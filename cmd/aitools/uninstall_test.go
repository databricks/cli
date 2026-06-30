package aitools

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUninstallMock(t *testing.T) *[]installer.UninstallOptions {
	t.Helper()
	orig := uninstallSkillsFn
	t.Cleanup(func() { uninstallSkillsFn = orig })

	var calls []installer.UninstallOptions
	uninstallSkillsFn = func(_ context.Context, opts installer.UninstallOptions) error {
		calls = append(calls, opts)
		return nil
	}
	return &calls
}

func TestUninstallScopeFlag(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantScope string
		wantErr   string
	}{
		// scope=project requires installed project state and is covered via TestParseScopeFlag
		// and TestResolveScopeForUninstallProjectFlagWithState. Here we cover the no-state paths
		// and the failure modes specific to the Cobra wiring.
		{name: "scope global", args: []string{"--scope", "global"}, wantScope: installer.ScopeGlobal},
		{name: "scope both rejected", args: []string{"--scope", "both"}, wantErr: "--scope=both is not supported"},
		{name: "scope invalid value", args: []string{"--scope", "all"}, wantErr: `invalid --scope "all"`},
		{name: "scope conflicts with legacy project", args: []string{"--scope", "global", "--project"}, wantErr: "cannot use --scope with --project or --global"},
		{name: "legacy both flags together rejected", args: []string{"--project", "--global"}, wantErr: "cannot uninstall both scopes at once"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestAgents(t)
			calls := setupUninstallMock(t)

			ctx := cmdio.MockDiscard(t.Context())
			cmd := NewUninstallCmd()
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
			assert.Equal(t, tt.wantScope, (*calls)[0].Scope)
		})
	}
}

func TestUninstallAgentsFlag(t *testing.T) {
	setupTestAgents(t)
	calls := setupUninstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewUninstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--scope", "global", "--agents", "cursor,claude-code", "--skills", "databricks-sql"})

	require.NoError(t, cmd.Execute())
	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"cursor", "claude-code"}, (*calls)[0].Agents)
	assert.Equal(t, []string{"databricks-sql"}, (*calls)[0].Skills)
}

func TestUninstallUnknownAgentErrors(t *testing.T) {
	setupTestAgents(t)
	setupUninstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewUninstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--scope", "global", "--agents", "invalid-agent"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent")
}

func TestUninstallConfirmMessage(t *testing.T) {
	// Nothing recorded: no prompt (installer surfaces its own guidance).
	_, ask := uninstallConfirmMessage(nil, installer.UninstallOptions{Scope: installer.ScopeGlobal})
	assert.False(t, ask)

	_, askEmpty := uninstallConfirmMessage(&installer.InstallState{}, installer.UninstallOptions{Scope: installer.ScopeGlobal})
	assert.False(t, askEmpty)

	st := &installer.InstallState{
		Skills:  map[string]string{"a": "1", "b": "2"},
		Plugins: map[string]installer.PluginRecord{"claude-code": {}},
	}

	msg, ask := uninstallConfirmMessage(st, installer.UninstallOptions{Scope: installer.ScopeGlobal})
	require.True(t, ask)
	assert.Contains(t, msg, "2 skills")
	assert.Contains(t, msg, "the databricks plugin from 1 agent")
	assert.Contains(t, msg, "(global scope)")

	// --skills filter names the requested skills.
	filtered := installer.UninstallOptions{Scope: installer.ScopeProject}
	filtered.Skills = []string{"alpha"}
	msg2, ask2 := uninstallConfirmMessage(st, filtered)
	require.True(t, ask2)
	assert.Contains(t, msg2, "skill alpha")
	assert.Contains(t, msg2, "from all agents")
	assert.Contains(t, msg2, "(project scope)")

	targeted := installer.UninstallOptions{Scope: installer.ScopeGlobal, Agents: []string{"cursor"}}
	targeted.Skills = []string{"alpha"}
	msg3, ask3 := uninstallConfirmMessage(st, targeted)
	require.True(t, ask3)
	assert.Contains(t, msg3, "from cursor")
}
