package structwalk

import (
	"errors"
	"reflect"
	"sort"

	"github.com/databricks/cli/libs/structdiff/jsontag"
	"github.com/databricks/cli/libs/structdiff/structpath"
)

// VisitFunc is invoked for every scalar (int, uint, float, string, bool) field encountered while walking v.
//
//   path         PathNode representing the JSON-style path to the field.
//   val          the field's value – if the field is a pointer to a scalar the pointer is *not* dereferenced; the
//                callback receives either nil (for a nil pointer) or the concrete value.
//
// NOTE: Fields lacking a json tag or tagged as "-" are ignored entirely.
//       Composite kinds (struct, slice/array, map, interface, function, chan, etc.) are *not* visited, but the walk
//       traverses them to reach nested scalar fields (except interface & func). Only maps with string keys are
//       traversed so that paths stay JSON-like.
//
// The walk is depth-first and deterministic (map keys are sorted lexicographically).
// A non-nil error stops the traversal immediately.
//
// Example:
//   err := structwalk.Walk(cfg, func(path *structpath.PathNode, v any) {
//       fmt.Printf("%s = %v\n", path.String(), v)
//   })
//
// ******************************************************************************************************

type VisitFunc func(path *structpath.PathNode, val any)

// Walk validates that v is a struct or pointer to one and starts the recursive traversal.
func Walk(v any, visit VisitFunc) error {
	if visit == nil {
		return errors.New("structwalk: visit callback must not be nil")
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	walkValue(nil, rv, visit)
	return nil
}

// isScalar reports whether kind is considered scalar for our purposes.
func isScalar(k reflect.Kind) bool {
	switch k {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	default:
		return false
	}
}

func walkValue(path *structpath.PathNode, val reflect.Value, visit VisitFunc) {
	if !val.IsValid() {
		return
	}
	kind := val.Kind()

	if isScalar(kind) {
		// Primitive scalar at the leaf – invoke.
		visit(path, val.Interface())
		return
	}

	switch kind {
	case reflect.Pointer:
		// Pointer – treat pointer itself as scalar? We choose to surface pointer *value* (nil / underlying scalar).
		// If element is scalar we still want to report it; if element is composite we drill down.
		if val.IsNil() {
			elemKind := val.Type().Elem().Kind()
			if isScalar(elemKind) {
				visit(path, nil)
			}
			return
		}
		elemKind := val.Type().Elem().Kind()
		if isScalar(elemKind) {
			visit(path, val.Elem().Interface())
			return
		}
		walkValue(path, val.Elem(), visit)

	case reflect.Struct:
		walkStruct(path, val, visit)

	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			node := structpath.NewIndex(path, i)
			walkValue(node, val.Index(i), visit)
		}

	case reflect.Map:
		if val.Type().Key().Kind() != reflect.String {
			return // unsupported map key type
		}
		var keys []string
		for _, k := range val.MapKeys() {
			keys = append(keys, k.String())
		}
		sort.Strings(keys)
		for _, ks := range keys {
			v := val.MapIndex(reflect.ValueOf(ks))
			node := structpath.NewMapKey(path, ks)
			walkValue(node, v, visit)
		}

	default:
		// func, chan, interface, invalid, etc. -> ignore
	}
}

func walkStruct(path *structpath.PathNode, s reflect.Value, visit VisitFunc) {
	st := s.Type()
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		if sf.PkgPath != "" {
			continue // unexported
		}
		tag := sf.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue // skip fields without json name
		}
		jsonTag := jsontag.JSONTag(tag)
		if jsonTag.Name() == "-" {
			continue
		}
		fieldVal := s.Field(i)
		node := structpath.NewStructField(path, jsonTag, sf.Name)
		walkValue(node, fieldVal, visit)
	}
}
