package structwalk

import (
	"errors"
	"reflect"

	"github.com/databricks/cli/libs/structdiff/jsontag"
	"github.com/databricks/cli/libs/structdiff/structpath"
)

// VisitTypeFunc is invoked for every scalar (int, uint, float, string, bool) field type encountered while walking t.
//
//   path         PathNode representing the JSON-style path to the field.
//   typ          the field's type – if the field is a pointer to a scalar the pointer type is preserved;
//                the callback receives the actual type (e.g., *string, *int, etc.).
//
// NOTE: Fields lacking a json tag or tagged as "-" are ignored entirely.
//       Composite kinds (struct, slice/array, map, interface, function, chan, etc.) are *not* visited, but the walk
//       traverses them to reach nested scalar field types (except interface & func). Only maps with string keys are
//       traversed so that paths stay JSON-like.
//
// The walk is depth-first and deterministic (map keys are sorted lexicographically).
//
// Example:
//   err := structwalk.WalkType(reflect.TypeOf(cfg), func(path *structpath.PathNode, t reflect.Type) {
//       fmt.Printf("%s = %v\n", path.String(), t)
//   })
//
// ******************************************************************************************************

type VisitTypeFunc func(path *structpath.PathNode, typ reflect.Type)

// WalkType validates that t is a struct or pointer to one and starts the recursive traversal.
func WalkType(t reflect.Type, visit VisitTypeFunc) error {
	if visit == nil {
		return errors.New("structwalk: visit callback must not be nil")
	}
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// Use a visited counter to allow one level of circular reference
	// Stop only when we see a type for the second time
	visitedCount := make(map[reflect.Type]int)
	walkTypeValue(nil, t, visit, visitedCount)
	return nil
}

func walkTypeValue(path *structpath.PathNode, typ reflect.Type, visit VisitTypeFunc, visitedCount map[reflect.Type]int) {
	if typ == nil {
		return
	}
	kind := typ.Kind()

	if isScalar(kind) {
		// Primitive scalar at the leaf – invoke.
		visit(path, typ)
		return
	}

	switch kind {
	case reflect.Pointer:
		// Pointer – treat pointer itself as scalar? We choose to surface pointer *type* (including pointer types to scalars).
		// If element is scalar we still want to report it; if element is composite we drill down.
		elemKind := typ.Elem().Kind()
		if isScalar(elemKind) {
			visit(path, typ)
			return
		}
		// For pointers to structs, check circular reference here
		if elemKind == reflect.Struct {
			elemType := typ.Elem()
			if visitedCount[elemType] >= 1 {
				return // Skip types we've already seen once to prevent infinite recursion
			}
			visitedCount[elemType]++
			walkTypeValue(path, elemType, visit, visitedCount)
			visitedCount[elemType]-- // Decrement after processing to allow same type in different branches
		} else {
			walkTypeValue(path, typ.Elem(), visit, visitedCount)
		}

	case reflect.Struct:
		walkTypeStruct(path, typ, visit, visitedCount)

	case reflect.Slice, reflect.Array:
		// For slices and arrays, we walk the element type
		walkTypeValue(structpath.NewIndex(path, 0), typ.Elem(), visit, visitedCount)

	case reflect.Map:
		if typ.Key().Kind() != reflect.String {
			return // unsupported map key type
		}
		// For maps, we walk the value type directly at the current path
		walkTypeValue(path, typ.Elem(), visit, visitedCount)

	default:
		// func, chan, interface, invalid, etc. -> ignore
	}
}

func walkTypeStruct(path *structpath.PathNode, st reflect.Type, visit VisitTypeFunc, visitedCount map[reflect.Type]int) {
	for i := range st.NumField() {
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
		fieldType := sf.Type
		node := structpath.NewStructField(path, jsonTag, sf.Name)
		walkTypeValue(node, fieldType, visit, visitedCount)
	}
}
