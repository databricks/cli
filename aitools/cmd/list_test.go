package aitools

import (
	"testing"

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
	orig := ListSkillsFn
	t.Cleanup(func() { ListSkillsFn = orig })

	called := false
	ListSkillsFn = func(cmd *cobra.Command, scope string) error {
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
	require.NotNil(t, f, "--project flag should exist")
	f = cmd.Flags().Lookup("global")
	require.NotNil(t, f, "--global flag should exist")
}
