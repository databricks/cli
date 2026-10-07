package structvar

import (
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/structs/structpath"
)

// Merge merges the value described by src into the value at path, with the semantics
// of merge.Merge: maps are merged recursively, sequences are concatenated, primitive
// values are replaced, and a nil value on either side yields the other one.
// Locations accumulate like merge.Merge accumulates them. A failed merge changes nothing.
func (sv *StructVar) Merge(path *structpath.PathNode, src View) error {
	dstView := sv.View().Lookup(path)
	if err := CheckMerge(dstView, src); err != nil {
		return err
	}

	m := merger{refs: maps.Clone(sv.Refs)}
	var locs *Locations
	err := update(reflect.ValueOf(sv.Value), components(path), func(dst reflect.Value) (bool, error) {
		dstView.v = dst
		var err error
		locs, err = m.merge(dst, dstView, src, path)
		return isZeroScalar(src), err
	})
	if err != nil {
		return err
	}
	sv.Refs = m.refs
	sv.Locations = sv.Locations.With(path, locs)
	return nil
}

// isZeroScalar reports whether src is a present zero scalar, which must be kept as an
// explicitly set value.
func isZeroScalar(src View) bool {
	k := src.Kind()
	return k != KindNil && k != KindMap && k != KindSequence && src.isZero()
}

// CheckMerge returns the error [StructVar.Merge] would return for merging src into dst.
func CheckMerge(dst, src View) error {
	ak, bk := dst.Kind(), src.Kind()
	if ak == KindInvalid || ak == KindNil || bk == KindInvalid || bk == KindNil {
		return nil
	}
	switch ak {
	case KindMap:
		if bk != KindMap {
			return fmt.Errorf("cannot merge map with %s", bk)
		}
		for k, sc := range src.MapItems() {
			if err := CheckMerge(dst.Get(k), sc); err != nil {
				return err
			}
		}
		return nil
	case KindSequence:
		if bk != KindSequence {
			return fmt.Errorf("cannot merge sequence with %s", bk)
		}
		return nil
	default:
		if ak != bk {
			return fmt.Errorf("cannot merge %s with %s", ak, bk)
		}
		return nil
	}
}

type merger struct {
	refs map[string]string
}

// assign sets dst to src (see [StructVar.Assign]) and returns its locations.
func (m *merger) assign(dst reflect.Value, src View, path *structpath.PathNode) (*Locations, error) {
	var d decoder
	n, ok, err := d.decode(dst, viewSource{src}, path)
	if err != nil {
		return nil, err
	}
	if !ok {
		dst.SetZero()
	}
	m.refs = n.refs(path, withoutRefs(m.refs, path))
	return n.locations(), nil
}

// merge merges src into dst (settable; dstView is its view, invalid if absent).
func (m *merger) merge(dst reflect.Value, dstView, src View, path *structpath.PathNode) (*Locations, error) {
	ak, bk := dstView.Kind(), src.Kind()
	switch {
	case bk == KindInvalid:
		return dstView.loc, nil
	case ak == KindInvalid:
		return m.assign(dst, src, path)
	case ak == KindNil:
		locs, err := m.assign(dst, src, path)
		return locs.WithLocations(concatLocations(src.loc.Get(), dstView.loc.Get())), err
	case bk == KindNil:
		return dstView.loc.WithLocations(concatLocations(dstView.loc.Get(), src.loc.Get())), nil
	}

	switch ak {
	case KindMap:
		out := dstView.loc.WithLocations(concatLocations(dstView.loc.Get(), src.loc.Get()))
		v := derefAlloc(dst)
		for k, sc := range src.MapItems() {
			c := pathComponent{key: k, isKey: true}
			cp := structpath.NewStringKey(path, k)
			var locs *Locations
			err := updateChild(v, c, func(child reflect.Value) (bool, error) {
				childView := dstView.Get(k)
				childView.v = child
				var err error
				locs, err = m.merge(child, childView, sc, cp)
				return isZeroScalar(sc), err
			})
			if err != nil {
				return nil, err
			}
			out.setChild(c, locs)
		}
		return out, nil
	case KindSequence:
		out := dstView.loc.WithLocations(concatLocations(dstView.loc.Get(), src.loc.Get()))
		v := derefAlloc(dst)
		holder := v
		if v.Kind() == reflect.Interface {
			// A sequence held in an interface (e.g. a complex variable default) is
			// appended to as a copy and stored back.
			v = reflect.New(holder.Elem().Type()).Elem()
			v.Set(holder.Elem())
		}
		for _, sc := range src.Sequence() {
			i := v.Len()
			e := reflect.New(v.Type().Elem()).Elem()
			locs, err := m.assign(e, sc, structpath.NewIndex(path, i))
			if err != nil {
				return nil, err
			}
			v.Set(reflect.Append(v, e))
			out.setChild(pathComponent{index: i}, locs)
		}
		if holder.Kind() == reflect.Interface {
			holder.Set(v)
		}
		return out, nil
	default:
		locs, err := m.assign(dst, src, path)
		return locs.WithLocations(concatLocations(src.loc.Get(), dstView.loc.Get())), err
	}
}

// MergeElementsByKey merges the elements of the sequence at path that have the same
// key, like merge.ElementsByKey: elements are merged into the first one with the same
// key (in order), the key field of every element is set to its key, and the keys are
// sorted if sortKeys is set.
func (sv *StructVar) MergeElementsByKey(path *structpath.PathNode, keyField string, keyFn func(View) string, sortKeys bool) error {
	seq := sv.View().Lookup(path)
	if seq.Kind() != KindSequence {
		return nil
	}

	var keys []string
	groups := map[string][]int{}
	for i, e := range seq.Sequence() {
		k := keyFn(e.Get(keyField))
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], i)
	}
	if len(keys) == 0 {
		return nil
	}
	if sortKeys {
		slices.Sort(keys)
	}

	// Build the merged sequence separately: the first element of each group is
	// assigned, the others are merged into it.
	sliceType := seq.Reflect().Type()
	for sliceType.Kind() == reflect.Pointer {
		sliceType = sliceType.Elem()
	}
	tmp := &StructVar{Value: reflect.New(sliceType).Interface()}
	for i, k := range keys {
		elem := structpath.NewIndex(nil, i)
		for j, idx := range groups[k] {
			var err error
			if j == 0 {
				_, err = tmp.Assign(elem, seq.Index(idx))
			} else {
				err = tmp.Merge(elem, seq.Index(idx))
			}
			if err != nil {
				return err
			}
		}
		// The key field holds the key (e.g. a normalized one), keeping its location.
		if err := tmp.Set(structpath.NewStringKey(elem, keyField), k); err != nil {
			return err
		}
	}
	tmp.Locations = tmp.Locations.WithLocations(seq.loc.Get())
	_, err := sv.Assign(path, tmp.View())
	return err
}
