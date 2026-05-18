package aitools

import (
	"testing"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCommandExists(t *testing.T) {
	cmd := NewListCmd()
	assert.Equal(t, "list", cmd.Use)
}

func TestListCommandCallsListFn(t *testing.T) {
	orig := listSkillsFn
	t.Cleanup(func() { listSkillsFn = orig })

	called := false
	listSkillsFn = func(cmd *cobra.Command, scope string) error {
		called = true
		return nil
	}

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewListCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestListCommandHasScopeFlags(t *testing.T) {
	cmd := NewListCmd()
	f := cmd.Flags().Lookup("project")
	require.NotNil(t, f, "--project flag should exist (deprecated alias)")
	assert.NotEmpty(t, f.Deprecated, "--project should be marked deprecated")
	f = cmd.Flags().Lookup("global")
	require.NotNil(t, f, "--global flag should exist (deprecated alias)")
	assert.NotEmpty(t, f.Deprecated, "--global should be marked deprecated")
	f = cmd.Flags().Lookup("scope")
	require.NotNil(t, f, "--scope flag should exist")
}

func TestListScopeFlag(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantScope string
		wantErr   string
	}{
		{name: "scope project", args: []string{"--scope", "project"}, wantScope: installer.ScopeProject},
		{name: "scope global", args: []string{"--scope", "global"}, wantScope: installer.ScopeGlobal},
		{name: "scope both shows both", args: []string{"--scope", "both"}, wantScope: ""},
		{name: "scope invalid", args: []string{"--scope", "all"}, wantErr: `invalid --scope "all"`},
		{name: "legacy both flags shows both", args: []string{"--project", "--global"}, wantScope: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := listSkillsFn
			t.Cleanup(func() { listSkillsFn = orig })

			var gotScope string
			called := false
			listSkillsFn = func(_ *cobra.Command, scope string) error {
				called = true
				gotScope = scope
				return nil
			}

			ctx := cmdio.MockDiscard(t.Context())
			cmd := NewListCmd()
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
			assert.True(t, called)
			assert.Equal(t, tt.wantScope, gotScope)
		})
	}
}
