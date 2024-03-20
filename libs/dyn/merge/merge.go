package merge

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// Merge recursively merges the specified values.
//
// Semantics are as follows:
// * Merging x with nil or nil with x always yields x.
// * Merging maps a and b means entries from map b take precedence.
// * Merging sequences a and b means concatenating them.
func Merge(a, b dyn.Value) (dyn.Value, error) {
	return merge(a, b)
}

func merge(a, b dyn.Value) (dyn.Value, error) {
	ak := a.Kind()
	bk := b.Kind()

	// If a is nil, return b.
	if ak == dyn.KindNil {
		return b, nil
	}

	// If b is nil, return a.
	if bk == dyn.KindNil {
		return a, nil
	}

	// Call the appropriate merge function based on the kind of a and b.
	switch ak {
	case dyn.KindMap:
		if bk != dyn.KindMap {
			return dyn.NilValue, fmt.Errorf("cannot merge map with %s", bk)
		}
		return mergeMap(a, b)
	case dyn.KindSequence:
		if bk != dyn.KindSequence {
			return dyn.NilValue, fmt.Errorf("cannot merge sequence with %s", bk)
		}
		return mergeSequence(a, b)
	default:
		if ak != bk {
			return dyn.NilValue, fmt.Errorf("cannot merge %s with %s", ak, bk)
		}
		return mergePrimitive(a, b)
	}
}

func mergeMap(a, b dyn.Value) (dyn.Value, error) {
	out := dyn.NewMapping()
	am := a.MustMapping()
	bm := b.MustMapping()

	// Add the values from a into the output map.
	out.Merge(am)

	// Merge the values from b into the output map.
	for _, pair := range bm.Pairs() {
		pk := pair.Key
		pv := pair.Value
		if ov, ok := out.Get(pk); ok {
			// If the key already exists, merge the values.
			merged, err := merge(ov, pv)
			if err != nil {
				return dyn.NilValue, err
			}
			out.Set(pk, merged)
		} else {
			// Otherwise, just set the value.
			out.Set(pk, pv)
		}
	}

	// Preserve the location of the first value.
	return dyn.NewValue(out, a.Location()), nil
}

func mergeSequence(a, b dyn.Value) (dyn.Value, error) {
	as := a.MustSequence()
	bs := b.MustSequence()

	// Merging sequences means concatenating them.
	out := make([]dyn.Value, len(as)+len(bs))
	copy(out[:], as)
	copy(out[len(as):], bs)

	// Preserve the location of the first value.
	return dyn.NewValue(out, a.Location()), nil
}

func mergePrimitive(a, b dyn.Value) (dyn.Value, error) {
	// Merging primitive values means using the incoming value.
	return b, nil
}
