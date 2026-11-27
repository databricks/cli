package deployplan_test

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasChange(t *testing.T) {
	tests := []struct {
		name    string
		changes deployplan.Changes
		path    string
		want    bool
	}{
		{
			name:    "nil changes",
			changes: nil,
			path:    "config",
			want:    false,
		},
		{
			name:    "actionable change matches prefix",
			changes: deployplan.Changes{"config.name": {Action: deployplan.Update}},
			path:    "config",
			want:    true,
		},
		{
			name:    "skip change is ignored",
			changes: deployplan.Changes{"config.traffic_config": {Action: deployplan.Skip}},
			path:    "config",
			want:    false,
		},
		{
			name: "skip alongside actionable change still reports",
			changes: deployplan.Changes{
				"config.traffic_config":  {Action: deployplan.Skip},
				"config.served_entities": {Action: deployplan.Update},
			},
			path: "config",
			want: true,
		},
		{
			name:    "no match for unrelated prefix",
			changes: deployplan.Changes{"tags.team": {Action: deployplan.Update}},
			path:    "config",
			want:    false,
		},
		{
			name:    "prefix respects path boundaries",
			changes: deployplan.Changes{"configuration": {Action: deployplan.Update}},
			path:    "config",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := structpath.ParsePath(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tt.changes.HasChange(path))
		})
	}
}

func TestHasOnlySingleReason(t *testing.T) {
	parentRenamed := &deployplan.ChangeDesc{Action: deployplan.UpdateWithID, Reason: deployplan.ReasonParentRenamed}
	tests := []struct {
		name    string
		changes deployplan.Changes
		want    bool
	}{
		{
			name:    "parent renamed",
			changes: deployplan.Changes{"schema_name": parentRenamed},
			want:    true,
		},
		{
			name: "skip change is ignored",
			changes: deployplan.Changes{
				"schema_name":      parentRenamed,
				"storage_location": {Action: deployplan.Skip, Reason: "immutable_backend_value"},
			},
			want: true,
		},
		{
			name: "own field also changes",
			changes: deployplan.Changes{
				"schema_name": parentRenamed,
				"comment":     {Action: deployplan.Update},
			},
			want: false,
		},
		{
			name:    "own rename",
			changes: deployplan.Changes{"name": {Action: deployplan.UpdateWithID, Reason: "id_changes"}},
			want:    false,
		},
		{
			name: "only skipped changes",
			changes: deployplan.Changes{
				"name": {Action: deployplan.Skip, Reason: deployplan.ReasonRemoteAlreadySet},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.changes.HasOnlySingleReason(deployplan.ReasonParentRenamed))
		})
	}
}

func TestWithParentIsADeleteThatSendsNoCall(t *testing.T) {
	gone := &deployplan.PlanEntry{Action: deployplan.Delete, Gone: true}
	withParent := &deployplan.PlanEntry{Action: deployplan.Delete, Gone: true, WithParent: "resources.schemas.s"}

	// A resource already gone is hidden and needs no call.
	assert.True(t, gone.IsStateOnly())
	assert.True(t, gone.SendsNoDelete())

	// One that goes with its parent's delete is shown and approved as a deletion, but no delete is
	// sent for it.
	assert.False(t, withParent.IsStateOnly())
	assert.True(t, withParent.SendsNoDelete())

	plan := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{"resources.volumes.v": withParent}}
	actions := plan.GetActions()
	assert.False(t, actions[0].IsStateOnly())
}
