package dstate

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
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

func TestFetchDeploymentResourcesPreservesLocalDependsOn(t *testing.T) {
	state := json.RawMessage(`{"name":"foo"}`)
	f := &fakeResourceLister{resources: []bundledeployments.Resource{
		{ResourceKey: "jobs.foo", ResourceId: "123", State: &state},
		{ResourceKey: "pipelines.bar", ResourceId: "456"},
	}}

	dependsOn := []deployplan.DependsOnEntry{{Node: "resources.pipelines.bar", Label: "pipeline_id"}}
	local := map[string]ResourceEntry{
		"resources.jobs.foo": {ID: "stale", DependsOn: dependsOn},
	}

	got, err := fetchDeploymentResources(t.Context(), f, "dep-1", local)
	require.NoError(t, err)

	// DMS owns the ID and state, but it does not record dependency edges, so
	// depends_on must survive from the local entry. Losing it breaks delete
	// ordering and --select expansion.
	assert.Equal(t, map[string]ResourceEntry{
		"resources.jobs.foo":      {ID: "123", State: state, DependsOn: dependsOn},
		"resources.pipelines.bar": {ID: "456"},
	}, got)
}

func TestFetchDeploymentResourcesWithNoLocalState(t *testing.T) {
	f := &fakeResourceLister{resources: []bundledeployments.Resource{
		{ResourceKey: "jobs.foo", ResourceId: "123"},
	}}

	// A bundle whose local state was wiped has no entry to carry depends_on from;
	// the resource is still recovered from DMS.
	got, err := fetchDeploymentResources(t.Context(), f, "dep-1", nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]ResourceEntry{
		"resources.jobs.foo": {ID: "123"},
	}, got)
}
