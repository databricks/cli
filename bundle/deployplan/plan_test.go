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

func TestCountApplied(t *testing.T) {
	tests := []struct {
		name  string
		entry deployplan.PlanEntry
		want  deployplan.AppliedCounts
	}{
		{
			name:  "create applied",
			entry: deployplan.PlanEntry{Action: deployplan.Create, Attempted: true, Applied: true},
			want:  deployplan.AppliedCounts{ActionCounts: deployplan.ActionCounts{Create: 1}},
		},
		{
			name:  "create attempted but failed",
			entry: deployplan.PlanEntry{Action: deployplan.Create, Attempted: true},
			want:  deployplan.AppliedCounts{Failed: 1},
		},
		{
			name:  "create never reached is neither applied nor failed",
			entry: deployplan.PlanEntry{Action: deployplan.Create},
			want:  deployplan.AppliedCounts{},
		},
		{
			name:  "update applied counts as changed",
			entry: deployplan.PlanEntry{Action: deployplan.Update, Attempted: true, Applied: true},
			want:  deployplan.AppliedCounts{ActionCounts: deployplan.ActionCounts{Change: 1}},
		},
		{
			name:  "delete applied",
			entry: deployplan.PlanEntry{Action: deployplan.Delete, Attempted: true, Applied: true},
			want:  deployplan.AppliedCounts{ActionCounts: deployplan.ActionCounts{Delete: 1}},
		},
		{
			name:  "delete attempted but failed",
			entry: deployplan.PlanEntry{Action: deployplan.Delete, Attempted: true},
			want:  deployplan.AppliedCounts{Failed: 1},
		},
		{
			name:  "state-only delete is excluded",
			entry: deployplan.PlanEntry{Action: deployplan.Delete, StateOnly: true, Attempted: true},
			want:  deployplan.AppliedCounts{},
		},
		{
			name:  "recreate applied counts as both create and delete",
			entry: deployplan.PlanEntry{Action: deployplan.Recreate, Attempted: true, Applied: true},
			want:  deployplan.AppliedCounts{ActionCounts: deployplan.ActionCounts{Create: 1, Delete: 1}},
		},
		{
			name:  "recreate failed counts as a single failure",
			entry: deployplan.PlanEntry{Action: deployplan.Recreate, Attempted: true},
			want:  deployplan.AppliedCounts{Failed: 1},
		},
		{
			name:  "skip is unchanged regardless of attempt",
			entry: deployplan.PlanEntry{Action: deployplan.Skip},
			want:  deployplan.AppliedCounts{ActionCounts: deployplan.ActionCounts{Unchanged: 1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &deployplan.Plan{Plan: map[string]*deployplan.PlanEntry{"resources.jobs.a": &tt.entry}}
			assert.Equal(t, tt.want, p.CountApplied())
		})
	}
}
