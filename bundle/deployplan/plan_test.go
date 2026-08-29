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
