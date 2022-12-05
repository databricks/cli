package codegen

import (
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// sortKeys returns a sorted copy of the keys in the specified map.
func sortKeys[M ~map[K]V, K string, V any](m M) []K {
	keys := maps.Keys(m)
	slices.Sort(keys)
	return keys
}
