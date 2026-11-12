package deployplan_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavedPlanPreservesBindPhase(t *testing.T) {
	plan := deployplan.NewPlanDirect()
	plan.Bind = map[string]*deployplan.BindEntry{
		"resources.jobs.job":               {ID: "123", State: json.RawMessage(`{"name":"existing"}`)},
		"resources.experiments.experiment": {ID: "456", State: json.RawMessage(`{"name":"unchanged"}`)},
	}
	plan.Plan = map[string]*deployplan.PlanEntry{
		"resources.jobs.job":               {ID: "123", Action: deployplan.Update, PreventIDChange: true},
		"resources.experiments.experiment": {ID: "456", Action: deployplan.Skip, PreventIDChange: true},
	}
	encoded, err := json.Marshal(plan)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "plan.json")
	require.NoError(t, os.WriteFile(path, encoded, 0o600))
	loaded, err := deployplan.LoadPlanFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, plan.Bind, loaded.Bind)
	assert.Equal(t, plan.Plan, loaded.Plan)
	assert.Equal(t, deployplan.ActionCounts{Bind: 2, Change: 1, Unchanged: 1}, loaded.CountActions())
	require.NoError(t, loaded.FilterToSelected(t.Context(), []string{"jobs.job"}))
	assert.Len(t, loaded.Bind, 1)
	assert.Contains(t, loaded.Bind, "resources.jobs.job")
	assert.Equal(t, deployplan.ActionCounts{Bind: 1, Change: 1}, loaded.CountActions())
}

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
