package lock

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/tmpdms"
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
