package statemgmt

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dstate"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBundleClient struct {
	sdkbundle.BundleInterface
	resources []sdkbundle.Resource
}

func (c *fakeBundleClient) ListResourcesAll(context.Context, sdkbundle.ListResourcesRequest) ([]sdkbundle.Resource, error) {
	return c.resources, nil
}

func raw(s string) *json.RawMessage {
	msg := json.RawMessage(s)
	return &msg
}

// writeLocalState writes a resources.json with the given lineage and resources,
// standing in for a prior local (direct) deployment, and returns its path.
func writeLocalState(t *testing.T, lineage string, state map[string]dstate.ResourceEntry) string {
	t.Helper()
	db := dstate.NewDatabase(lineage, 1)
	maps.Copy(db.State, state)
	content, err := json.Marshal(db)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "resources.json")
	require.NoError(t, os.WriteFile(path, content, 0o600))
	return path
}

// TestNewStateReaderSelection covers which reader is chosen. The DMS branch
// (managed state on + an existing lineage) needs a workspace client, so it is
// exercised directly through NewDMSStateReader in TestDMSStateReader.
func TestNewStateReaderSelection(t *testing.T) {
	tests := []struct {
		name    string
		managed bool
		lineage string
	}{
		{"managed state disabled uses the local file even with a deployment", false, "existing-lineage"},
		{"new deployment: managed state on but no lineage yet uses the local file", true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{}
			b.Config.Experimental = &config.Experimental{RecordDeploymentHistory: tc.managed}

			reader, err := NewStateReader(t.Context(), b, writeLocalState(t, tc.lineage, nil))
			require.NoError(t, err)
			assert.IsType(t, &fileStateReader{}, reader)
		})
	}
}

// TestDMSStateReader covers reading an existing deployment from DMS: the lineage
// (deployment id) is kept from the local state so a later deploy reuses the
// deployment, while the resource set is taken from DMS and any local resources
// (e.g. left over from a prior direct deployment) are dropped.
func TestDMSStateReader(t *testing.T) {
	path := writeLocalState(t, "dep-1", map[string]dstate.ResourceEntry{
		"resources.jobs.stale": {ID: "stale"},
	})
	client := &fakeBundleClient{resources: []sdkbundle.Resource{
		{ResourceKey: "jobs.foo", ResourceId: "job-1", State: raw(`{"name":"foo"}`)},
	}}

	var db dstate.DeploymentState
	require.NoError(t, NewDMSStateReader(client, "dep-1", path).Load(t.Context(), &db))

	assert.Equal(t, "dep-1", db.Data.Lineage)

	_, hasStale := db.GetResourceEntry("resources.jobs.stale")
	assert.False(t, hasStale)

	entry, ok := db.GetResourceEntry("resources.jobs.foo")
	require.True(t, ok)
	assert.Equal(t, "job-1", entry.ID)
}

// TestFetchDeploymentResources covers mapping DMS resources to state entries.
func TestFetchDeploymentResources(t *testing.T) {
	tests := []struct {
		name      string
		resources []sdkbundle.Resource
		want      map[string]dstate.ResourceEntry
	}{
		{
			name:      "keys are prefixed with resources. and id/state are copied",
			resources: []sdkbundle.Resource{{ResourceKey: "jobs.foo", ResourceId: "job-1", State: raw(`{"name":"foo"}`)}},
			want:      map[string]dstate.ResourceEntry{"resources.jobs.foo": {ID: "job-1", State: json.RawMessage(`{"name":"foo"}`)}},
		},
		{
			name:      "missing state becomes nil",
			resources: []sdkbundle.Resource{{ResourceKey: "jobs.foo", ResourceId: "job-1"}},
			want:      map[string]dstate.ResourceEntry{"resources.jobs.foo": {ID: "job-1"}},
		},
		{
			name:      "empty list yields no entries",
			resources: nil,
			want:      map[string]dstate.ResourceEntry{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fetchDeploymentResources(t.Context(), &fakeBundleClient{resources: tc.resources}, "dep-1")
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestFileStateReader covers the non-DMS path: reading the local resources.json.
func TestFileStateReader(t *testing.T) {
	t.Run("reads existing state", func(t *testing.T) {
		path := writeLocalState(t, "lineage-1", map[string]dstate.ResourceEntry{"resources.jobs.foo": {ID: "job-1"}})
		var db dstate.DeploymentState
		require.NoError(t, NewFileStateReader(path).Load(t.Context(), &db))
		assert.Equal(t, "lineage-1", db.Data.Lineage)
		assert.Equal(t, "job-1", db.GetResourceID("resources.jobs.foo"))
	})

	t.Run("missing file is empty state", func(t *testing.T) {
		var db dstate.DeploymentState
		require.NoError(t, NewFileStateReader(filepath.Join(t.TempDir(), "absent.json")).Load(t.Context(), &db))
		_, ok := db.GetResourceEntry("resources.jobs.foo")
		assert.False(t, ok)
	})
}

func TestReadLocalDatabase(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		db, err := readLocalDatabase(writeLocalState(t, "lineage-9", nil))
		require.NoError(t, err)
		assert.Equal(t, "lineage-9", db.Lineage)
	})

	t.Run("missing file has no lineage", func(t *testing.T) {
		db, err := readLocalDatabase(filepath.Join(t.TempDir(), "absent.json"))
		require.NoError(t, err)
		assert.Empty(t, db.Lineage)
	})

	t.Run("corrupt file errors", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "resources.json")
		require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))
		_, err := readLocalDatabase(path)
		assert.Error(t, err)
	})
}
