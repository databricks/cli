package aitools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	aitoolscmd "github.com/databricks/cli/aitools/cmd"
	"github.com/databricks/cli/aitools/lib/agents"
	"github.com/databricks/cli/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type installCall struct {
	agents []string
	opts   installer.InstallOptions
}

func setupInstallMock(t *testing.T) *[]installCall {
	t.Helper()
	orig := aitoolscmd.InstallSkillsForAgentsFn
	t.Cleanup(func() { aitoolscmd.InstallSkillsForAgentsFn = orig })

	var calls []installCall
	aitoolscmd.InstallSkillsForAgentsFn = func(_ context.Context, _ installer.ManifestSource, targetAgents []*agents.Agent, opts installer.InstallOptions) error {
		names := make([]string, len(targetAgents))
		for i, a := range targetAgents {
			names[i] = a.Name
		}
		calls = append(calls, installCall{agents: names, opts: opts})
		return nil
	}
	return &calls
}

func setupTestAgents(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))
	return tmp
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

func TestSkillsListDelegatesToListFn(t *testing.T) {
	orig := aitoolscmd.ListSkillsFn
	t.Cleanup(func() { aitoolscmd.ListSkillsFn = orig })

	called := false
	aitoolscmd.ListSkillsFn = func(cmd *cobra.Command, scope string) error {
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
