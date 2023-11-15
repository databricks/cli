package merge

import (
	"fmt"

	"github.com/databricks/cli/libs/config"
)

// Merge recursively merges the specified values.
//
// Semantics are as follows:
// * Merging x with nil or nil with x always yields x.
// * Merging maps a and b means entries from map b take precedence.
// * Merging sequences a and b means concatenating them.
func Merge(a, b config.Value) (config.Value, error) {
	return merge(a, b)
}

func merge(a, b config.Value) (config.Value, error) {
	ak := a.Kind()
	bk := b.Kind()

	// If a is nil, return b.
	if ak == config.KindNil {
		return b, nil
	}

	// If b is nil, return a.
	if bk == config.KindNil {
		return a, nil
	}

	// Call the appropriate merge function based on the kind of a and b.
	switch ak {
	case config.KindMap:
		if bk != config.KindMap {
			return config.NilValue, fmt.Errorf("cannot merge map with %s", bk)
		}
		return mergeMap(a, b)
	case config.KindSequence:
		if bk != config.KindSequence {
			return config.NilValue, fmt.Errorf("cannot merge sequence with %s", bk)
		}
		return mergeSequence(a, b)
	default:
		if ak != bk {
			return config.NilValue, fmt.Errorf("cannot merge %s with %s", ak, bk)
		}
		return mergePrimitive(a, b)
	}
}

func mergeMap(a, b config.Value) (config.Value, error) {
	out := make(map[string]config.Value)
	am := a.MustMap()
	bm := b.MustMap()

	// Add the values from a into the output map.
	for k, v := range am {
		out[k] = v
	}

	// Merge the values from b into the output map.
	for k, v := range bm {
		if _, ok := out[k]; ok {
			// If the key already exists, merge the values.
			merged, err := merge(out[k], v)
			if err != nil {
				return config.NilValue, err
			}
			out[k] = merged
		} else {
			// Otherwise, just set the value.
			out[k] = v
		}
	}

	// Preserve the location of the first value.
	return config.NewValue(out, a.Location()), nil
}

func mergeSequence(a, b config.Value) (config.Value, error) {
	as := a.MustSequence()
	bs := b.MustSequence()

	// Merging sequences means concatenating them.
	out := make([]config.Value, len(as)+len(bs))
	copy(out[:], as)
	copy(out[len(as):], bs)

	// Preserve the location of the first value.
	return config.NewValue(out, a.Location()), nil
}

func mergePrimitive(a, b config.Value) (config.Value, error) {
	// Merging primitive values means using the incoming value.
	return b, nil
}
