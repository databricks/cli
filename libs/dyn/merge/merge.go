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
//
// Merging retains and accumulates the locations metadata associated with the values.
// This allows users of the module to track the provenance of values across merging of
// configuration trees, which is useful for reporting errors and warnings.
func Merge(a, b dyn.Value) (dyn.Value, error) {
	return merge(a, b)
}

func merge(a, b dyn.Value) (dyn.Value, error) {
	ak := a.Kind()
	bk := b.Kind()

	// If a is nil, return b.
	if ak == dyn.KindNil {
		return mergeLocations(b, a), nil
	}

	// If b is nil, return a.
	if bk == dyn.KindNil {
		return mergeLocations(a, b), nil
	}

	// Call the appropriate merge function based on the kind of a and b.
	switch ak {
	case dyn.KindMap:
		if bk != dyn.KindMap {
			return dyn.InvalidValue, fmt.Errorf("cannot merge map with %s", bk)
		}
		return mergeMap(a, b)
	case dyn.KindSequence:
		if bk != dyn.KindSequence {
			return dyn.InvalidValue, fmt.Errorf("cannot merge sequence with %s", bk)
		}
		return mergeSequence(a, b)
	default:
		if ak != bk {
			return dyn.InvalidValue, fmt.Errorf("cannot merge %s with %s", ak, bk)
		}
		return mergePrimitive(a, b)
	}
}

func mergeMap(a, b dyn.Value) (dyn.Value, error) {
	out := dyn.NewMapping()
	am := a.MustMap()
	bm := b.MustMap()

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
				return dyn.InvalidValue, err
			}
			out.Set(pk, merged)
		} else {
			// Otherwise, just set the value.
			out.Set(pk, pv)
		}
	}

	// Preserve the location of the first value. Accumulate the locations of the second value.
	return mergeLocations(dyn.NewValue(out, a.Locations()), b), nil
}

func mergeSequence(a, b dyn.Value) (dyn.Value, error) {
	as := a.MustSequence()
	bs := b.MustSequence()

	// Merging sequences means concatenating them.
	out := make([]dyn.Value, len(as)+len(bs))
	copy(out[:], as)
	copy(out[len(as):], bs)

	// Preserve the location of the first value. Accumulate the locations of the second value.
	return mergeLocations(dyn.NewValue(out, a.Locations()), b), nil
}

func mergePrimitive(a, b dyn.Value) (dyn.Value, error) {
	// Merging primitive values means using the incoming value. Preserve the
	// location of the first value. Accumulate the locations of the second value.
	return mergeLocations(b, a), nil
}

// This function adds locations associated with the second value to the locations
// associated with the first value. The locations are prepended to preserve the
// "effective" location associated with the first value.
func mergeLocations(v, w dyn.Value) dyn.Value {
	return v.WithLocations(append(w.Locations(), v.Locations()...))
}
