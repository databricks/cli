package dmsmigration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/deploy/dmsmigration"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareMigration(t *testing.T) {
	tests := []struct {
		name         string
		serial       int
		lastVersion  int
		registration int
	}{
		{
			name:         "first registration",
			serial:       2,
			registration: 3,
		},
		{
			name:         "service ahead",
			serial:       2,
			lastVersion:  5,
			registration: 6,
		},
		{
			name:         "file ahead",
			serial:       8,
			lastVersion:  3,
			registration: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statePath := filepath.Join(t.TempDir(), "resources.json")
			original := dstate.NewDatabase("lineage", tt.serial)
			original.Features = map[string]struct{}{}
			original.State["resources.jobs.foo"] = dstate.ResourceEntry{ID: "original-id", State: json.RawMessage(`{"name":"original"}`)}
			var db dstate.DeploymentState
			db.OpenWithData(statePath, original)
			require.NoError(t, db.Persist())
			before, err := os.ReadFile(statePath)
			require.NoError(t, err)

			migration := dmsmigration.Prepare(&db, "123", tt.lastVersion)

			assert.True(t, db.IsDeploymentMetadataService())
			assert.Equal(t, tt.registration, db.GetSerial())
			assert.Equal(t, "123", db.DeploymentID)
			assert.Contains(t, db.StateFeatures(), dstate.FeatureDeploymentHistory)
			assert.NotContains(t, migration.Snapshot.Features, dstate.FeatureDeploymentHistory)
			assert.Equal(t, tt.serial, migration.Snapshot.Serial)
			assert.Equal(t, original.State, migration.Snapshot.State)
			assert.Equal(t, "original-id", db.GetResourceID("resources.jobs.foo"))
			after, err := os.ReadFile(statePath)
			require.NoError(t, err)
			assert.Equal(t, before, after)
			assert.NoFileExists(t, statePath+".wal")
		})
	}
}
