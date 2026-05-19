package aitools

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUpdateMock(t *testing.T) *[]installer.UpdateOptions {
	t.Helper()
	orig := updateSkillsFn
	t.Cleanup(func() { updateSkillsFn = orig })

	var calls []installer.UpdateOptions
	updateSkillsFn = func(_ context.Context, _ installer.ManifestSource, _ []*agents.Agent, opts installer.UpdateOptions) (*installer.UpdateResult, error) {
		calls = append(calls, opts)
		return &installer.UpdateResult{}, nil
	}
	return &calls
}

func TestUpdateScopeFlag(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantScopes []string
		wantErr    string
	}{
		// scope=project requires installed project state and is covered via TestParseScopeFlag
		// (cmd/aitools/scope_test.go) and TestResolveScopeForUpdateProjectFlagWithState. Here
		// we cover the no-state paths and the failure modes specific to the Cobra wiring.
		{name: "scope global", args: []string{"--scope", "global"}, wantScopes: []string{installer.ScopeGlobal}},
		{name: "scope both with no installs falls through to global", args: []string{"--scope", "both"}, wantScopes: []string{installer.ScopeGlobal}},
		{name: "scope invalid value", args: []string{"--scope", "all"}, wantErr: `invalid --scope "all"`},
		{name: "scope conflicts with legacy project", args: []string{"--scope", "global", "--project"}, wantErr: "cannot use --scope with --project or --global"},
		{name: "legacy both flags together rejected", args: []string{"--project", "--global"}, wantErr: "cannot use --global and --project together"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestAgents(t)
			calls := setupUpdateMock(t)

			ctx := cmdio.MockDiscard(t.Context())
			cmd := NewUpdateCmd()
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
			require.Len(t, *calls, len(tt.wantScopes))
			for i, scope := range tt.wantScopes {
				assert.Equal(t, scope, (*calls)[i].Scope)
			}
		})
	}
}
