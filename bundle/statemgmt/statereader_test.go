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
// (record deployment history on + an existing lineage) needs a workspace client,
// so it is exercised directly through NewDMSStateReader in TestDMSStateReader.
func TestNewStateReaderSelection(t *testing.T) {
	tests := []struct {
		name          string
		recordHistory bool
		lineage       string
	}{
		{"record deployment history disabled uses the local file even with a deployment", false, "existing-lineage"},
		{"new deployment: record deployment history on but no lineage yet uses the local file", true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{}
			b.Config.Experimental = &config.Experimental{RecordDeploymentHistory: tc.recordHistory}

			reader, err := NewStateReader(t.Context(), b, writeLocalState(t, tc.lineage, nil))
			require.NoError(t, err)
			assert.IsType(t, &fileStateReader{}, reader)
		})
	}
}

// TestDMSStateReader covers reading an existing deployment from DMS. The lineage
// (deployment id) always comes from resources.json. The resource set comes from
// DMS once DMS has it; until then (record_deployment_history just enabled on an
// existing deployment) resources.json's resources are kept so they aren't
// re-created.
func TestDMSStateReader(t *testing.T) {
	const lineage = "dep-1"
	localResources := map[string]dstate.ResourceEntry{
		"resources.jobs.from_file": {ID: "file-1"},
	}
	tests := []struct {
		name         string
		dmsResources []sdkbundle.Resource
		wantKey      string // the single resource expected in the loaded state
	}{
		{
			name:         "DMS owns the resources: the set comes from DMS, not resources.json",
			dmsResources: []sdkbundle.Resource{{ResourceKey: "jobs.foo", ResourceId: "job-1", State: raw(`{"name":"foo"}`)}},
			wantKey:      "resources.jobs.foo",
		},
		{
			name:         "feature just enabled on an existing deployment (DMS empty): resources.json is kept",
			dmsResources: nil,
			wantKey:      "resources.jobs.from_file",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeLocalState(t, lineage, localResources)

			var db dstate.DeploymentState
			require.NoError(t, NewDMSStateReader(&fakeBundleClient{resources: tc.dmsResources}, lineage, path).Load(t.Context(), &db))

			assert.Equal(t, lineage, db.Data.Lineage)
			_, ok := db.GetResourceEntry(tc.wantKey)
			assert.True(t, ok)
			assert.Len(t, db.Data.State, 1)
		})
	}
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
