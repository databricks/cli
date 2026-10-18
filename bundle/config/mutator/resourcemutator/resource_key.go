package resourcemutator

import "github.com/databricks/cli/bundle/config"

// ResourceKey uniquely identifies a resource in configuration.
type ResourceKey struct {
	// Type is the type of the resource. E.g. "jobs", "dashboards", etc.
	Type string

	// Name is the resource name of the resource. E.g. "my_job"
	Name string
}

// ResourceKeySet is a set of resource keys in configuration.
//
// ResourceKeySet is used to track how mutators modify resources.
type ResourceKeySet map[string]map[string]struct{}

func NewResourceKeySet() ResourceKeySet {
	return make(map[string]map[string]struct{})
}

// AddResourceKey adds a resource key to the set.
func (r ResourceKeySet) AddResourceKey(key ResourceKey) {
	if _, ok := r[key.Type]; !ok {
		r[key.Type] = make(map[string]struct{})
	}

	r[key.Type][key.Name] = struct{}{}
}

func (r ResourceKeySet) IsEmpty() bool {
	return len(r) == 0
}

func (r ResourceKeySet) Size() int {
	size := 0

	for _, resources := range r {
		size += len(resources)
	}

	return size
}

// AddResources adds the keys of all resources in r.
func (r ResourceKeySet) AddResources(resources *config.Resources) {
	resources.IterateAllResources(func(typ, name string, _ config.ConfigResource) bool {
		r.AddResourceKey(ResourceKey{Type: typ, Name: name})
		return true
	})
}

// AddResourcesOfType adds the keys of the resources of type typ in r.
func (r ResourceKeySet) AddResourcesOfType(resources *config.Resources, typ string) {
	resources.IterateAllResources(func(t, name string, _ config.ConfigResource) bool {
		if t == typ {
			r.AddResourceKey(ResourceKey{Type: typ, Name: name})
		}
		return true
	})
}

// Types returns the types of all resources in the set.
func (r ResourceKeySet) Types() []string {
	var result []string

	for resourceType := range r {
		result = append(result, resourceType)
	}

	return result
}

// Names returns the names of all resources of a given type.
func (r ResourceKeySet) Names(resourceType string) []string {
	var result []string

	for resourceName := range r[resourceType] {
		result = append(result, resourceName)
	}

	return result
}

// ToArray converts the set to an array of resource keys.
func (r ResourceKeySet) ToArray() []ResourceKey {
	var result []ResourceKey

	for resourceType, resources := range r {
		for resourceName := range resources {
			result = append(result, ResourceKey{
				Type: resourceType,
				Name: resourceName,
			})
		}
	}

	return result
}
