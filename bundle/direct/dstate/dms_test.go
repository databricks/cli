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

func TestFetchDeploymentResourcesUnwrapsEnvelope(t *testing.T) {
	recorded := `{"state":{"name":"foo"},"depends_on":[{"node":"resources.pipelines.bar","label":"${resources.pipelines.bar.id}"}]}`
	f := &fakeResourceLister{resources: []bundledeployments.Resource{
		{ResourceKey: "jobs.foo", ResourceId: "123", State: recorded},
		{ResourceKey: "pipelines.bar", ResourceId: "456"},
	}}

	got, err := fetchDeploymentResources(t.Context(), f, "dep-1")
	require.NoError(t, err)

	// depends_on comes back from the envelope, so a bundle whose local state was
	// wiped still has the edges needed for delete ordering.
	assert.Equal(t, map[string]ResourceEntry{
		"resources.jobs.foo": {
			ID:        "123",
			State:     json.RawMessage(`{"name":"foo"}`),
			DependsOn: []deployplan.DependsOnEntry{{Node: "resources.pipelines.bar", Label: "${resources.pipelines.bar.id}"}},
		},
		"resources.pipelines.bar": {ID: "456"},
	}, got)
}

func TestFetchDeploymentResourcesRejectsMalformedState(t *testing.T) {
	recorded := `not json`
	f := &fakeResourceLister{resources: []bundledeployments.Resource{
		{ResourceKey: "jobs.foo", ResourceId: "123", State: recorded},
	}}

	_, err := fetchDeploymentResources(t.Context(), f, "dep-1")
	assert.ErrorContains(t, err, "interpreting state recorded for resources.jobs.foo")
}

func TestFetchDeploymentResourcesNormalizesIntegralDoubles(t *testing.T) {
	// State recorded by an older server that round-tripped it through a protobuf
	// Struct comes back with integers as doubles ("1.0"). The typed job state
	// unmarshals those fields as int, so they must be restored to integers on
	// the way out. (A current server stores the string unchanged, so this only
	// matters for legacy state.)
	recorded := `{"state":{"max_concurrent_runs":1.0,"tasks":[{"new_cluster":{"num_workers":2.0}}],"timeout_seconds":0.0}}`
	f := &fakeResourceLister{resources: []bundledeployments.Resource{
		{ResourceKey: "jobs.foo", ResourceId: "123", State: recorded},
	}}

	got, err := fetchDeploymentResources(t.Context(), f, "dep-1")
	require.NoError(t, err)
	assert.Equal(t, json.RawMessage(`{"max_concurrent_runs":1,"tasks":[{"new_cluster":{"num_workers":2}}],"timeout_seconds":0}`), got["resources.jobs.foo"].State)
}

func TestNormalizeIntegralNumbers(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"integral doubles become ints", `{"a":1.0,"b":2.0}`, `{"a":1,"b":2}`},
		{"fractions are preserved", `{"a":1.5,"b":0.25}`, `{"a":1.5,"b":0.25}`},
		{"nested objects and arrays", `{"tasks":[{"n":1.0},{"n":2.5}]}`, `{"tasks":[{"n":1},{"n":2.5}]}`},
		{"large integral double", `{"id":1000000000000000.0}`, `{"id":1000000000000000}`},
		{"non-numbers untouched", `{"s":"x","b":true,"z":null}`, `{"b":true,"s":"x","z":null}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeIntegralNumbers(json.RawMessage(tc.in))
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(got))
		})
	}
}

func TestNormalizeIntegralNumbersEmptyInputUnchanged(t *testing.T) {
	got, err := normalizeIntegralNumbers(nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}
