package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitFlagsPopulateResolver(t *testing.T) {
	cmd := newInitCommand()
	err := cmd.Flags().Parse([]string{
		"--tag", "my-tag",
		"--template-dir", "inner",
		"--output-dir", "out",
		"--config-file", "config.json",
	})
	require.NoError(t, err)

	r, err := resolverFromInitFlags(cmd, []string{"some-template"})
	require.NoError(t, err)

	assert.Equal(t, "my-tag", r.Tag)
	assert.Empty(t, r.Branch)
	assert.Equal(t, "inner", r.TemplateDir)
	assert.Equal(t, "out", r.OutputDir)
	assert.Equal(t, "config.json", r.ConfigFile)
	assert.Equal(t, "some-template", r.TemplatePathOrUrl)
}

func TestInitBranchFlagBindsToBranchField(t *testing.T) {
	cmd := newInitCommand()
	err := cmd.Flags().Parse([]string{"--branch", "my-branch"})
	require.NoError(t, err)

	r, err := resolverFromInitFlags(cmd, nil)
	require.NoError(t, err)

	assert.Equal(t, "my-branch", r.Branch)
	assert.Empty(t, r.Tag)
}

func TestInitRejectsBothTagAndBranch(t *testing.T) {
	cmd := newInitCommand()
	err := cmd.Flags().Parse([]string{"--tag", "t", "--branch", "b"})
	require.NoError(t, err)

	_, err = resolverFromInitFlags(cmd, nil)
	assert.ErrorContains(t, err, "only one of --tag or --branch")
}
