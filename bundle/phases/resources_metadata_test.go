package phases

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceMetadata_CountsFromConfigSizesFromState(t *testing.T) {
	r := &config.Resources{
		Jobs: map[string]*resources.Job{
			"foo": {},
			"bar": {},
		},
		Pipelines: map[string]*resources.Pipeline{"qux": {}},
		// Declared but never deployed, so it has no state and no sizes.
		Volumes: map[string]*resources.Volume{"vol": {}},
	}
	state := resourcestate.ExportedResourcesMap{
		"resources.jobs.foo":             {StateSizeBytes: 20},
		"resources.jobs.bar":             {StateSizeBytes: 10},
		"resources.jobs.foo.permissions": {StateSizeBytes: 2},
		"resources.pipelines.qux":        {StateSizeBytes: 14},
	}

	got := resourceMetadata(r, state)

	// Sorted by resource type. jobs median is the lower-middle of [10,20], so 10.
	// jobs.permissions is not a configuration type, so its count comes from state.
	assert.Equal(t, []protos.ResourceMetadata{
		{ResourceType: "jobs", Count: 2, StateSizeMaxBytes: 20, StateSizeMeanBytes: 15, StateSizeMedianBytes: 10},
		{ResourceType: "jobs.permissions", Count: 1, StateSizeMaxBytes: 2, StateSizeMeanBytes: 2, StateSizeMedianBytes: 2},
		{ResourceType: "pipelines", Count: 1, StateSizeMaxBytes: 14, StateSizeMeanBytes: 14, StateSizeMedianBytes: 14},
		{ResourceType: "volumes", Count: 1},
	}, got)
}

// A terraform deploy has no per-resource state, so it reports counts only.
func TestResourceMetadata_CountsWithoutState(t *testing.T) {
	r := &config.Resources{
		Jobs: map[string]*resources.Job{"foo": {}},
	}

	got := resourceMetadata(r, nil)

	assert.Equal(t, []protos.ResourceMetadata{
		{ResourceType: "jobs", Count: 1},
	}, got)
}

func TestStatHelpers(t *testing.T) {
	assert.Equal(t, int64(3), statMax([]int64{1, 2, 3}))
	assert.Equal(t, int64(2), statMean([]int64{1, 2, 3}))
	assert.Equal(t, int64(2), statMedian([]int64{1, 2, 3}))
	// Lower-middle for even count: sorted [1,2,3,4] -> index (4-1)/2 = 1 -> 2.
	assert.Equal(t, int64(2), statMedian([]int64{1, 2, 3, 4}))
}

func TestResourceMetadata_SkipsNonResourceStateKeys(t *testing.T) {
	state := resourcestate.ExportedResourcesMap{
		"resources.jobs.foo": {StateSizeBytes: 5},
		"bogus":              {StateSizeBytes: 99},
	}
	got := resourceMetadata(&config.Resources{}, state)
	assert.Equal(t, []protos.ResourceMetadata{
		{ResourceType: "jobs", Count: 1, StateSizeMaxBytes: 5, StateSizeMeanBytes: 5, StateSizeMedianBytes: 5},
	}, got)
}

func TestCollectResourcesMetadata_NilWhenNoResources(t *testing.T) {
	b := &bundle.Bundle{}
	assert.Nil(t, collectResourcesMetadata(t.Context(), b))
}

// A type missing from the report is indistinguishable from zero adoption on the
// dashboard, so fill one resource of every type to catch a type that falls out of
// AllResources.
func TestResourceMetadataCoversAllResourceTypes(t *testing.T) {
	var r config.Resources
	rv := reflect.ValueOf(&r).Elem()
	rt := rv.Type()

	// Resource fields are all map[string]*T; TestCustomMarshallerIsImplemented in
	// bundle/config enforces that shape.
	for i := range rt.NumField() {
		mapType := rt.Field(i).Type
		m := reflect.MakeMap(mapType)
		m.SetMapIndex(reflect.ValueOf("my_resource"), reflect.New(mapType.Elem().Elem()))
		rv.Field(i).Set(m)
	}

	got := resourceMetadata(&r, nil)
	require.Len(t, got, rt.NumField())

	for _, m := range got {
		assert.Equal(t, int64(1), m.Count, "resource type %q", m.ResourceType)
	}
}
