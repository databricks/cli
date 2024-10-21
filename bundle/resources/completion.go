package resources

import (
	"github.com/databricks/cli/bundle"
	"golang.org/x/exp/maps"
)

func CompletionMap(b *bundle.Bundle) map[string]string {
	out := make(map[string]string)
	keyOnly, keyWithType := Keys(b)

	// Keep track of resources we have seen by their fully qualified key.
	seen := make(map[string]bool)

	// First add resources that can be identified by key alone.
	for k, v := range keyOnly {
		// Invariant: len(v) >= 1. See [ResourceKeys].
		if len(v) == 1 {
			seen[v[0].key] = true
			out[k] = v[0].resource.GetName()
		}
	}

	// Then add resources that can only be identified by their type and key.
	for k, v := range keyWithType {
		// Invariant: len(v) == 1. See [ResourceKeys].
		_, ok := seen[v[0].key]
		if ok {
			continue
		}
		out[k] = v[0].resource.GetName()
	}

	return out
}

func Completions(b *bundle.Bundle) []string {
	return maps.Keys(CompletionMap(b))
}
