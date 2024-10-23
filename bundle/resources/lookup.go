package resources

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
)

// Reference is a reference to a resource.
// It includes the resource key, the resource type description, and a reference to the resource itself.
type Reference struct {
	Key         string
	Description config.ResourceDescription
	Resource    config.ConfigResource
}

// Map is the core type for resource lookup and completion.
type Map map[string][]Reference

// References returns a map of resource keys to a slice of [Reference].
// While its return type allows for multiple resources to share the same key,
// this is confirmed not to happen in the [validate.UniqueResourceKeys]	mutator.
func References(b *bundle.Bundle) Map {
	output := make(Map)

	// Collect map of resource references indexed by their keys.
	for _, group := range b.Config.Resources.AllResources() {
		for k, v := range group.Resources {
			output[k] = append(output[k], Reference{
				Key:         k,
				Description: group.Description,
				Resource:    v,
			})
		}
	}

	return output
}

// Lookup returns the resource with the specified key.
// If the key maps to more than one resource, an error is returned.
// If the key does not map to any resource, an error is returned.
func Lookup(b *bundle.Bundle, key string) (config.ConfigResource, error) {
	refs := References(b)
	res, ok := refs[key]
	if !ok {
		return nil, fmt.Errorf("resource with key %q not found", key)
	}

	switch {
	case len(res) == 1:
		return res[0].Resource, nil
	case len(res) > 1:
		return nil, fmt.Errorf("multiple resources with key %q found", key)
	default:
		panic("unreachable")
	}
}
