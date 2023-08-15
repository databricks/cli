package generator

import (
	"slices"

	"golang.org/x/exp/maps"
)

// sortKeys returns a sorted copy of the keys in the specified map.
func sortKeys[M ~map[K]V, K string, V any](m M) []K {
	keys := maps.Keys(m)
	slices.Sort(keys)
	return keys
}
