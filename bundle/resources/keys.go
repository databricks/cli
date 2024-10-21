package resources

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
)

type pair struct {
	key      string
	resource config.ConfigResource
}

// lookup maps identifiers to a list of resources that match that identifier.
// The list can have more than 1 entry if resources of different types use the
// same key. When this happens, the user should disambiguate between them.
type lookup map[string][]pair

// Keys computes maps that index resources by their key (e.g. `my_job`) and by their key
// prefixed by their type (e.g. `jobs.my_job`). The resource key alone may be ambiguous (it is
// possible for resources of different types to have the same key), but the key prefixed by
// the type is guaranteed to be unique.
func Keys(b *bundle.Bundle) (keyOnly lookup, keyWithType lookup) {
	keyOnly = make(lookup)
	keyWithType = make(lookup)

	// Collect all resources by their key and prefixed key.
	for _, group := range b.Config.Resources.AllResources() {
		typ := group.Description.PluralName
		for k, v := range group.Resources {
			kt := fmt.Sprintf("%s.%s", typ, k)
			p := pair{key: kt, resource: v}
			keyOnly[k] = append(keyOnly[k], p)
			keyWithType[kt] = append(keyWithType[kt], p)
		}
	}

	return
}

// Lookup returns the resource with the given key.
// It first attempts to find a resource with the key alone.
// If this fails, it tries the key prefixed by the resource type.
// If this also fails, it returns an error.
func Lookup(b *bundle.Bundle, key string) (config.ConfigResource, error) {
	keyOnly, keyWithType := Keys(b)

	// First try to find the resource by key alone.
	if res, ok := keyOnly[key]; ok {
		if len(res) == 1 {
			return res[0].resource, nil
		}
	}

	// Then try to find the resource by key and type.
	if res, ok := keyWithType[key]; ok {
		if len(res) == 1 {
			return res[0].resource, nil
		}
	}

	return nil, fmt.Errorf("resource with key %q not found", key)
}
