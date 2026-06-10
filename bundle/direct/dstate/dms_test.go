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

	// Number of versions consumed from ListVersions iterators, to observe how
	// far a scan read into the (paginated) list.
	versionNexts int
}

func (c *fakeBundleClient) ListResources(context.Context, sdkbundle.ListResourcesRequest) listing.Iterator[sdkbundle.Resource] {
	it := listing.SliceIterator[sdkbundle.Resource](c.resources)
	return &it
}

func (c *fakeBundleClient) ListVersions(context.Context, sdkbundle.ListVersionsRequest) listing.Iterator[sdkbundle.Version] {
	it := listing.SliceIterator[sdkbundle.Version](c.versions)
	return &countingIterator{Iterator: &it, nexts: &c.versionNexts}
}

// countingIterator counts Next calls on behalf of fakeBundleClient.
type countingIterator struct {
	listing.Iterator[sdkbundle.Version]
	nexts *int
}

func (c *countingIterator) Next(ctx context.Context) (sdkbundle.Version, error) {
	*c.nexts++
	return c.Iterator.Next(ctx)
}

func raw(s string) *json.RawMessage {
	msg := json.RawMessage(s)
	return &msg
}

func completed(reason sdkbundle.VersionComplete) sdkbundle.Version {
	return sdkbundle.Version{Status: sdkbundle.VersionStatusVersionStatusCompleted, CompletionReason: reason}
}

var succeeded = completed(sdkbundle.VersionCompleteVersionCompleteSuccess)

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

	tests := []struct {
		name    string
		lineage string
		client  sdkbundle.BundleInterface
		want    map[string]string // expected resource key -> ID after Open
	}{
		{
			name:    "DMS owns the deployment: resources come from DMS, identity from the file",
			lineage: "dep-1",
			client:  &fakeBundleClient{resources: dmsResources, versions: []sdkbundle.Version{succeeded}},
			want:    map[string]string{"resources.jobs.foo": "job-1"},
		},
		{
			name:    "no successful version yet: fall back to the direct file-based state",
			lineage: "dep-1",
			client:  &fakeBundleClient{resources: dmsResources},
			want:    map[string]string{"resources.jobs.from_file": "file-1"},
		},
		{
			// The client reports a successful version and resources; if Open consulted
			// DMS despite the missing lineage, jobs.foo would appear in state.
			name:    "no lineage in the file (nothing deployed yet): never consult DMS",
			lineage: "",
			client:  &fakeBundleClient{resources: dmsResources, versions: []sdkbundle.Version{succeeded}},
			want:    map[string]string{"resources.jobs.from_file": "file-1"},
		},
		{
			name:    "nil client (record_deployment_history off): file-based state only",
			lineage: "dep-1",
			client:  nil,
			want:    map[string]string{"resources.jobs.from_file": "file-1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeStateFile(t, tc.lineage, fileState)
			var db DeploymentState
			require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(false), tc.client))

			assert.Equal(t, tc.lineage, db.Data.Lineage)
			ids := make(map[string]string)
			for key := range db.Data.State {
				// GetResourceID reads the stateIDs cache, so this also checks it was
				// rebuilt in sync with the overlaid state.
				ids[key] = db.GetResourceID(key)
			}
			assert.Equal(t, tc.want, ids)
		})
	}
}

// TestDeploymentHasSuccessfulVersion is the gate that decides whether DMS owns
// a deployment's state: only once a version has completed successfully.
// wantNexts pins the pagination contract: versions are listed newest-first and
// the scan stops at the first success, so deployments with long version
// histories don't fetch the whole list.
func TestDeploymentHasSuccessfulVersion(t *testing.T) {
	failed := completed(sdkbundle.VersionCompleteVersionCompleteFailure)
	inProgress := sdkbundle.Version{Status: sdkbundle.VersionStatusVersionStatusInProgress}
	tests := []struct {
		name      string
		versions  []sdkbundle.Version
		want      bool
		wantNexts int
	}{
		{"no versions", nil, false, 0},
		{"in progress", []sdkbundle.Version{inProgress}, false, 1},
		{"failed", []sdkbundle.Version{failed}, false, 1},
		{"succeeded", []sdkbundle.Version{succeeded}, true, 1},
		{"failed then succeeded", []sdkbundle.Version{failed, succeeded}, true, 2},
		{"stops at first success", []sdkbundle.Version{succeeded, inProgress}, true, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeBundleClient{versions: tc.versions}
			got, err := deploymentHasSuccessfulVersion(t.Context(), client, "dep-1")
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantNexts, client.versionNexts)
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
