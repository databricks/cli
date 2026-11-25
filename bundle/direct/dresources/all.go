package dresources

import (
	"fmt"
	"iter"
	"maps"
	"sync"

	"github.com/databricks/databricks-sdk-go"
)

type adapterRegistration struct {
	resourceType string
	resource     any
}

// adapterRegistrations holds what the init() of each adapter's file registered.
var adapterRegistrations []adapterRegistration

// derivedAdapterSources yield adapters for resource types that follow from what the
// resource types declare in bundle/config/resources. They are read on first use only,
// since computing them inspects every resource type.
var derivedAdapterSources []iter.Seq2[string, any]

// registerAdapter registers resource, a typed nil pointer to the adapter
// implementation, under resourceType. Call it from the init() of the adapter's file.
func registerAdapter(resourceType string, resource any) {
	adapterRegistrations = append(adapterRegistrations, adapterRegistration{resourceType: resourceType, resource: resource})
}

// registerDerivedAdapters registers a source of adapters in the form registerAdapter takes.
// Call it from an init(); the source is not read until the adapters are first used.
func registerDerivedAdapters(source iter.Seq2[string, any]) {
	derivedAdapterSources = append(derivedAdapterSources, source)
}

var supportedResources = sync.OnceValue(func() map[string]any {
	return mergeAdapters(adapterRegistrations, derivedAdapterSources)
})

// mergeAdapters returns the registered and the derived adapters by resource type. The
// result does not depend on the order of registration, because a resource type with
// more than one adapter panics.
func mergeAdapters(registrations []adapterRegistration, sources []iter.Seq2[string, any]) map[string]any {
	result := make(map[string]any)
	add := func(resourceType string, resource any) {
		if _, ok := result[resourceType]; ok {
			panic(fmt.Sprintf("resource type %q has more than one adapter", resourceType))
		}
		result[resourceType] = resource
	}
	for _, r := range registrations {
		add(r.resourceType, r.resource)
	}
	for _, source := range sources {
		for resourceType, resource := range source {
			add(resourceType, resource)
		}
	}
	return result
}

// SupportedResources returns the adapter of every resource type the direct engine
// deploys, keyed by resource type ("jobs", "jobs.permissions"). Each adapter is a
// typed nil pointer to its implementation, as [NewAdapter] takes it. It returns a new
// map on every call, so the caller may modify it.
func SupportedResources() map[string]any {
	return maps.Clone(supportedResources())
}

func InitAll(client *databricks.WorkspaceClient) (map[string]*Adapter, error) {
	result := make(map[string]*Adapter)
	for resourceType, resource := range supportedResources() {
		adapter, err := NewAdapter(resource, resourceType, client)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", resourceType, err)
		}
		result[resourceType] = adapter
	}
	return result, nil
}
