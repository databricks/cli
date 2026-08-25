package dstate

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyDMSState(t *testing.T) {
	// The service stores state as an opaque string, so the envelope the write path sent
	// arrives verbatim.
	envelope := `{"state":{"name":"foo"},"depends_on":[{"node":"resources.pipelines.bar","label":"${resources.pipelines.bar.id}"}]}`

	// What the file loaded, for the cases that check it is replaced or survives.
	fileState := map[string]ResourceEntry{
		"resources.jobs.bar": {ID: "file-id", State: json.RawMessage(`{"name":"from-file"}`)},
	}

	tests := []struct {
		name     string
		existing map[string]ResourceEntry
		recorded []dms.Resource
		want     map[string]ResourceEntry
		wantErr  string
	}{
		{
			name: "the envelope is unwrapped, depends_on and all",
			recorded: []dms.Resource{
				{Key: "resources.jobs.foo", ID: "123", State: envelope},
				{Key: "resources.pipelines.bar", ID: "456"},
			},
			// depends_on comes back from the envelope, so a bundle whose local state was
			// wiped still has the edges needed for delete ordering.
			want: map[string]ResourceEntry{
				"resources.jobs.foo": {
					ID:        "123",
					State:     json.RawMessage(`{"name":"foo"}`),
					DependsOn: []deployplan.DependsOnEntry{{Node: "resources.pipelines.bar", Label: "${resources.pipelines.bar.id}"}},
				},
				"resources.pipelines.bar": {ID: "456"},
			},
		},
		{
			name:     "what DMS holds replaces what the file loaded",
			existing: fileState,
			recorded: []dms.Resource{{Key: "resources.jobs.foo", ID: "dms-id", State: `{"state":{"name":"from-dms"}}`}},
			want:     map[string]ResourceEntry{"resources.jobs.foo": {ID: "dms-id", State: json.RawMessage(`{"name":"from-dms"}`)}},
		},
		{
			// Nil is what a list holding nothing returns, and it means a successful deploy of
			// nothing rather than missing data.
			name:     "nothing recorded empties the state",
			existing: fileState,
			want:     map[string]ResourceEntry{},
		},
		{
			// The good resource comes first, so the error lands mid-way: what the file loaded
			// has to survive whole rather than end up half replaced.
			name:     "a malformed envelope leaves the state as it was",
			existing: fileState,
			recorded: []dms.Resource{
				{Key: "resources.jobs.ok", ID: "999", State: `{"state":{"name":"ok"}}`},
				{Key: "resources.jobs.foo", ID: "123", State: "not json"},
			},
			wantErr: "interpreting state recorded for resources.jobs.foo",
			want:    fileState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var db DeploymentState
			db.Data.State = tt.existing
			db.stateIDs = stateIDsOf(tt.existing)

			err := db.applyDMSState(tt.recorded)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, db.Data.State)
			assert.Equal(t, stateIDsOf(tt.want), db.stateIDs)
		})
	}
}

// stateIDsOf is the id index the state keeps alongside its entries.
func stateIDsOf(entries map[string]ResourceEntry) map[string]string {
	ids := make(map[string]string, len(entries))
	for key, entry := range entries {
		ids[key] = entry.ID
	}
	return ids
}
