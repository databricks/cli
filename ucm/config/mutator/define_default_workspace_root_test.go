package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUcmWithNameAndTarget(name, target string) *ucm.Ucm {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Ucm.Name = name
	u.Config.Ucm.Target = target
	return u
}

func TestDefineDefaultWorkspaceRoot_SetsDefaultWhenEmpty(t *testing.T) {
	u := newUcmWithNameAndTarget("demo", "dev")

	diags := ucm.Apply(t.Context(), u, mutator.DefineDefaultWorkspaceRoot())
	require.NoError(t, diags.Error())
	assert.Equal(t, "~/databricks/ucm/demo/dev", u.Config.Workspace.RootPath)
}

func TestDefineDefaultWorkspaceRoot_PreservesUserSetValue(t *testing.T) {
	u := newUcmWithNameAndTarget("demo", "dev")
	u.Config.Workspace.RootPath = "/Workspace/Shared/custom"

	diags := ucm.Apply(t.Context(), u, mutator.DefineDefaultWorkspaceRoot())
	require.NoError(t, diags.Error())
	assert.Equal(t, "/Workspace/Shared/custom", u.Config.Workspace.RootPath)
}

func TestDefineDefaultWorkspaceRoot_ErrorsWhenNameMissing(t *testing.T) {
	u := newUcmWithNameAndTarget("", "dev")

	diags := ucm.Apply(t.Context(), u, mutator.DefineDefaultWorkspaceRoot())
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "ucm.name not defined")
}

func TestDefineDefaultWorkspaceRoot_ErrorsWhenTargetMissing(t *testing.T) {
	u := newUcmWithNameAndTarget("demo", "")

	diags := ucm.Apply(t.Context(), u, mutator.DefineDefaultWorkspaceRoot())
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "target not selected")
}
