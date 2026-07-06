package phases

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
)

func TestResourceMetadataFromState_GroupsByType(t *testing.T) {
	state := resourcestate.ExportedResourcesMap{
		"resources.jobs.foo":             {StateSizeBytes: 20, StateCompressedSizeBytes: 12},
		"resources.jobs.bar":             {StateSizeBytes: 10, StateCompressedSizeBytes: 8},
		"resources.jobs.foo.permissions": {StateSizeBytes: 2, StateCompressedSizeBytes: 3},
		"resources.pipelines.qux":        {StateSizeBytes: 14, StateCompressedSizeBytes: 9},
	}

	got := resourceMetadataFromState(state)

	// Sorted by resource type. Sub-resources (permissions) group under
	// "<parent>.permissions" per config.GetResourceTypeFromKey. jobs median is
	// the lower-middle of sorted [10,20] -> index (2-1)/2 = 0 -> 10. Raw and
	// compressed stats are computed independently (each slice sorted on its own),
	// so a resource's raw and compressed values need not share a rank.
	assert.Equal(t, []protos.ResourceMetadata{
		{
			ResourceType: "jobs", Count: 2,
			StateSizeMaxBytes: 20, StateSizeMeanBytes: 15, StateSizeMedianBytes: 10,
			StateCompressedSizeMaxBytes: 12, StateCompressedSizeMeanBytes: 10, StateCompressedSizeMedianBytes: 8,
		},
		{
			ResourceType: "jobs.permissions", Count: 1,
			StateSizeMaxBytes: 2, StateSizeMeanBytes: 2, StateSizeMedianBytes: 2,
			StateCompressedSizeMaxBytes: 3, StateCompressedSizeMeanBytes: 3, StateCompressedSizeMedianBytes: 3,
		},
		{
			ResourceType: "pipelines", Count: 1,
			StateSizeMaxBytes: 14, StateSizeMeanBytes: 14, StateSizeMedianBytes: 14,
			StateCompressedSizeMaxBytes: 9, StateCompressedSizeMeanBytes: 9, StateCompressedSizeMedianBytes: 9,
		},
	}, got)
}

func TestStatHelpers(t *testing.T) {
	assert.Equal(t, int64(3), statMax([]int64{1, 2, 3}))
	assert.Equal(t, int64(2), statMean([]int64{1, 2, 3}))
	assert.Equal(t, int64(2), statMedian([]int64{1, 2, 3}))
	// Lower-middle for even count: sorted [1,2,3,4] -> index (4-1)/2 = 1 -> 2.
	assert.Equal(t, int64(2), statMedian([]int64{1, 2, 3, 4}))
}

func TestResourceMetadataFromState_SkipsNonResourceKeys(t *testing.T) {
	state := resourcestate.ExportedResourcesMap{
		"resources.jobs.foo": {StateSizeBytes: 5, StateCompressedSizeBytes: 4},
		"bogus":              {StateSizeBytes: 99, StateCompressedSizeBytes: 50},
	}
	got := resourceMetadataFromState(state)
	assert.Equal(t, []protos.ResourceMetadata{
		{
			ResourceType: "jobs", Count: 1,
			StateSizeMaxBytes: 5, StateSizeMeanBytes: 5, StateSizeMedianBytes: 5,
			StateCompressedSizeMaxBytes: 4, StateCompressedSizeMeanBytes: 4, StateCompressedSizeMedianBytes: 4,
		},
	}, got)
}

func TestCollectResourcesMetadata_NilWhenNoState(t *testing.T) {
	// Terraform deploys leave Metrics.ResourceState nil.
	b := &bundle.Bundle{}
	assert.Nil(t, collectResourcesMetadata(t.Context(), b))
}
