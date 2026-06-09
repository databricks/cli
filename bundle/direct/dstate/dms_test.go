package dstate

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/listing"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBundleClient struct {
	sdkbundle.BundleInterface
	resources []sdkbundle.Resource
	versions  []sdkbundle.Version
}

func (c *fakeBundleClient) ListResources(context.Context, sdkbundle.ListResourcesRequest) listing.Iterator[sdkbundle.Resource] {
	it := listing.SliceIterator[sdkbundle.Resource](c.resources)
	return &it
}

func (c *fakeBundleClient) ListVersions(context.Context, sdkbundle.ListVersionsRequest) listing.Iterator[sdkbundle.Version] {
	it := listing.SliceIterator[sdkbundle.Version](c.versions)
	return &it
}

func raw(s string) *json.RawMessage {
	msg := json.RawMessage(s)
	return &msg
}

func succeeded() sdkbundle.Version {
	return sdkbundle.Version{
		Status:           sdkbundle.VersionStatusVersionStatusCompleted,
		CompletionReason: sdkbundle.VersionCompleteVersionCompleteSuccess,
	}
}

// writeStateFile writes a resources.json with the given lineage and resources,
// standing in for a prior local (direct) deployment, and returns its path.
func writeStateFile(t *testing.T, lineage string, state map[string]ResourceEntry) string {
	t.Helper()
	db := NewDatabase(lineage, 1)
	maps.Copy(db.State, state)
	content, err := json.Marshal(db)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "resources.json")
	require.NoError(t, os.WriteFile(path, content, 0o600))
	return path
}

// TestOpenWithDMS covers how Open chooses between the local file and the
// deployment metadata service. Identity (lineage) always comes from the file;
// only the resource set is overlaid from DMS, and only when DMS owns the
// deployment (a version has completed successfully).
func TestOpenWithDMS(t *testing.T) {
	fileState := map[string]ResourceEntry{"resources.jobs.from_file": {ID: "file-1"}}
	dmsResources := []sdkbundle.Resource{
		{ResourceKey: "jobs.foo", ResourceId: "job-1", State: raw(`{"name":"foo"}`)},
	}

	t.Run("DMS owns the deployment: resources come from DMS, identity from the file", func(t *testing.T) {
		path := writeStateFile(t, "dep-1", fileState)
		client := &fakeBundleClient{resources: dmsResources, versions: []sdkbundle.Version{succeeded()}}

		var db DeploymentState
		require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(false), client))

		assert.Equal(t, "dep-1", db.Data.Lineage)
		_, fromFile := db.GetResourceEntry("resources.jobs.from_file")
		assert.False(t, fromFile)
		entry, ok := db.GetResourceEntry("resources.jobs.foo")
		require.True(t, ok)
		assert.Equal(t, "job-1", entry.ID)
		assert.Equal(t, "job-1", db.GetResourceID("resources.jobs.foo"))
	})

	t.Run("no successful version yet: fall back to the direct file-based state", func(t *testing.T) {
		path := writeStateFile(t, "dep-1", fileState)
		client := &fakeBundleClient{resources: dmsResources, versions: nil}

		var db DeploymentState
		require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(false), client))

		assert.Equal(t, "dep-1", db.Data.Lineage)
		assert.Equal(t, "file-1", db.GetResourceID("resources.jobs.from_file"))
		_, fromDMS := db.GetResourceEntry("resources.jobs.foo")
		assert.False(t, fromDMS)
	})

	t.Run("nothing deployed yet: empty lineage never consults DMS", func(t *testing.T) {
		path := writeStateFile(t, "", nil)
		// The client reports a successful version and resources; if Open consulted
		// DMS despite the missing lineage, the resource below would appear in state.
		client := &fakeBundleClient{resources: dmsResources, versions: []sdkbundle.Version{succeeded()}}

		var db DeploymentState
		require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(false), client))

		assert.Empty(t, db.Data.Lineage)
		_, fromDMS := db.GetResourceEntry("resources.jobs.foo")
		assert.False(t, fromDMS)
	})

	t.Run("nil client: file-based state only", func(t *testing.T) {
		path := writeStateFile(t, "dep-1", fileState)

		var db DeploymentState
		require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(false), nil))

		assert.Equal(t, "file-1", db.GetResourceID("resources.jobs.from_file"))
	})
}

// TestDeploymentHasSuccessfulVersion is the gate that decides whether DMS owns a
// deployment's state. DMS is authoritative only once a version has completed
// successfully; otherwise (no versions, or only failed/in-progress ones) the
// caller falls back to the local file.
func TestDeploymentHasSuccessfulVersion(t *testing.T) {
	completed := func(reason sdkbundle.VersionComplete) sdkbundle.Version {
		return sdkbundle.Version{Status: sdkbundle.VersionStatusVersionStatusCompleted, CompletionReason: reason}
	}
	tests := []struct {
		name     string
		versions []sdkbundle.Version
		want     bool
	}{
		{"no versions", nil, false},
		{"in progress", []sdkbundle.Version{{Status: sdkbundle.VersionStatusVersionStatusInProgress}}, false},
		{"failed", []sdkbundle.Version{completed(sdkbundle.VersionCompleteVersionCompleteFailure)}, false},
		{"succeeded", []sdkbundle.Version{completed(sdkbundle.VersionCompleteVersionCompleteSuccess)}, true},
		{
			"failed then succeeded",
			[]sdkbundle.Version{completed(sdkbundle.VersionCompleteVersionCompleteFailure), completed(sdkbundle.VersionCompleteVersionCompleteSuccess)},
			true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := deploymentHasSuccessfulVersion(t.Context(), &fakeBundleClient{versions: tc.versions}, "dep-1")
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestFetchDeploymentResources covers mapping DMS resources to state entries.
func TestFetchDeploymentResources(t *testing.T) {
	tests := []struct {
		name      string
		resources []sdkbundle.Resource
		want      map[string]ResourceEntry
	}{
		{
			name:      "keys are prefixed with resources. and id/state are copied",
			resources: []sdkbundle.Resource{{ResourceKey: "jobs.foo", ResourceId: "job-1", State: raw(`{"name":"foo"}`)}},
			want:      map[string]ResourceEntry{"resources.jobs.foo": {ID: "job-1", State: json.RawMessage(`{"name":"foo"}`)}},
		},
		{
			name:      "missing state becomes nil",
			resources: []sdkbundle.Resource{{ResourceKey: "jobs.foo", ResourceId: "job-1"}},
			want:      map[string]ResourceEntry{"resources.jobs.foo": {ID: "job-1"}},
		},
		{
			name:      "empty list yields no entries",
			resources: nil,
			want:      map[string]ResourceEntry{},
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
