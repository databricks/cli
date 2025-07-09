package merge

import (
	"sort"

	"github.com/databricks/cli/libs/dyn"
)

type elementsByKey struct {
	key      string
	keyFunc  func(dyn.Value) string
	sortKeys bool
}

func (e elementsByKey) doMap(_ dyn.Path, v dyn.Value, mergeFunc func(a, b dyn.Value) (dyn.Value, error)) (dyn.Value, error) {
	// We know the type of this value is a sequence.
	// For additional defence, return self if it is not.
	elements, ok := v.AsSequence()
	if !ok {
		return v, nil
	}

	seen := make(map[string]dyn.Value, len(elements))
	keys := make([]string, 0, len(elements))

	// Iterate in natural order. For a given key, we first see the
	// base definition and merge instances that come after it.
	for i := range elements {
		kv := elements[i].Get(e.key)
		key := e.keyFunc(kv)

		// Register element with key if not yet seen before.
		ref, ok := seen[key]
		if !ok {
			keys = append(keys, key)
			seen[key] = elements[i]
			continue
		}

		// Merge this instance into the reference.
		nv, err := mergeFunc(ref, elements[i])
		if err != nil {
			return v, err
		}

		// Overwrite reference.
		seen[key] = nv
	}

	if e.sortKeys {
		sort.Strings(keys)
	}

	// Gather resulting elements in natural order.
	out := make([]dyn.Value, 0, len(keys))
	for _, key := range keys {
		nv, err := dyn.Set(seen[key], e.key, dyn.V(key))
		if err != nil {
			return dyn.InvalidValue, err
		}
		out = append(out, nv)
	}

	return dyn.NewValue(out, v.Locations()), nil
}

func (e elementsByKey) Map(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
	return e.doMap(nil, v, Merge)
}

func (e elementsByKey) MapWithOverride(p dyn.Path, v dyn.Value) (dyn.Value, error) {
	return e.doMap(nil, v, func(a, b dyn.Value) (dyn.Value, error) {
		return Override(a, b, OverrideVisitor{
			VisitInsert: func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
				return v, nil
			},
			VisitDelete: func(valuePath dyn.Path, left dyn.Value) error {
				return nil
			},
			VisitUpdate: func(_ dyn.Path, a, b dyn.Value) (dyn.Value, error) {
				return b, nil
			},
		})
	})
}

// ElementsByKey returns a [dyn.MapFunc] that operates on a sequence
// where each element is a map. It groups elements by a key and merges
// elements with the same key.
//
// The function that extracts the key from an element is provided as
// a parameter. The resulting elements get their key field overwritten
// with the value as returned by the key function.
func ElementsByKey(key string, keyFunc func(dyn.Value) string) dyn.MapFunc {
	return elementsByKey{key, keyFunc, false}.Map
}

func ElementsBySortedKey(key string, keyFunc func(dyn.Value) string) dyn.MapFunc {
	return elementsByKey{key, keyFunc, true}.Map
}

func ElementsByKeyWithOverride(key string, keyFunc func(dyn.Value) string) dyn.MapFunc {
	return elementsByKey{key, keyFunc, false}.MapWithOverride
}
