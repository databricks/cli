package resourcemutator

import (
	"errors"

	"github.com/databricks/cli/libs/dyn"
)

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

// AddPattern adds all resource keys that match the pattern.
func (r ResourceKeySet) AddPattern(pattern dyn.Pattern, root dyn.Value) error {
	if len(pattern) != 3 {
		return errors.New("pattern must have 3 keys")
	}

	_, err := dyn.MapByPattern(root, pattern, func(path dyn.Path, v dyn.Value) (dyn.Value, error) {
		parsed, err := getResourceKey(path)
		if err != nil {
			return dyn.InvalidValue, err
		}

		r.AddResourceKey(parsed)

		return v, nil
	})

	return err
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

func getResourceKey(path dyn.Path) (ResourceKey, error) {
	if len(path) < 3 {
		return ResourceKey{}, errors.New("can't parse resource key")
	}

	if path[0].Key() != "resources" {
		return ResourceKey{}, errors.New("can't parse resource key")
	}

	resourceType := path[1].Key()
	resourceName := path[2].Key()

	if resourceType == "" || resourceName == "" {
		return ResourceKey{}, errors.New("can't parse resource key")
	}

	return ResourceKey{
		Type: resourceType,
		Name: resourceName,
	}, nil
}
