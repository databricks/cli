package structvar

import (
	"encoding/json"
	"iter"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type option uint8

const (
	// includeZero keeps zero values: the value is behind a pointer, a map value,
	// a slice element, or held in an interface.
	includeZero option = 1 << iota

	// fromInterface is set for values held in an interface (e.g. variable defaults).
	fromInterface
)

// View is a read-only view of a typed value with its references and locations, as the
// configuration tree they describe. It is what [convert.FromTyped] would return for the typed value given
// the source tree as reference, without building that tree: it decides presence the
// same way (zero values are absent unless forced, present in the source, behind a
// pointer, or a map value or slice element) and keeps pure references.
//
// The zero View is invalid and represents an absent value.
type View struct {
	v reflect.Value

	// Locations of the value.
	loc *Locations

	// Path of the value and the references of the whole value the view was created for.
	// The path is only tracked if there are references.
	path *structpath.PathNode
	refs map[string]string

	opts  option
	valid bool
}

// NewView returns the view of the typed value v (typically a pointer to a struct)
// with references refs (keyed by path string) and locations.
func NewView(v any, refs map[string]string, locs *Locations) View {
	return View{v: reflect.ValueOf(v), loc: locs, refs: refs, valid: true}
}

// View returns the view of sv.
func (sv *StructVar) View() View {
	return NewView(sv.Value, sv.Refs, sv.Locations)
}

// ref returns the pure reference held by the value, if its type cannot hold a string.
func (x View) ref() string {
	if len(x.refs) == 0 {
		return ""
	}
	return x.refs[x.path.String()]
}

// child returns a view of the child value v at key k (or index i).
func (x View) childKey(v reflect.Value, k string, opts option) View {
	c := View{v: v, loc: x.loc.Key(k), refs: x.refs, opts: opts, valid: true}
	if len(x.refs) > 0 {
		c.path = structpath.NewStringKey(x.path, k)
	}
	return c
}

func (x View) childIndex(v reflect.Value, i int, opts option) View {
	c := View{v: v, loc: x.loc.Index(i), refs: x.refs, opts: opts, valid: true}
	if len(x.refs) > 0 {
		c.path = structpath.NewIndex(x.path, i)
	}
	return c
}

// IsValid reports whether the value is present.
func (x View) IsValid() bool {
	return x.valid
}

// Locations returns the source locations of the value.
func (x View) Locations() []diag.Location {
	if !x.valid {
		return nil
	}
	return x.loc.Get()
}

// Location returns the first source location of the value.
func (x View) Location() diag.Location {
	locs := x.loc.Get()
	if !x.valid || len(locs) == 0 {
		return diag.Location{}
	}
	return locs[0]
}

// Reflect returns the typed value the view is based on (it may be a pointer or interface).
// It is settable if it was reached through pointers or is the root pointer's element.
func (x View) Reflect() reflect.Value {
	return x.v
}

// Locs returns the locations of this value and the values below it.
func (x View) Locs() *Locations {
	return x.loc
}

// elemOptions returns the options for a map value or a slice element: they keep
// zero values.
func elemOptions(v reflect.Value) option {
	if v.Kind() == reflect.Interface {
		return includeZero | fromInterface
	}
	return includeZero
}

// deref follows pointers and interfaces; it returns an invalid value for nil.
func (x View) deref() (reflect.Value, option) {
	v, opts := x.v, x.opts
	for {
		switch v.Kind() {
		case reflect.Pointer:
			if v.IsNil() {
				return reflect.Value{}, opts
			}
			v = v.Elem()
			// A pointer to a zero value was intentionally set.
			opts |= includeZero
		case reflect.Interface:
			if v.IsNil() {
				return reflect.Value{}, opts
			}
			v = v.Elem()
		default:
			return v, opts
		}
	}
}

// Kind returns the kind of the value in the configuration tree.
func (x View) Kind() Kind {
	if !x.valid {
		return KindInvalid
	}
	v, opts := x.deref()
	if !v.IsValid() {
		return KindNil
	}
	ref := x.ref()
	switch v.Kind() {
	case reflect.Struct:
		if ref != "" {
			return KindString
		}
		if IsSDKNativeType(v.Type()) {
			if _, ok := sdkNativeString(v, opts); ok {
				return KindString
			}
			return KindNil
		}
		// A struct is present if it was present in the source, even if empty.
		if opts&includeZero != 0 || x.loc != nil || x.hasFields(v) {
			return KindMap
		}
		return KindNil
	case reflect.Map:
		if ref != "" {
			return KindString
		}
		if v.IsNil() {
			return KindNil
		}
		return KindMap
	case reflect.Slice:
		if ref != "" {
			return KindString
		}
		if v.IsNil() {
			return KindNil
		}
		return KindSequence
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		if ref != "" {
			return KindString
		}
		// A zero value is absent unless it was present in the source or is kept.
		if v.IsZero() && opts&includeZero == 0 && x.loc == nil {
			return KindNil
		}
		switch v.Kind() {
		case reflect.String:
			return KindString
		case reflect.Bool:
			return KindBool
		case reflect.Float32, reflect.Float64:
			return KindFloat
		default:
			return KindInt
		}
	default:
		return KindInvalid
	}
}

func sdkNativeString(v reflect.Value, opts option) (string, bool) {
	if v.IsZero() && opts&includeZero == 0 {
		return "", false
	}
	// SDK native types implement json.Marshaler and marshal to a JSON string.
	buf, err := json.Marshal(v.Interface())
	if err != nil {
		return "", false
	}
	var s string
	if json.Unmarshal(buf, &s) != nil {
		return "", false
	}
	if s == "" && opts&includeZero == 0 {
		return "", false
	}
	return s, true
}

// isZero mirrors [dyn.Value.IsZero] for the value.
func (x View) isZero() bool {
	switch x.Kind() {
	case KindInvalid, KindNil:
		return true
	case KindMap:
		for range x.MapItems() {
			return false
		}
		return true
	case KindSequence:
		v, _ := x.deref()
		return v.Len() == 0
	default:
		if x.ref() != "" {
			return false
		}
		s, ok := x.AsString()
		if ok {
			return s == ""
		}
		v, _ := x.deref()
		return v.IsZero()
	}
}

// field returns the view of the struct field k, and whether it is represented.
func (x View) field(info *StructInfo, sv reflect.Value, k string, fv reflect.Value, isForced bool) (View, bool) {
	var opts option
	if fv.Kind() == reflect.Interface {
		opts = includeZero | fromInterface
	}
	child := x.childKey(fv, k, opts)
	kind := child.Kind()

	// The field was present in the source (it has a node in the locations tree), it is
	// not zero-valued, or it's forced.
	if child.loc == nil && kind == KindNil && !isForced {
		return View{}, false
	}

	// A zero field is absent unless forced (it may still be a reference: check the view too).
	if fv.Kind() != reflect.Struct && fv.IsZero() && child.isZero() && !info.ForceEmpty[k] && !isForced {
		return View{}, false
	}

	// A forced field with a nil value is represented by its zero value.
	if isForced && kind == KindNil {
		child.opts |= includeZero
	}
	return child, true
}

func (x View) hasFields(v reflect.Value) bool {
	for range x.structFields(v) {
		return true
	}
	return false
}

func (x View) structFields(v reflect.Value) iter.Seq2[string, View] {
	return func(yield func(string, View) bool) {
		info := GetStructInfo(v.Type())
		for _, fv := range info.FieldValues(v) {
			child, ok := x.field(&info, v, fv.Key, fv.Value, fv.IsForced)
			if ok && !yield(fv.Key, child) {
				return
			}
		}
	}
}

// MapItems returns the entries of a map value: struct fields in declaration order,
// map entries in key order.
func (x View) MapItems() iter.Seq2[string, View] {
	return func(yield func(string, View) bool) {
		if x.ref() != "" {
			return
		}
		v, _ := x.deref()
		switch {
		case !v.IsValid():
		case v.Kind() == reflect.Struct && !IsSDKNativeType(v.Type()):
			for k, c := range x.structFields(v) {
				if !yield(k, c) {
					return
				}
			}
		case v.Kind() == reflect.Map:
			keys := v.MapKeys()
			slices.SortFunc(keys, func(a, b reflect.Value) int {
				switch {
				case a.String() < b.String():
					return -1
				case a.String() > b.String():
					return 1
				}
				return 0
			})
			for _, key := range keys {
				ev := v.MapIndex(key)
				child := x.childKey(ev, key.String(), elemOptions(ev))
				if !yield(key.String(), child) {
					return
				}
			}
		}
	}
}

// Sequence returns the elements of a sequence value.
func (x View) Sequence() iter.Seq2[int, View] {
	return func(yield func(int, View) bool) {
		if x.Kind() != KindSequence {
			return
		}
		v, _ := x.deref()
		for i := range v.Len() {
			ev := v.Index(i)
			child := x.childIndex(ev, i, elemOptions(ev))
			if !yield(i, child) {
				return
			}
		}
	}
}

// Get returns the value of key k of a map value, or an invalid value.
func (x View) Get(k string) View {
	if x.Kind() != KindMap {
		return View{}
	}
	v, _ := x.deref()
	switch v.Kind() {
	case reflect.Struct:
		info := GetStructInfo(v.Type())
		index, ok := info.Fields[k]
		if !ok {
			return View{}
		}
		fv := FieldByIndex(v, index)
		if !fv.IsValid() {
			return View{}
		}
		isForced := !fv.IsZero() || info.IsForceSend(v, k)
		child, ok := x.field(&info, v, k, fv, isForced)
		if !ok {
			return View{}
		}
		return child
	case reflect.Map:
		ev := v.MapIndex(reflect.ValueOf(k).Convert(v.Type().Key()))
		if !ev.IsValid() {
			return View{}
		}
		child := x.childKey(ev, k, elemOptions(ev))
		return child
	default:
		return View{}
	}
}

// Index returns element i of a sequence value, or an invalid value.
func (x View) Index(i int) View {
	if x.Kind() != KindSequence {
		return View{}
	}
	v, _ := x.deref()
	if i < 0 || i >= v.Len() {
		return View{}
	}
	ev := v.Index(i)
	child := x.childIndex(ev, i, elemOptions(ev))
	return child
}

// Lookup returns the value at path, or an invalid value if there is none.
func (x View) Lookup(path *structpath.PathNode) View {
	return x.lookupComponents(components(path))
}

func (x View) lookupComponents(path []pathComponent) View {
	for _, c := range path {
		if c.isKey {
			x = x.Get(c.key)
		} else {
			x = x.Index(c.index)
		}
		if !x.valid {
			return x
		}
	}
	return x
}

// AsString returns the string value (a string, a pure reference, or an SDK native value).
func (x View) AsString() (string, bool) {
	if x.Kind() != KindString {
		return "", false
	}
	if ref := x.ref(); ref != "" {
		return ref, true
	}
	v, opts := x.deref()
	if v.Kind() == reflect.Struct {
		return sdkNativeString(v, opts)
	}
	return v.String(), true
}

// AsBool returns the bool value.
func (x View) AsBool() (bool, bool) {
	if x.Kind() != KindBool {
		return false, false
	}
	v, _ := x.deref()
	return v.Bool(), true
}

// AsInt returns the int value.
func (x View) AsInt() (int64, bool) {
	if x.Kind() != KindInt {
		return 0, false
	}
	v, _ := x.deref()
	return v.Int(), true
}

// AsFloat returns the float value.
func (x View) AsFloat() (float64, bool) {
	if x.Kind() != KindFloat {
		return 0, false
	}
	v, _ := x.deref()
	return v.Float(), true
}

// AsAny returns the value as a generic Go value (map[string]any, []any, string,
// bool, int64 or int, float64, or nil), like [dyn.Value.AsAny].
func (x View) AsAny() any {
	switch x.Kind() {
	case KindMap:
		out := map[string]any{}
		for k, c := range x.MapItems() {
			out[k] = c.AsAny()
		}
		return out
	case KindSequence:
		out := []any{} //nolint:gocritic // an empty sequence renders as [], not null
		for _, c := range x.Sequence() {
			out = append(out, c.AsAny())
		}
		return out
	case KindString:
		s, _ := x.AsString()
		return s
	case KindBool:
		b, _ := x.AsBool()
		return b
	case KindInt:
		v, opts := x.deref()
		// Values held in an interface keep the type the YAML loader produced (int),
		// typed fields hold the normalized int64.
		if opts&fromInterface != 0 && v.Type() == reflect.TypeFor[int]() {
			return int(v.Int())
		}
		return v.Int()
	case KindFloat:
		f, _ := x.AsFloat()
		return f
	default:
		return nil
	}
}
