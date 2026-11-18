package dstate_test

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseClone(t *testing.T) {
	original := dstate.NewDatabase("lineage", 4)
	original.Features = map[string]struct{}{dstate.FeatureDeploymentHistoryShadow: {}}
	original.State["resources.jobs.foo"] = dstate.ResourceEntry{
		ID:        "123",
		State:     json.RawMessage(`{"name":"foo"}`),
		DependsOn: []deployplan.DependsOnEntry{{Node: "resources.jobs.bar", Label: "dependency"}},
	}
	before, err := json.Marshal(original)
	require.NoError(t, err)

	clone := original.Clone()
	require.Equal(t, original, clone)
	clone.Serial++
	delete(clone.Features, dstate.FeatureDeploymentHistoryShadow)
	clone.State["resources.jobs.foo"].State[0] = '['
	clone.State["resources.jobs.foo"].DependsOn[0].Node = "changed"
	delete(clone.State, "resources.jobs.foo")
	clone.State["resources.jobs.new"] = dstate.ResourceEntry{ID: "456"}

	after, err := json.Marshal(original)
	require.NoError(t, err)
	assert.Equal(t, before, after)

	empty := (dstate.Database{}).Clone()
	require.NotNil(t, empty.State)
	assert.Empty(t, empty.State)
}
