package resources

import (
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
)

// Reference is a reference to a resource.
// It includes the resource type description, and a reference to the resource itself.
type Reference struct {
	// Key is the unique key of the resource, e.g. "my_job".
	Key string

	// KeyWithType is the unique key of the resource, including the resource type, e.g. "jobs.my_job".
	KeyWithType string

	// Description is the resource type description.
	Description resources.ResourceDescription

	// Resource is the resource itself.
	Resource config.ConfigResource
}

// Map is the core type for resource lookup and completion.
type Map map[string][]Reference

// Filter defines the function signature for filtering resources.
type Filter func(Reference) bool

// includeReference checks if the specified reference passes all filters.
// If the list of filters is empty, the reference is always included.
func includeReference(filters []Filter, ref Reference) bool {
	for _, filter := range filters {
		if !filter(ref) {
			return false
		}
	}
	return true
}

// References returns maps of resource keys to a slice of [Reference].
//
// The first map is indexed by the resource key only.
// The second map is indexed by the resource type name and its key.
//
// While the return types allows for multiple resources to share the same key,
// this is confirmed not to happen in the [validate.UniqueResourceKeys]	mutator.
func References(b *bundle.Bundle, filters ...Filter) (Map, Map) {
	keyOnly := make(Map)
	keyWithType := make(Map)

	// Collect map of resource references indexed by their keys.
	for _, group := range b.Config.Resources.AllResources() {
		for k, v := range group.Resources {
			ref := Reference{
				Key:         k,
				KeyWithType: fmt.Sprintf("%s.%s", group.Description.PluralName, k),
				Description: group.Description,
				Resource:    v,
			}

			// Skip resources that do not pass all filters.
			if !includeReference(filters, ref) {
				continue
			}

			keyOnly[ref.Key] = append(keyOnly[ref.Key], ref)
			keyWithType[ref.KeyWithType] = append(keyWithType[ref.KeyWithType], ref)
		}
	}

	return keyOnly, keyWithType
}

// Lookup returns the resource with the specified key.
// If the key maps to more than one resource, an error is returned.
// If the key does not map to any resource, an error is returned.
func Lookup(b *bundle.Bundle, key string, filters ...Filter) (Reference, error) {
	keyOnlyRefs, keyWithTypeRefs := References(b, filters...)
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
