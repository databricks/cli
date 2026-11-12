package dstate_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindingPreviewDoesNotChangeState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "resources.json")
	var state dstate.DeploymentState
	state.OpenWithData(statePath, dstate.NewDatabase("lineage", 4))
	bindings := map[string]*deployplan.BindEntry{
		"resources.jobs.job": {ID: "123", State: json.RawMessage(`{"name":"existing"}`)},
	}
	preview := state.WithBindings(bindings)
	assert.Equal(t, "123", preview.State["resources.jobs.job"].ID)
	assert.Empty(t, state.GetResourceID("resources.jobs.job"))
	assert.Empty(t, state.Data.State)
	assert.Equal(t, 4, state.GetSerial())
	assert.NoFileExists(t, statePath)
	assert.NoFileExists(t, statePath+".wal")
}

func TestBindRejectsExistingResourcesBeforeWriting(t *testing.T) {
	for _, id := range []string{"existing-id", "another-id"} {
		t.Run(id, func(t *testing.T) {
			statePath := filepath.Join(t.TempDir(), "resources.json")
			var state dstate.DeploymentState
			data := dstate.NewDatabase("lineage", 1)
			data.State["resources.jobs.existing"] = dstate.ResourceEntry{ID: "existing-id", State: json.RawMessage(`{}`)}
			state.OpenWithData(statePath, data)
			require.NoError(t, state.UpgradeToWrite())
			err := state.Bind(t.Context(), map[string]*deployplan.BindEntry{
				"resources.jobs.a_new":    {ID: "new-id", State: json.RawMessage(`{}`)},
				"resources.jobs.existing": {ID: id, State: json.RawMessage(`{}`)},
			})
			require.ErrorContains(t, err, "already bound")
			assert.Empty(t, state.GetResourceID("resources.jobs.a_new"))
			assert.Equal(t, "existing-id", state.GetResourceID("resources.jobs.existing"))
			_, err = state.Finalize(t.Context())
			require.NoError(t, err)
		})
	}
}
