package lock

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/tmpdms"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanActionToOperationAction(t *testing.T) {
	tests := []struct {
		action   deployplan.ActionType
		expected tmpdms.OperationActionType
	}{
		{deployplan.Skip, ""},
		{deployplan.Create, tmpdms.OperationActionTypeCreate},
		{deployplan.Update, tmpdms.OperationActionTypeUpdate},
		{deployplan.UpdateWithID, tmpdms.OperationActionTypeUpdateWithID},
		{deployplan.Delete, tmpdms.OperationActionTypeDelete},
		{deployplan.Recreate, tmpdms.OperationActionTypeRecreate},
		{deployplan.Resize, tmpdms.OperationActionTypeResize},
		{"unknown_action", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			result, err := planActionToOperationAction(tt.action)
			if tt.action == "unknown_action" {
				assert.ErrorContains(t, err, "unsupported operation action type")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGoalToVersionType(t *testing.T) {
	vt, ok := goalToVersionType(GoalDeploy)
	assert.True(t, ok)
	assert.Equal(t, tmpdms.VersionTypeDeploy, vt)

	vt, ok = goalToVersionType(GoalDestroy)
	assert.True(t, ok)
	assert.Equal(t, tmpdms.VersionTypeDestroy, vt)

	_, ok = goalToVersionType(GoalBind)
	assert.False(t, ok)

	_, ok = goalToVersionType(GoalUnbind)
	assert.False(t, ok)
}

func TestDeploymentMode(t *testing.T) {
	tests := []struct {
		mode     config.Mode
		expected tmpdms.DeploymentMode
	}{
		{config.Development, tmpdms.DeploymentModeDevelopment},
		{config.Production, tmpdms.DeploymentModeProduction},
		{"", ""},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.expected, deploymentMode(tt.mode))
		})
	}
}

func TestWorkspaceInfo(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Workspace/Users/me@databricks.com/.bundle/my-bundle/prod",
				FilePath: "/Workspace/Users/me@databricks.com/.bundle/my-bundle/prod/files",
			},
		},
	}

	info := workspaceInfo(b)
	assert.Equal(t, "/Workspace/Users/me@databricks.com/.bundle/my-bundle/prod", info.RootPath)
	assert.Equal(t, "/Workspace/Users/me@databricks.com/.bundle/my-bundle/prod/files", info.FilePath)
	assert.False(t, info.SourceLinked)
	assert.Empty(t, info.GitFolderPath)
	assert.Empty(t, info.BundleRootPath)
}

func TestWorkspaceInfoSourceLinked(t *testing.T) {
	enabled := true
	syncRootPath := "/Workspace/Users/me@databricks.com/source"
	b := &bundle.Bundle{
		SyncRootPath: syncRootPath,
		Config: config.Root{
			Presets: config.Presets{
				SourceLinkedDeployment: &enabled,
			},
			Workspace: config.Workspace{
				RootPath: "/Workspace/Users/me@databricks.com/.bundle/my-bundle/prod",
				FilePath: "/Workspace/Users/me@databricks.com/.bundle/my-bundle/prod/files",
			},
		},
	}

	info := workspaceInfo(b)
	// In source-linked deployments the sync root is the effective file path.
	assert.Equal(t, syncRootPath, info.FilePath)
	assert.True(t, info.SourceLinked)
}

func TestWorkspaceInfoGitFolderPath(t *testing.T) {
	// The native path of the worktree root on Windows will never match the
	// /Workspace prefix, so GitFolderPath is never set there.
	if runtime.GOOS == "windows" {
		t.Skip("this test is not applicable on Windows")
	}
	gitFolderPath := "/Workspace/Users/me@databricks.com/git_folder"
	b := &bundle.Bundle{
		WorktreeRoot: vfs.MustNew(gitFolderPath),
	}
	b.Config.Bundle.Git.BundleRootPath = "subdir/bundle"

	info := workspaceInfo(b)
	assert.Equal(t, gitFolderPath, info.GitFolderPath)
	assert.Equal(t, "subdir/bundle", info.BundleRootPath)
}
