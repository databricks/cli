package mutator

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// ResourceKey uniquely identifies a resource in configuration.
type ResourceKey struct {
	// Type is the type of the resource. E.g. "jobs", "dashboards", etc.
	Type string

	// Path is the path to the resource.
	Name string
}

// ResourceKeySet is a set of resource keys in configuration.
type ResourceKeySet map[string]map[string]struct{}

func NewResourceKeySet() ResourceKeySet {
	return make(map[string]map[string]struct{})
}

// Add adds a resource key to the set.
func (r ResourceKeySet) AddResourceKey(key ResourceKey) {
	if _, ok := r[key.Type]; !ok {
		r[key.Type] = make(map[string]struct{})
	}

	r[key.Type][key.Name] = struct{}{}
}

// Add adds resources associated with prefix/value to the set.
//
// prefix is a path to value in configuration.
//
// If prefix is a resource key ("resources.jobs.job_1"), it's added to the set.
// If prefix is a resource value ("resources.jobs.job_1.foo"), resource key is extracted and added.
// If prefix is a sub-tree ("resources" or "resources.jobs"), all resource keys are added.
func (r ResourceKeySet) Add(prefix dyn.Path, value dyn.Value) error {
	if len(prefix) >= 3 {
		parsed, err := GetResourceKey(prefix)
		if err != nil {
			return err
		}

		r.AddResourceKey(parsed)

		return nil
	} else {
		if value.Kind() != dyn.KindMap {
			return fmt.Errorf("expected value to be a map, got %s", value.Kind())
		}

		for _, pair := range value.MustMap().Pairs() {
			key := dyn.Key(pair.Key.MustString())

			err := r.Add(prefix.Append(key), pair.Value)
			if err != nil {
				return err
			}
		}

		return nil
	}
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

func GetResourceKey(path dyn.Path) (ResourceKey, error) {
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
