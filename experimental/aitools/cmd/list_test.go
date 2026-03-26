package aitools

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCommandExists(t *testing.T) {
	cmd := newListCmd()
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
	cmd := newListCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestListCommandHasScopeFlags(t *testing.T) {
	cmd := newListCmd()
	f := cmd.Flags().Lookup("project")
	require.NotNil(t, f, "--project flag should exist")
	f = cmd.Flags().Lookup("global")
	require.NotNil(t, f, "--global flag should exist")
}

func TestSkillsListDelegatesToListFn(t *testing.T) {
	orig := listSkillsFn
	t.Cleanup(func() { listSkillsFn = orig })

	called := false
	listSkillsFn = func(cmd *cobra.Command, scope string) error {
		called = true
		return nil
	}

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newSkillsListCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	assert.True(t, called)
}
