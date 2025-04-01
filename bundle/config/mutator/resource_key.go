package mutator

import (
	"errors"
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

func (r ResourceKeySet) IsEmpty() bool {
	return len(r) == 0
}

func (r ResourceKeySet) AddPath(path dyn.Path) error {
	resourceKey, err := GetResourceKey(path)
	if err != nil {
		return err
	}

	r.AddResourceKey(resourceKey)
	return nil
}

func (r ResourceKeySet) AddPattern(pattern dyn.Pattern, value dyn.Value) error {
	if len(pattern) != 3 {
		return errors.New("pattern must have 3 keys")
	}

	_, err := dyn.MapByPattern(value, pattern, func(path dyn.Path, v dyn.Value) (dyn.Value, error) {
		parsed, err := GetResourceKey(path)
		if err != nil {
			return dyn.InvalidValue, err
		}

		r.AddResourceKey(parsed)

		return v, nil
	})

	return err
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
