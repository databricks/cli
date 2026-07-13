package aitools

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLegacySkillsInstallDelegatesToInstall(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newLegacySkillsInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Len(t, (*calls)[0].agents, 2)
}

func TestLegacySkillsInstallForwardsSkillName(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newLegacySkillsInstallCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"databricks"})
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"databricks"}, (*calls)[0].opts.SpecificSkills)
}

func TestLegacySkillsInstallExecuteNoArgs(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newLegacySkillsInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Len(t, (*calls)[0].agents, 2)
	assert.Nil(t, (*calls)[0].opts.SpecificSkills)
}

func TestLegacySkillsInstallExecuteWithSkillName(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newLegacySkillsInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"databricks"})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, []string{"databricks"}, (*calls)[0].opts.SpecificSkills)
}

func TestLegacySkillsInstallForwardsExperimental(t *testing.T) {
	setupTestAgents(t)
	calls := setupInstallMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newLegacySkillsInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--experimental"})

	err := cmd.Execute()
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.True(t, (*calls)[0].opts.IncludeExperimental, "--experimental should be forwarded")
}

func TestLegacySkillsInstallExecuteRejectsTwoArgs(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := newLegacySkillsInstallCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"a", "b"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts at most 1 arg")
}

func TestLegacySkillsListDelegatesToListFn(t *testing.T) {
	orig := listSkillsFn
	t.Cleanup(func() { listSkillsFn = orig })

	called := false
	listSkillsFn = func(cmd *cobra.Command, scope string) error {
		called = true
		return nil
	}

	ctx := cmdio.MockDiscard(t.Context())
	cmd := newLegacySkillsListCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	assert.True(t, called)
}
