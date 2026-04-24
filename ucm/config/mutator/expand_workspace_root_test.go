package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUcmWithRootAndUser(rootPath, userName string) *ucm.Ucm {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Workspace.RootPath = rootPath
	if userName != "" {
		u.CurrentUser = &config.User{User: &iam.User{UserName: userName}}
	}
	return u
}

func TestExpandWorkspaceRoot_ReplacesTildeWithUserHome(t *testing.T) {
	u := newUcmWithRootAndUser("~/databricks/ucm/demo/dev", "alice@example.com")

	diags := ucm.Apply(t.Context(), u, mutator.ExpandWorkspaceRoot())
	require.NoError(t, diags.Error())
	assert.Equal(t, "/Workspace/Users/alice@example.com/databricks/ucm/demo/dev", u.Config.Workspace.RootPath)
}

func TestExpandWorkspaceRoot_LeavesAbsolutePathUntouched(t *testing.T) {
	u := newUcmWithRootAndUser("/Workspace/Shared/custom", "alice@example.com")

	diags := ucm.Apply(t.Context(), u, mutator.ExpandWorkspaceRoot())
	require.NoError(t, diags.Error())
	assert.Equal(t, "/Workspace/Shared/custom", u.Config.Workspace.RootPath)
}

func TestExpandWorkspaceRoot_ErrorsWhenRootPathEmpty(t *testing.T) {
	u := newUcmWithRootAndUser("", "alice@example.com")

	diags := ucm.Apply(t.Context(), u, mutator.ExpandWorkspaceRoot())
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "workspace root not defined")
}

func TestExpandWorkspaceRoot_ErrorsWhenCurrentUserMissing(t *testing.T) {
	u := newUcmWithRootAndUser("~/databricks/ucm/demo/dev", "")

	diags := ucm.Apply(t.Context(), u, mutator.ExpandWorkspaceRoot())
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "current user not set")
}
