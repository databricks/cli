package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefineDefaultWorkspacePaths(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/",
			},
		},
	}
	diags := ucm.Apply(t.Context(), u, mutator.DefineDefaultWorkspacePaths())
	require.NoError(t, diags.Error())
	assert.Equal(t, "/state", u.Config.Workspace.StatePath)
}

func TestDefineDefaultWorkspacePathsAlreadySet(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:  "/",
				StatePath: "/foo/bar",
			},
		},
	}
	diags := ucm.Apply(t.Context(), u, mutator.DefineDefaultWorkspacePaths())
	require.NoError(t, diags.Error())
	assert.Equal(t, "/foo/bar", u.Config.Workspace.StatePath)
}

func TestDefineDefaultWorkspacePathsErrorsWhenRootMissing(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}

	diags := ucm.Apply(t.Context(), u, mutator.DefineDefaultWorkspacePaths())
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "workspace root not defined")
}
