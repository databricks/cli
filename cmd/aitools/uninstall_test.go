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
		{name: "legacy both flags together rejected", args: []string{"--project", "--global"}, wantErr: "cannot use --global and --project together"},
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
