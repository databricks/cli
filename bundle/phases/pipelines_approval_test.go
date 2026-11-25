package phases

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineDeletionCascades(t *testing.T) {
	// The value must be read from persisted state (what the delete uses), not config.
	b := &bundle.Bundle{}
	data := dstate.NewDatabase("lineage", 1)
	data.State["resources.pipelines.cascade_true"] = dstate.ResourceEntry{ID: "1", State: json.RawMessage(`{"cascade_on_destroy": true}`)}
	data.State["resources.pipelines.cascade_false"] = dstate.ResourceEntry{ID: "2", State: json.RawMessage(`{"cascade_on_destroy": false}`)}
	data.State["resources.pipelines.cascade_unset"] = dstate.ResourceEntry{ID: "3", State: json.RawMessage(`{}`)}
	data.State["resources.pipelines.cascade_empty"] = dstate.ResourceEntry{ID: "4"}
	b.DeploymentBundle.StateDB.OpenWithData(filepath.Join(t.TempDir(), "resources.json"), data)

	tests := []struct {
		name        string
		bundle      *bundle.Bundle
		resourceKey string
		want        bool
	}{
		{
			name:        "cascade_on_destroy explicitly true",
			bundle:      b,
			resourceKey: "resources.pipelines.cascade_true",
			want:        true,
		},
		{
			name:        "cascade_on_destroy explicitly false",
			bundle:      b,
			resourceKey: "resources.pipelines.cascade_false",
			want:        false,
		},
		{
			name:        "cascade_on_destroy unset defaults to cascade",
			bundle:      b,
			resourceKey: "resources.pipelines.cascade_unset",
			want:        true,
		},
		{
			name:        "empty persisted state defaults to cascade",
			bundle:      b,
			resourceKey: "resources.pipelines.cascade_empty",
			want:        true,
		},
		{
			name:        "missing state entry defaults to cascade",
			bundle:      b,
			resourceKey: "resources.pipelines.does_not_exist",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := deployplan.Action{
				ResourceKey: tt.resourceKey,
				ActionType:  deployplan.Delete,
			}
			cascade, err := pipelineDeletionCascades(tt.bundle, action)
			require.NoError(t, err)
			assert.Equal(t, tt.want, cascade)
		})
	}
}
