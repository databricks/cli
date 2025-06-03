package structwalk

import (
	"errors"
	"reflect"

	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/cli/libs/structdiff/structtag"
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

	// We're tracking visited and allowing single repeat to support JobSettings.Tasks.ForEachTask.Task
	if visitedCount[typ] >= 2 {
		return
	}

	visitedCount[typ]++

	switch kind {
	case reflect.Pointer:
		walkTypeValue(path, typ.Elem(), visit, visitedCount)

	case reflect.Struct:
		walkTypeStruct(path, typ, visit, visitedCount)

	case reflect.Slice, reflect.Array:
		walkTypeValue(structpath.NewAnyIndex(path), typ.Elem(), visit, visitedCount)

	case reflect.Map:
		if typ.Key().Kind() != reflect.String {
			return // unsupported map key type
		}
		// For maps, we walk the value type directly at the current path
		walkTypeValue(structpath.NewAnyKey(path), typ.Elem(), visit, visitedCount)

	default:
		// func, chan, interface, invalid, etc. -> ignore
	}

	visitedCount[typ]--
}

func walkTypeStruct(path *structpath.PathNode, st reflect.Type, visit VisitTypeFunc, visitedCount map[reflect.Type]int) {
	for i := range st.NumField() {
		sf := st.Field(i)
		if sf.PkgPath != "" {
			continue // unexported
		}
		tag := sf.Tag.Get("json")

		// Handle embedded structs (anonymous fields without json tags)
		if sf.Anonymous && tag == "" {
			// For embedded structs, walk the embedded type at the current path level
			// This flattens the embedded struct's fields into the parent struct
			walkTypeValue(path, sf.Type, visit, visitedCount)
			continue
		}

		if tag == "-" {
			continue // skip fields without json name
		}
		jsonTag := structtag.JSONTag(tag)
		if jsonTag.Name() == "-" {
			continue
		}
		fieldType := sf.Type
		bundleTag := structtag.BundleTag(sf.Tag.Get("bundle"))
		node := structpath.NewStructField(path, jsonTag, bundleTag, sf.Name)
		walkTypeValue(node, fieldType, visit, visitedCount)
	}
}
