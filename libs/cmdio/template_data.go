package cmdio

import (
	"reflect"
	"sync"
)

// templateData returns a view of v for text/template in which pointers to
// scalars are replaced by the values they point to (zero values when nil).
//
// API types represent optional fields as pointers. Templates were written for
// value fields: text/template prints a nil pointer as "<nil>", treats a
// pointer to false or "" as true in {{if}}, and template functions taking an
// int64 reject an *int64. Structs (and slices and maps of them) containing
// such pointers are copied into equivalent struct types with value fields;
// types without them are passed through unchanged and keep their methods.
func templateData(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	return viewValue(rv, viewType(rv.Type())).Interface()
}

func isScalar(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// structView describes the view of a struct type: the view type and, for
// each of its fields, the index path of the source field. Embedded structs
// are flattened into their parent so promoted fields stay accessible by name.
type structView struct {
	typ     reflect.Type
	indexes [][]int
}

var (
	viewTypes   sync.Map // reflect.Type -> reflect.Type
	structViews sync.Map // reflect.Type -> *structView
)

func viewType(t reflect.Type) reflect.Type {
	if cached, ok := viewTypes.Load(t); ok {
		return cached.(reflect.Type)
	}
	// Store the type itself first so recursive types terminate.
	viewTypes.Store(t, t)
	vt := computeViewType(t)
	viewTypes.Store(t, vt)
	return vt
}

func computeViewType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Pointer:
		elem := t.Elem()
		if isScalar(elem.Kind()) {
			return elem
		}
		if ev := viewType(elem); ev != elem {
			return reflect.PointerTo(ev)
		}
	case reflect.Struct:
		if sv := buildStructView(t); sv != nil {
			return sv.typ
		}
	case reflect.Slice:
		if ev := viewType(t.Elem()); ev != t.Elem() {
			return reflect.SliceOf(ev)
		}
	case reflect.Map:
		if ev := viewType(t.Elem()); ev != t.Elem() {
			return reflect.MapOf(t.Key(), ev)
		}
	}
	return t
}

// buildStructView returns the view of struct type t, or nil if no field of t
// (including promoted ones) needs a different type.
func buildStructView(t reflect.Type) *structView {
	var fields []reflect.StructField
	var indexes [][]int
	seen := map[string]bool{}
	changed := false

	var collect func(t reflect.Type, prefix []int)
	collect = func(t reflect.Type, prefix []int) {
		var embedded []reflect.StructField
		for i := range t.NumField() {
			f := t.Field(i)
			index := append(append([]int{}, prefix...), i)
			if f.Anonymous {
				ft := f.Type
				if ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				if ft.Kind() == reflect.Struct {
					f.Index = index
					embedded = append(embedded, f)
					continue
				}
			}
			if !f.IsExported() || seen[f.Name] {
				continue
			}
			seen[f.Name] = true
			ft := viewType(f.Type)
			if ft != f.Type {
				changed = true
			}
			fields = append(fields, reflect.StructField{Name: f.Name, Type: ft, Tag: f.Tag})
			indexes = append(indexes, index)
		}
		// Fields of embedded structs are promoted after the parent's own fields,
		// which take precedence, as in Go.
		for _, f := range embedded {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			collect(ft, f.Index)
		}
	}
	collect(t, nil)

	if !changed {
		return nil
	}
	sv := &structView{typ: reflect.StructOf(fields), indexes: indexes}
	structViews.Store(t, sv)
	return sv
}

func viewValue(v reflect.Value, vt reflect.Type) reflect.Value {
	if v.Type() == vt {
		return v
	}
	switch v.Kind() {
	case reflect.Pointer:
		if vt.Kind() != reflect.Pointer {
			// Pointer to scalar: dereference, or zero value when nil.
			if v.IsNil() {
				return reflect.Zero(vt)
			}
			return v.Elem()
		}
		if v.IsNil() {
			return reflect.Zero(vt)
		}
		out := reflect.New(vt.Elem())
		out.Elem().Set(viewValue(v.Elem(), vt.Elem()))
		return out
	case reflect.Struct:
		cached, _ := structViews.Load(v.Type())
		sv := cached.(*structView)
		out := reflect.New(vt).Elem()
		for i, index := range sv.indexes {
			if src, ok := fieldByIndex(v, index); ok {
				out.Field(i).Set(viewValue(src, vt.Field(i).Type))
			}
		}
		return out
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(vt)
		}
		out := reflect.MakeSlice(vt, v.Len(), v.Len())
		for i := range v.Len() {
			out.Index(i).Set(viewValue(v.Index(i), vt.Elem()))
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(vt)
		}
		out := reflect.MakeMapWithSize(vt, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			out.SetMapIndex(iter.Key(), viewValue(iter.Value(), vt.Elem()))
		}
		return out
	}
	return v
}

// fieldByIndex is reflect.Value.FieldByIndex that reports false instead of
// panicking when it would step through a nil embedded pointer.
func fieldByIndex(v reflect.Value, index []int) (reflect.Value, bool) {
	for i, x := range index {
		if i > 0 && v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		}
		v = v.Field(x)
	}
	return v, true
}
