package dstate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndGetState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	ds := &DeploymentState{}
	err := ds.Open(path)
	require.NoError(t, err)

	state := map[string]any{"name": "test"}
	err = ds.SaveState("resources.jobs.foo", "123", state)
	require.NoError(t, err)

	entry, ok := ds.GetResourceEntry("resources.jobs.foo")
	assert.True(t, ok)
	assert.Equal(t, "123", entry.ID)
	assert.Equal(t, state, entry.State)

	err = ds.DeleteState("resources.jobs.foo")
	require.NoError(t, err)

	_, ok = ds.GetResourceEntry("resources.jobs.foo")
	assert.False(t, ok)
}

func TestOpenExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	data := Database{
		StateVersion: 1,
		CLIVersion:   "test-version",
		Lineage:      "test-lineage",
		Serial:       10,
		State: map[string]ResourceEntry{
			"resources.jobs.foo": {
				ID:    "456",
				State: map[string]any{"name": "foo"},
			},
		},
	}
	content, err := json.Marshal(data)
	require.NoError(t, err)
	err = os.WriteFile(path, content, 0o600)
	require.NoError(t, err)

	ds := &DeploymentState{}
	err = ds.Open(path)
	require.NoError(t, err)

	assert.Equal(t, "test-lineage", ds.Data.Lineage)
	assert.Equal(t, 10, ds.Data.Serial)

	entry, ok := ds.GetResourceEntry("resources.jobs.foo")
	assert.True(t, ok)
	assert.Equal(t, "456", entry.ID)
}

func TestOpenPreservesLargeIntegers(t *testing.T) {
	// Without UseNumber(), json.Unmarshal decodes numbers into float64,
	// which loses precision for integers > 2^53.
	content := []byte(`{"state":{"resources.jobs.foo":{"__id__":"x","state":{"job_id":9007199254740993}}}}`)

	// json.Unmarshal loses precision
	var badData struct {
		State map[string]struct {
			State map[string]any `json:"state"`
		} `json:"state"`
	}
	err := json.Unmarshal(content, &badData)
	require.NoError(t, err)
	badValue := badData.State["resources.jobs.foo"].State["job_id"].(float64)
	assert.InEpsilon(t, float64(9007199254740992), badValue, 0.0001) // precision lost: ends in 2 not 3

	// UseNumber() preserves precision
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	err = os.WriteFile(path, content, 0o600)
	require.NoError(t, err)

	ds := &DeploymentState{}
	err = ds.Open(path)
	require.NoError(t, err)

	entry, _ := ds.GetResourceEntry("resources.jobs.foo")
	state := entry.State.(map[string]any)
	jobID := state["job_id"].(json.Number)
	parsed, _ := jobID.Int64()
	assert.Equal(t, int64(9007199254740993), parsed)
}

func TestFinalize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	ds := &DeploymentState{}
	err := ds.Open(path)
	require.NoError(t, err)

	initialSerial := ds.Data.Serial
	err = ds.SaveState("resources.jobs.foo", "123", nil)
	require.NoError(t, err)

	err = ds.Finalize()
	require.NoError(t, err)

	assert.Equal(t, initialSerial+1, ds.Data.Serial)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "resources.jobs.foo")
}

func TestExportStateWithDashboardEtag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	ds := &DeploymentState{}
	err := ds.Open(path)
	require.NoError(t, err)

	err = ds.SaveState("resources.dashboards.map_dash", "dash-1", map[string]any{
		"display_name": "Dashboard",
		"etag":         "etag-map",
	})
	require.NoError(t, err)

	err = ds.SaveState("resources.dashboards.struct_dash", "dash-2", &resources.DashboardConfig{
		DisplayName: "Dashboard",
		Etag:        "etag-struct",
	})
	require.NoError(t, err)

	exported := ds.ExportState(context.Background())
	assert.Equal(t, "etag-map", exported["resources.dashboards.map_dash"].ETag)
	assert.Equal(t, "etag-struct", exported["resources.dashboards.struct_dash"].ETag)
}
