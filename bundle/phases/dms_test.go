package phases

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	assert.ElementsMatch(t, []dms.StagedOperation{
		{ResourceKey: "jobs.foo", ActionType: bundledeployments.OperationActionTypeOperationActionTypeCreate},
		{ResourceKey: "pipelines.bar", ActionType: bundledeployments.OperationActionTypeOperationActionTypeRecreate},
		{ResourceKey: "schemas.baz", ActionType: bundledeployments.OperationActionTypeOperationActionTypeDelete},
		{ResourceKey: "clusters.small", ActionType: bundledeployments.OperationActionTypeOperationActionTypeResize},
	}, staged)
}

func TestStagedOperationsLeavesOutUntouchedResources(t *testing.T) {
	// A skipped resource is never applied, so staging it would leave an operation pending for
	// the life of the version. Undefined is not a real action either.
	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{
		"resources.jobs.touched":   {Action: deployplan.Update},
		"resources.jobs.unchanged": {Action: deployplan.Skip},
		"resources.jobs.unknown":   {Action: deployplan.Undefined},
	}}

	staged, err := stagedOperations(plan)
	require.NoError(t, err)

	assert.Equal(t, []dms.StagedOperation{
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
