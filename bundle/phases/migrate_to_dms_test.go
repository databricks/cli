package phases

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationResources(t *testing.T) {
	direct := map[string]dstate.ResourceEntry{
		"resources.pipelines.second": {
			ID:    "pipeline-id",
			State: json.RawMessage(`{"name":"second"}`),
		},
		"resources.jobs.first": {
			ID:        "job-id",
			State:     json.RawMessage(`{"name":"first"}`),
			DependsOn: []deployplan.DependsOnEntry{{Node: "resources.pipelines.second", Label: "${resources.pipelines.second.id}"}},
		},
	}
	jobState, err := json.Marshal(dstate.RecordedState{
		State:     direct["resources.jobs.first"].State,
		DependsOn: direct["resources.jobs.first"].DependsOn,
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		existing []dms.Resource
		wantKeys []string
	}{
		{
			name:     "new migration includes every resource in stable order",
			wantKeys: []string{"resources.jobs.first", "resources.pipelines.second"},
		},
		{
			name: "retry skips a resource already recorded with identical state",
			existing: []dms.Resource{{
				Key:   "resources.jobs.first",
				ID:    "job-id",
				State: string(jobState),
			}},
			wantKeys: []string{"resources.pipelines.second"},
		},
		{
			name: "retry rewrites a stale recorded resource",
			existing: []dms.Resource{{
				Key:   "resources.jobs.first",
				ID:    "old-job-id",
				State: string(jobState),
			}},
			wantKeys: []string{"resources.jobs.first", "resources.pipelines.second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, staged, err := migrationResources(direct, tt.existing)
			require.NoError(t, err)
			assert.Equal(t, tt.wantKeys, migrationResourceKeys(resources))
			assert.Equal(t, tt.wantKeys, stagedResourceKeys(staged))
			for _, operation := range staged {
				assert.Equal(t, dms.OperationActionTypeMigrate, operation.ActionType)
			}
		})
	}
}

func migrationResourceKeys(resources []migrationResource) []string {
	keys := make([]string, len(resources))
	for i, resource := range resources {
		keys[i] = resource.key
	}
	return keys
}

func stagedResourceKeys(operations []dms.StagedOperation) []string {
	keys := make([]string, len(operations))
	for i, operation := range operations {
		keys[i] = operation.ResourceKey
	}
	return keys
}

func TestMigrationVersionID(t *testing.T) {
	tests := []struct {
		name          string
		directSerial  int
		lastDMS       string
		wantVersionID int
	}{
		{name: "direct state is newer", directSerial: 7, lastDMS: "3", wantVersionID: 8},
		{name: "DMS is newer after a partial retry", directSerial: 3, lastDMS: "7", wantVersionID: 8},
		{name: "new DMS deployment follows direct state", directSerial: 1, wantVersionID: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := migrationVersionID(tt.directSerial, tt.lastDMS)
			require.NoError(t, err)
			assert.Equal(t, tt.wantVersionID, got)
		})
	}
}
