package dresources

import (
	"iter"
	"maps"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

// resourceTypesWithoutAdapter lists the resource types that the direct engine does not
// deploy, each with the reason. Keys are as in config.ResourcesTypes ("jobs", "jobs.permissions").
var resourceTypesWithoutAdapter = map[string]string{}

// Every resource type, and the permissions and grants of each type that has them, has an
// adapter registered from its file, and every adapter belongs to one of them.
func TestEveryResourceTypeHasAdapter(t *testing.T) {
	supported := SupportedResources()
	resourceTypes := config.ResourcesTypes()

	for _, resourceType := range slices.Sorted(maps.Keys(resourceTypes)) {
		_, hasAdapter := supported[resourceType]
		_, optedOut := resourceTypesWithoutAdapter[resourceType]
		if !hasAdapter && !optedOut {
			t.Errorf("resource type %q has no adapter; register one from the init() of its file", resourceType)
		}
		if hasAdapter && optedOut {
			t.Errorf("resource type %q has an adapter but is listed in resourceTypesWithoutAdapter", resourceType)
		}
	}

	for _, resourceType := range slices.Sorted(maps.Keys(resourceTypesWithoutAdapter)) {
		if _, ok := resourceTypes[resourceType]; !ok {
			t.Errorf("%q is listed in resourceTypesWithoutAdapter but is not a resource type", resourceType)
		}
	}

	for _, resourceType := range slices.Sorted(maps.Keys(supported)) {
		if _, ok := resourceTypes[resourceType]; !ok {
			t.Errorf("%q has an adapter but is not a resource type", resourceType)
		}
	}
}

func TestMergeAdaptersRejectsTwoAdaptersForOneType(t *testing.T) {
	registered := []adapterRegistration{{resourceType: "jobs", resource: (*ResourceJob)(nil)}}
	derived := func(yield func(string, any) bool) {
		yield("jobs", (*ResourcePipeline)(nil))
	}

	assert.PanicsWithValue(t, `resource type "jobs" has more than one adapter`, func() {
		mergeAdapters(append(registered, registered...), nil)
	})
	assert.PanicsWithValue(t, `resource type "jobs" has more than one adapter`, func() {
		mergeAdapters(registered, []iter.Seq2[string, any]{derived})
	})
}

func TestRegisterAdapterAfterFirstReadPanics(t *testing.T) {
	SupportedResources()

	assert.PanicsWithValue(t, `adapter for "jobs" registered after the adapters were first read`, func() {
		registerAdapter("jobs", (*ResourceJob)(nil))
	})
	assert.PanicsWithValue(t, "derived adapters registered after the adapters were first read", func() {
		registerDerivedAdapters(func(yield func(string, any) bool) {})
	})
}

func TestSupportedResourcesReturnsCopy(t *testing.T) {
	m := SupportedResources()
	delete(m, "jobs")
	assert.Contains(t, SupportedResources(), "jobs")
}
