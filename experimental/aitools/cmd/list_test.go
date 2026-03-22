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
	listSkillsFn = func(cmd *cobra.Command) error {
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

func TestListCommandHasSkillsFlag(t *testing.T) {
	cmd := newListCmd()
	f := cmd.Flags().Lookup("skills")
	require.NotNil(t, f, "--skills flag should exist")
	assert.Equal(t, "false", f.DefValue)
}

func TestSkillsListDelegatesToListFn(t *testing.T) {
	orig := listSkillsFn
	t.Cleanup(func() { listSkillsFn = orig })

	called := false
	listSkillsFn = func(cmd *cobra.Command) error {
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
