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
//
// Semantics for location metadata in the merged value are similar to the semantics
// for the values themselves:
//
//   - When merging x with nil or nil with x, the location of x is retained.
//
//   - When merging maps or sequences, the combined value retains the location of a and
//     accumulates the location of b. The individual elements of the map or sequence retain
//     their original locations, i.e., whether they were originally defined in a or b.
//
//     The rationale for retaining location of a is that we would like to return
//     the first location a bit of configuration showed up when reporting errors and warnings.
//
//   - Merging primitive values means using the incoming value `b`. The location of the
//     incoming value is retained and the location of the existing value `a` is accumulated.
//     This is because the incoming value overwrites the existing value.
func Merge(a, b dyn.Value) (dyn.Value, error) {
	return merge(a, b)
}

func merge(a, b dyn.Value) (dyn.Value, error) {
	ak := a.Kind()
	bk := b.Kind()

	// If a is nil, return b.
	if ak == dyn.KindNil {
		return b.AppendLocationsFromValue(a), nil
	}

	// If b is nil, return a.
	if bk == dyn.KindNil {
		return a.AppendLocationsFromValue(b), nil
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
		key := pk.MustString()
		pv := pair.Value
		if ov, ok := out.Get(pk); ok {
			// If the key already exists, merge the values.
			merged, err := merge(ov, pv)
			if err != nil {
				return dyn.InvalidValue, err
			}
			out.SetLoc(key, pair.Key.Locations(), merged)
		} else {
			// Otherwise, just set the value.
			out.SetLoc(key, pair.Key.Locations(), pv)
		}
	}

	// Preserve the location of the first value. Accumulate the locations of the second value.
	return dyn.NewValue(out, a.Locations()).AppendLocationsFromValue(b), nil
}

func mergeSequence(a, b dyn.Value) (dyn.Value, error) {
	as := a.MustSequence()
	bs := b.MustSequence()

	// Merging sequences means concatenating them.
	out := make([]dyn.Value, len(as)+len(bs))
	copy(out[:], as)
	copy(out[len(as):], bs)

	// Preserve the location of the first value. Accumulate the locations of the second value.
	return dyn.NewValue(out, a.Locations()).AppendLocationsFromValue(b), nil
}

func mergePrimitive(a, b dyn.Value) (dyn.Value, error) {
	// Merging primitive values means using the incoming value.
	return b.AppendLocationsFromValue(a), nil
}
