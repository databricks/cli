package lock

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanActionToOperationAction(t *testing.T) {
	tests := []struct {
		name     string
		action   deployplan.ActionType
		expected sdkbundle.OperationActionType
		wantErr  string
	}{
		{
			name:     "skip maps to empty (no DMS operation)",
			action:   deployplan.Skip,
			expected: "",
		},
		{
			name:     "create",
			action:   deployplan.Create,
			expected: sdkbundle.OperationActionTypeOperationActionTypeCreate,
		},
		{
			name:     "update",
			action:   deployplan.Update,
			expected: sdkbundle.OperationActionTypeOperationActionTypeUpdate,
		},
		{
			name:     "update_id",
			action:   deployplan.UpdateWithID,
			expected: sdkbundle.OperationActionTypeOperationActionTypeUpdateWithId,
		},
		{
			name:     "delete",
			action:   deployplan.Delete,
			expected: sdkbundle.OperationActionTypeOperationActionTypeDelete,
		},
		{
			name:     "recreate",
			action:   deployplan.Recreate,
			expected: sdkbundle.OperationActionTypeOperationActionTypeRecreate,
		},
		{
			name:     "resize",
			action:   deployplan.Resize,
			expected: sdkbundle.OperationActionTypeOperationActionTypeResize,
		},
		{
			name:    "undefined returns error",
			action:  deployplan.Undefined,
			wantErr: "unsupported operation action type",
		},
		{
			name:    "unknown returns error",
			action:  "some_garbage_value",
			wantErr: "unsupported operation action type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := planActionToOperationAction(tt.action)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
