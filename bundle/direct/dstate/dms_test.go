package dstate

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeResourceLister serves a fixed set of resources from ListResources. It
// embeds the SDK interface so it satisfies it while only overriding the one
// method the read path uses.
type fakeResourceLister struct {
	bundledeployments.BundleDeploymentsInterface
	resources []bundledeployments.Resource
}

func (f *fakeResourceLister) ListResources(ctx context.Context, req bundledeployments.ListResourcesRequest) listing.Iterator[bundledeployments.Resource] {
	return listing.NewIterator(
		&req,
		func(ctx context.Context, r bundledeployments.ListResourcesRequest) (*bundledeployments.ListResourcesResponse, error) {
			return &bundledeployments.ListResourcesResponse{Resources: f.resources}, nil
		},
		func(resp *bundledeployments.ListResourcesResponse) []bundledeployments.Resource {
			return resp.Resources
		},
		func(resp *bundledeployments.ListResourcesResponse) *bundledeployments.ListResourcesRequest {
			return nil
		},
	)
}

// testClient returns a DMS client that lists the resources f holds.
func testClient(f *fakeResourceLister) *dms.Client {
	return &dms.Client{Service: f}
}

func TestReadDMSStateUnwrapsEnvelope(t *testing.T) {
	// The service stores state as an opaque string, so the envelope arrives verbatim.
	envelope := `{"state":{"name":"foo"},"depends_on":[{"node":"resources.pipelines.bar","label":"${resources.pipelines.bar.id}"}]}`
	f := &fakeResourceLister{resources: []bundledeployments.Resource{
		{ResourceKey: "jobs.foo", ResourceId: "123", State: envelope},
		{ResourceKey: "pipelines.bar", ResourceId: "456"},
	}}

	var db DeploymentState
	require.NoError(t, db.readDMSState(t.Context(), &DMSSource{Client: testClient(f), DeploymentID: "dep-1"}))

	// depends_on comes back from the envelope, so a bundle whose local state was
	// wiped still has the edges needed for delete ordering.
	assert.Equal(t, map[string]ResourceEntry{
		"resources.jobs.foo": {
			ID:        "123",
			State:     json.RawMessage(`{"name":"foo"}`),
			DependsOn: []deployplan.DependsOnEntry{{Node: "resources.pipelines.bar", Label: "${resources.pipelines.bar.id}"}},
		},
		"resources.pipelines.bar": {ID: "456"},
	}, db.Data.State)
}

func TestReadDMSStateRejectsMalformedState(t *testing.T) {
	// The good resource comes first, so the error lands mid-way: what the file loaded has to
	// survive whole rather than end up half replaced.
	f := &fakeResourceLister{resources: []bundledeployments.Resource{
		{ResourceKey: "jobs.ok", ResourceId: "999", State: `{"state":{"name":"ok"}}`},
		{ResourceKey: "jobs.foo", ResourceId: "123", State: "not json"},
	}}

	var db DeploymentState
	db.Data.State = map[string]ResourceEntry{"resources.jobs.bar": {ID: "file-id"}}
	db.stateIDs = map[string]string{"resources.jobs.bar": "file-id"}

	err := db.readDMSState(t.Context(), &DMSSource{Client: testClient(f), DeploymentID: "dep-1"})
	assert.ErrorContains(t, err, "interpreting state recorded for resources.jobs.foo")

	assert.Equal(t, map[string]ResourceEntry{"resources.jobs.bar": {ID: "file-id"}}, db.Data.State)
	assert.Equal(t, map[string]string{"resources.jobs.bar": "file-id"}, db.stateIDs)
}

func TestReadDMSStateReplacesLocalState(t *testing.T) {
	// readDMSState should replace the file-derived state with what DMS has,
	// even if the file has different resources.
	src := &DMSSource{
		Client: testClient(&fakeResourceLister{resources: []bundledeployments.Resource{
			{ResourceKey: "jobs.foo", ResourceId: "dms-id", State: `{"state":{"name":"from-dms"}}`},
		}}),
		DeploymentID: "dep-1",
	}

	var db DeploymentState
	db.Data.State = map[string]ResourceEntry{
		"resources.jobs.bar": {ID: "file-id", State: json.RawMessage(`{"name":"from-file"}`)},
	}
	db.stateIDs = map[string]string{"resources.jobs.bar": "file-id"}
	db.Path = "test-path"

	err := db.readDMSState(t.Context(), src)
	require.NoError(t, err)

	// State now reflects DMS, not the file.
	assert.Equal(t, map[string]ResourceEntry{
		"resources.jobs.foo": {ID: "dms-id", State: json.RawMessage(`{"name":"from-dms"}`)},
	}, db.Data.State)
	assert.Equal(t, map[string]string{"resources.jobs.foo": "dms-id"}, db.stateIDs)
}

func TestReadDMSStateAcceptsEmptyResourceList(t *testing.T) {
	// An empty DMS response is valid: it means a successful deploy of nothing.
	src := &DMSSource{
		Client:       testClient(&fakeResourceLister{resources: []bundledeployments.Resource{}}),
		DeploymentID: "dep-1",
	}

	var db DeploymentState
	db.Data.State = map[string]ResourceEntry{
		"resources.jobs.bar": {ID: "file-id", State: json.RawMessage(`{"name":"from-file"}`)},
	}
	db.stateIDs = map[string]string{"resources.jobs.bar": "file-id"}
	db.Path = "test-path"

	err := db.readDMSState(t.Context(), src)
	require.NoError(t, err)

	// State is now empty, reflecting the empty DMS response.
	assert.Empty(t, db.Data.State)
	assert.Empty(t, db.stateIDs)
}
