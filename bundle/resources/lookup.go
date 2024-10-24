package resources

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
)

// Reference is a reference to a resource.
// It includes the resource type description, and a reference to the resource itself.
type Reference struct {
	Description config.ResourceDescription
	Resource    config.ConfigResource
}

// Map is the core type for resource lookup and completion.
type Map map[string][]Reference

// References returns maps of resource keys to a slice of [Reference].
//
// The first map is indexed by the resource key only.
// The second map is indexed by the resource type name and its key.
//
// While the return types allows for multiple resources to share the same key,
// this is confirmed not to happen in the [validate.UniqueResourceKeys]	mutator.
func References(b *bundle.Bundle) (Map, Map) {
	keyOnly := make(Map)
	keyWithType := make(Map)

	// Collect map of resource references indexed by their keys.
	for _, group := range b.Config.Resources.AllResources() {
		for k, v := range group.Resources {
			ref := Reference{
				Description: group.Description,
				Resource:    v,
			}

			kt := fmt.Sprintf("%s.%s", group.Description.PluralName, k)
			keyOnly[k] = append(keyOnly[k], ref)
			keyWithType[kt] = append(keyWithType[kt], ref)
		}
	}

	return keyOnly, keyWithType
}

// Lookup returns the resource with the specified key.
// If the key maps to more than one resource, an error is returned.
// If the key does not map to any resource, an error is returned.
func Lookup(b *bundle.Bundle, key string) (Reference, error) {
	keyOnlyRefs, keyWithTypeRefs := References(b)
	refs, ok := keyOnlyRefs[key]
	if !ok {
		refs, ok = keyWithTypeRefs[key]
		if !ok {
			return Reference{}, fmt.Errorf("resource with key %q not found", key)
		}
	}

	switch {
	case len(refs) == 1:
		return refs[0], nil
	case len(refs) > 1:
		return Reference{}, fmt.Errorf("multiple resources with key %q found", key)
	default:
		panic("unreachable")
	}
}
