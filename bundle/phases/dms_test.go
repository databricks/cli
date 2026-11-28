package phases

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentMetadataForBindWithoutDeployedFiles(t *testing.T) {
	for _, mode := range []config.SyncMode{config.SyncModeImmutable, config.SyncModeInPlace} {
		t.Run(string(mode), func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Bundle: config.Bundle{Name: "test", Target: "prod", Mode: config.Production},
					Workspace: config.Workspace{
						RootPath: "/Workspace/bundle",
						FilePath: "${resources.snapshots.files.full_path}/files",
						SyncMode: mode,
					},
				},
				SyncRootPath: "/Workspace/source",
			}

			assert.Equal(t, bundledeployments.Deployment{
				DisplayName:    "test",
				TargetName:     "prod",
				DeploymentMode: bundledeployments.DeploymentModeDeploymentModeProduction,
				WorkspaceInfo:  &bundledeployments.WorkspaceInfo{RootPath: "/Workspace/bundle"},
			}, deploymentMetadataForBind(b).Deployment())
		})
	}
}

func TestDeploymentMetadataForBindGitFolder(t *testing.T) {
	// The native path of a worktree root never has the /Workspace prefix on Windows.
	if runtime.GOOS == "windows" {
		t.Skip("this test is not applicable on Windows")
	}
	gitFolderPath := "/Workspace/Users/test.user@databricks.com/git_folder"
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "test",
				Git:  config.Git{BundleRootPath: "bundle"},
			},
			Workspace: config.Workspace{RootPath: "/Workspace/bundle"},
		},
		WorktreeRoot: vfs.MustNew(gitFolderPath),
	}

	assert.Equal(t, &bundledeployments.WorkspaceInfo{
		RootPath:       "/Workspace/bundle",
		GitFolderPath:  gitFolderPath,
		BundleRootPath: "bundle",
	}, deploymentMetadataForBind(b).Deployment().WorkspaceInfo)
}

func TestStagedOperationsCoversEveryTouchedResource(t *testing.T) {
	// The version fixes its operation set, so anything the apply will write has to appear
	// here. Keys go out in the service's form, without the CLI's "resources." prefix.
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.foo":       {Action: deployplan.Create},
		"resources.pipelines.bar":  {Action: deployplan.Recreate},
		"resources.schemas.baz":    {Action: deployplan.Delete},
		"resources.clusters.small": {Action: deployplan.Resize},
	}}

	staged, err := stagedOperations(plan)
	require.NoError(t, err)

	assert.ElementsMatch(t, []bundledeployments.StagedOperation{
		{ResourceKey: "jobs.foo", ActionType: bundledeployments.OperationActionTypeOperationActionTypeCreate},
		{ResourceKey: "pipelines.bar", ActionType: bundledeployments.OperationActionTypeOperationActionTypeRecreate},
		{ResourceKey: "schemas.baz", ActionType: bundledeployments.OperationActionTypeOperationActionTypeDelete},
		{ResourceKey: "clusters.small", ActionType: bundledeployments.OperationActionTypeOperationActionTypeResize},
	}, staged)
}

func TestStagedOperationsLeavesOutUntouchedResources(t *testing.T) {
	// A skipped resource is never applied, so staging it would leave an operation pending for
	// the life of the version. Undefined is not a real action either. The one skip that is
	// applied, as a rewrite of its state entry for a changed lifecycle.prevent_destroy, is
	// staged as an update so the service's copy of the state follows.
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.touched":     {Action: deployplan.Update},
		"resources.jobs.unchanged":   {Action: deployplan.Skip},
		"resources.jobs.reprotected": {Action: deployplan.Skip, PreventDestroyChanged: true},
		"resources.jobs.unknown":     {Action: deployplan.Undefined},
	}}

	staged, err := stagedOperations(plan)
	require.NoError(t, err)

	assert.Equal(t, []bundledeployments.StagedOperation{
		{ResourceKey: "jobs.reprotected", ActionType: bundledeployments.OperationActionTypeOperationActionTypeUpdate},
		{ResourceKey: "jobs.touched", ActionType: bundledeployments.OperationActionTypeOperationActionTypeUpdate},
	}, staged)
}

func TestStagedOperationsEmptyPlan(t *testing.T) {
	staged, err := stagedOperations(&deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{}})
	require.NoError(t, err)
	assert.Empty(t, staged)
}

func TestActionToSDK(t *testing.T) {
	cases := []struct {
		action deployplan.ActionType
		want   bundledeployments.OperationActionType
	}{
		{deployplan.Create, bundledeployments.OperationActionTypeOperationActionTypeCreate},
		{deployplan.Update, bundledeployments.OperationActionTypeOperationActionTypeUpdate},
		{deployplan.UpdateWithID, bundledeployments.OperationActionTypeOperationActionTypeUpdateWithId},
		{deployplan.Recreate, bundledeployments.OperationActionTypeOperationActionTypeRecreate},
		{deployplan.Resize, bundledeployments.OperationActionTypeOperationActionTypeResize},
		{deployplan.Delete, bundledeployments.OperationActionTypeOperationActionTypeDelete},
	}
	for _, c := range cases {
		got, err := actionToSDK(c.action)
		require.NoError(t, err)
		assert.Equal(t, c.want, got)
	}

	// Nothing is applied for these, so stagedOperations leaves them out rather than
	// mapping them; the guard is here in case a new action type arrives without a mapping.
	_, err := actionToSDK(deployplan.Skip)
	assert.Error(t, err)
	_, err = actionToSDK(deployplan.Undefined)
	assert.Error(t, err)
}
