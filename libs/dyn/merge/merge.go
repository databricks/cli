package merge

import (
	"fmt"
	"slices"

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
	return merge(a, b, nullIsIgnored)
}

// MergeKeepingNulls is like [Merge], except that a nil in b replaces the value in a.
// The nils in b are kept in the result.
func MergeKeepingNulls(a, b dyn.Value) (dyn.Value, error) {
	return merge(a, b, nullReplaces)
}

// MergeRemovingNulls is like [Merge], except that a nil map value in b removes the
// same key from a. The nil map values of b are not added to the result, also where a
// has no value to remove (see [dyn.DropNulls]). A nil b leaves a unchanged.
func MergeRemovingNulls(a, b dyn.Value) (dyn.Value, error) {
	return merge(a, b, nullRemoves)
}

// nullMode is what a nil in the second value does to the first one.
type nullMode int

const (
	nullIsIgnored nullMode = iota
	nullReplaces
	nullRemoves
)

func merge(a, b dyn.Value, mode nullMode) (dyn.Value, error) {
	ak := a.Kind()
	bk := b.Kind()

	// If b is nil and replaces a, return b.
	if bk == dyn.KindNil && mode == nullReplaces {
		return b, nil
	}

	// If a is nil, return b.
	if ak == dyn.KindNil {
		if mode == nullRemoves {
			b = dyn.DropNulls(b)
		}
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
		return mergeMap(a, b, mode)
	case dyn.KindSequence:
		if bk != dyn.KindSequence {
			return dyn.InvalidValue, fmt.Errorf("cannot merge sequence with %s", bk)
		}
		return mergeSequence(a, b, mode)
	default:
		if ak != bk {
			return dyn.InvalidValue, fmt.Errorf("cannot merge %s with %s", ak, bk)
		}
		return mergePrimitive(a, b)
	}
}

func mergeMap(a, b dyn.Value, mode nullMode) (dyn.Value, error) {
	out := dyn.NewMapping()
	am := a.MustMap()
	bm := b.MustMap()

	// Add the values from a into the output map.
	out.Merge(am)

	// Merge the values from b into the output map.
	var removed []string
	for _, pair := range bm.Pairs() {
		pk := pair.Key
		key := pk.MustString()
		pv := pair.Value
		if mode == nullRemoves && pv.Kind() == dyn.KindNil {
			removed = append(removed, key)
			continue
		}
		if ov, ok := out.Get(pk); ok {
			// If the key already exists, merge the values.
			merged, err := merge(ov, pv, mode)
			if err != nil {
				return dyn.InvalidValue, err
			}
			out.SetLoc(key, pair.Key.Locations(), merged)
		} else {
			// Otherwise, just set the value.
			if mode == nullRemoves {
				pv = dyn.DropNulls(pv)
			}
			out.SetLoc(key, pair.Key.Locations(), pv)
		}
	}

	if len(removed) > 0 {
		out = withoutKeys(out, removed)
	}

	// Preserve the location of the first value. Accumulate the locations of the second value.
	return dyn.NewValue(out, a.Locations()).AppendLocationsFromValue(b), nil
}

func mergeSequence(a, b dyn.Value, mode nullMode) (dyn.Value, error) {
	as := a.MustSequence()
	bs := b.MustSequence()
	if mode == nullRemoves {
		bs = dyn.DropNulls(b).MustSequence()
	}

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

// withoutKeys returns m without the given keys.
func withoutKeys(m dyn.Mapping, keys []string) dyn.Mapping {
	out := dyn.NewMapping()
	for _, pair := range m.Pairs() {
		key := pair.Key.MustString()
		if !slices.Contains(keys, key) {
			out.SetLoc(key, pair.Key.Locations(), pair.Value)
		}
	}
	return out
}
