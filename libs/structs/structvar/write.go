package structvar

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

// The methods in this file change sv.Value (a pointer) and keep sv.Refs and
// sv.Locations in sync. Values along the path are allocated as needed (nil pointers,
// nil maps), and map values are written back, so maps of structs can be changed too.
// Refs is replaced rather than modified, so copies of a StructVar can share it.

// Set sets value at path, like setting it in the configuration tree: a zero value
// is kept as explicitly set (it is added to the governing ForceSendFields). The
// locations recorded for path are kept; references at and below path are dropped.
func (sv *StructVar) Set(path *structpath.PathNode, value any) error {
	err := update(reflect.ValueOf(sv.Value), components(path), func(dst reflect.Value) (bool, error) {
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			dst.SetZero()
			return false, nil
		}
		if err := setGoValue(dst, rv); err != nil {
			return false, fmt.Errorf("cannot set %s: %w", path, err)
		}
		return rv.IsZero(), nil
	})
	if err != nil {
		return err
	}
	sv.Refs = withoutRefs(sv.Refs, path)
	// The node marks the value as present even if it is zero (e.g. an empty struct).
	// A leaf node is already what we would write; skip rewriting the tree, which clones
	// every ancestor (e.g. the map of all jobs).
	cur := sv.Locations.Sub(path)
	if value == nil {
		if cur != nil {
			sv.Locations = sv.Locations.With(path, nil)
		}
		return nil
	}
	if cur == nil || len(cur.keys) > 0 || len(cur.elems) > 0 {
		sv.Locations = sv.Locations.With(path, (*Locations)(nil).WithLocations(cur.Get()))
	}
	return nil
}

func setGoValue(dst, src reflect.Value) error {
	for src.Kind() == reflect.Interface || (src.Kind() == reflect.Pointer && dst.Kind() != reflect.Pointer) {
		if src.IsNil() {
			dst.SetZero()
			return nil
		}
		src = src.Elem()
	}
	switch {
	case src.Type().AssignableTo(dst.Type()):
		dst.Set(src)
	case dst.Kind() == reflect.Pointer && src.Type().AssignableTo(dst.Type().Elem()):
		p := reflect.New(dst.Type().Elem())
		p.Elem().Set(src)
		dst.Set(p)
	case isScalarKind(src.Kind()) && isScalarKind(dst.Kind()) && src.Type().ConvertibleTo(dst.Type()) && (src.Kind() == reflect.String) == (dst.Kind() == reflect.String):
		dst.Set(src.Convert(dst.Type()))
	default:
		return fmt.Errorf("cannot assign %s to %s", src.Type(), dst.Type())
	}
	return nil
}

func isScalarKind(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// Delete removes the value at path: a map entry is deleted, a struct field is zeroed.
func (sv *StructVar) Delete(path *structpath.PathNode) error {
	p := components(path)
	if len(p) == 0 {
		return errors.New("cannot delete the root")
	}
	parent, last := p[:len(p)-1], p[len(p)-1]
	err := walkTo(reflect.ValueOf(sv.Value), parent, func(v reflect.Value) error {
		v = derefAlloc(v)
		switch {
		case v.Kind() == reflect.Map && last.isKey:
			if !v.IsNil() {
				v.SetMapIndex(reflect.ValueOf(last.key).Convert(v.Type().Key()), reflect.Value{})
			}
		case v.Kind() == reflect.Struct && last.isKey:
			info := GetStructInfo(v.Type())
			index, ok := info.Fields[last.key]
			if !ok {
				return fmt.Errorf("no field %q at %s", last.key, path.Parent())
			}
			if f := FieldByIndex(v, index); f.IsValid() {
				f.SetZero()
			}
			removeForceSend(v, &info, last.key)
		default:
			return fmt.Errorf("cannot delete %s", path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sv.Refs = withoutRefs(sv.Refs, path)
	sv.Locations = sv.Locations.With(path, nil)
	return nil
}

// SetLocations sets the locations of the value at path and of every value below it,
// including values that have no locations yet (e.g. set in Go code).
func (sv *StructVar) SetLocations(path *structpath.PathNode, locs []diag.Location) {
	x := sv.View().Lookup(path)
	if !x.IsValid() {
		return
	}
	sv.Locations = sv.Locations.With(path, locationsOf(x, locs))
}

func locationsOf(x View, locs []diag.Location) *Locations {
	out := (*Locations)(nil).WithLocations(locs)
	switch x.Kind() {
	case KindMap:
		for k, c := range x.MapItems() {
			out.setChild(pathComponent{key: k, isKey: true}, locationsOf(c, locs))
		}
	case KindSequence:
		for i, c := range x.Sequence() {
			out.setChild(pathComponent{index: i}, locationsOf(c, locs))
		}
	default:
	}
	return out
}

// UpdateSequence records that the elements of the sequence at path were rebuilt from
// the old ones: new element i comes from old elements sources[i] (see
// [Locations.WithSequence]). References of the old elements move along.
func (sv *StructVar) UpdateSequence(path *structpath.PathNode, sources [][]int) {
	sv.Locations = sv.Locations.WithSequence(path, sources)
	if len(sv.Refs) == 0 {
		return
	}
	elems := path.String() + "["
	refs := make(map[string]string, len(sv.Refs))
	for k, v := range sv.Refs {
		if !strings.HasPrefix(k, elems) {
			refs[k] = v
		}
	}
	for i, src := range sources {
		to := structpath.NewIndex(path, i).String()
		for _, old := range src {
			from := structpath.NewIndex(path, old).String()
			for k, v := range sv.Refs {
				if isUnder(k, from) {
					refs[to+strings.TrimPrefix(k, from)] = v
				}
			}
		}
	}
	sv.Refs = refs
}

// isUnder reports whether the path string key is prefix or a path below it.
func isUnder(key, prefix string) bool {
	if prefix == "" || key == prefix {
		return true
	}
	if !strings.HasPrefix(key, prefix) {
		return false
	}
	c := key[len(prefix)]
	return c == '.' || c == '['
}

// withoutRefs returns a copy of refs without the references at and below path.
func withoutRefs(refs map[string]string, path *structpath.PathNode) map[string]string {
	if len(refs) == 0 {
		return refs
	}
	prefix := path.String()
	out := make(map[string]string, len(refs))
	for k, v := range refs {
		if !isUnder(k, prefix) {
			out[k] = v
		}
	}
	return out
}

// SetReference sets the value at path to the reference ref. A string (or interface)
// field holds the reference itself; for other fields the typed value is set to zero,
// like decoding does, and the reference is recorded in Refs.
func (sv *StructVar) SetReference(path *structpath.PathNode, ref string) error {
	isText := false
	err := walkTo(reflect.ValueOf(sv.Value), components(path), func(dst reflect.Value) error {
		switch dst.Kind() {
		case reflect.String, reflect.Interface:
			isText = true
		case reflect.Pointer:
			// The reference is recorded on an allocated zero value, so it is present.
			dst.Set(reflect.New(dst.Type().Elem()))
		default:
			dst.SetZero()
		}
		return nil
	})
	if err != nil {
		return err
	}
	if isText {
		return sv.Set(path, ref)
	}
	locs := sv.Locations.At(path)
	// withoutRefs returns a fresh map unless there is nothing to filter.
	refs := withoutRefs(sv.Refs, path)
	if len(refs) == 0 {
		refs = map[string]string{}
	}
	refs[path.String()] = ref
	sv.Refs = refs
	sv.Locations = sv.Locations.With(path, (*Locations)(nil).WithLocations(locs))
	return nil
}

func addForceSend(v reflect.Value, info *StructInfo, k string) {
	index, ok := info.ForceSendFieldsIndex[k]
	if !ok {
		return
	}
	fsf := GetOrNewFieldByIndex(v, index)
	name := info.GolangNames[k]
	if slices.Contains(fsf.Interface().([]string), name) {
		return
	}
	fsf.Set(reflect.Append(fsf, reflect.ValueOf(name)))
}

func removeForceSend(v reflect.Value, info *StructInfo, k string) {
	index, ok := info.ForceSendFieldsIndex[k]
	if !ok {
		return
	}
	fsf := FieldByIndex(v, index)
	if !fsf.IsValid() {
		return
	}
	name := info.GolangNames[k]
	names := fsf.Interface().([]string)
	if i := slices.Index(names, name); i >= 0 {
		fsf.Set(reflect.ValueOf(slices.Delete(slices.Clone(names), i, i+1)))
	}
}

// derefAlloc follows pointers, allocating nil ones, and returns the value they point to.
func derefAlloc(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

// walkTo calls fn with the settable value at path.
func walkTo(v reflect.Value, path []pathComponent, fn func(reflect.Value) error) error {
	if len(path) == 0 {
		return fn(v)
	}
	return updateChild(derefAlloc(v), path[0], func(child reflect.Value) (bool, error) {
		return false, walkTo(child, path[1:], fn)
	})
}

// update calls fn with the settable value at path. If fn reports that it set a zero
// value into a struct field, the field is added to the governing ForceSendFields.
func update(v reflect.Value, path []pathComponent, fn func(reflect.Value) (bool, error)) error {
	if len(path) == 0 {
		if v.Kind() == reflect.Pointer && !v.CanSet() {
			v = v.Elem()
		}
		_, err := fn(v)
		return err
	}
	return walkTo(v, path[:len(path)-1], func(parent reflect.Value) error {
		return updateChild(derefAlloc(parent), path[len(path)-1], fn)
	})
}

// updateChild calls fn with the settable child c of v. Interfaces and map values are
// copied and written back; an index one past the end of a slice appends an element.
func updateChild(v reflect.Value, c pathComponent, fn func(reflect.Value) (bool, error)) error {
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return errors.New("cannot index nil value")
		}
		cp := reflect.New(v.Elem().Type()).Elem()
		cp.Set(v.Elem())
		err := updateChild(derefAlloc(cp), c, fn)
		v.Set(cp)
		return err
	}

	switch {
	case c.isKey && v.Kind() == reflect.Struct:
		info := GetStructInfo(v.Type())
		index, ok := info.Fields[c.key]
		if !ok {
			return fmt.Errorf("no field %q", c.key)
		}
		zero, err := fn(GetOrNewFieldByIndex(v, index))
		if err != nil {
			return err
		}
		if zero {
			addForceSend(v, &info, c.key)
		}
		return nil
	case c.isKey && v.Kind() == reflect.Map:
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
		key := reflect.ValueOf(c.key).Convert(v.Type().Key())
		e := reflect.New(v.Type().Elem()).Elem()
		if old := v.MapIndex(key); old.IsValid() {
			e.Set(old)
		}
		_, err := fn(e)
		v.SetMapIndex(key, e)
		return err
	case !c.isKey && v.Kind() == reflect.Slice:
		if c.index == v.Len() {
			v.Set(reflect.Append(v, reflect.New(v.Type().Elem()).Elem()))
		}
		if c.index < 0 || c.index >= v.Len() {
			return fmt.Errorf("index %d out of range", c.index)
		}
		_, err := fn(v.Index(c.index))
		return err
	default:
		return fmt.Errorf("cannot index %s", v.Kind())
	}
}
