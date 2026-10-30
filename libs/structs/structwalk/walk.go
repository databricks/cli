package structwalk

import (
	"errors"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// VisitFunc is invoked for every scalar (int, uint, float, string, bool) field encountered while walking v.
//
//   path         PathNode representing the JSON-style path to the field.
//   val          the field's value – if the field is a pointer to a scalar the pointer is *not* dereferenced; the
//                callback receives either nil (for a nil pointer) or the concrete value.
//   field        the reflect.StructField for struct fields, nil for map keys and array indices.
//
// NOTE: Fields lacking a json tag or tagged as "-" are ignored entirely.
//       Composite kinds (struct, slice/array, map, interface, function, chan, etc.) are *not* visited, but the walk
//       traverses them to reach nested scalar fields (except interface & func). Only maps with string keys are
//       traversed so that paths stay JSON-like.
//
// The walk is depth-first and deterministic (map keys are sorted lexicographically).
//
// Example:
//   err := structwalk.Walk(cfg, func(path *structpath.PathNode, v any, field *reflect.StructField) {
//       fmt.Printf("%s = %v\n", path.String(), v)
//   })
//
// ******************************************************************************************************

type VisitFunc func(path *structpath.PathNode, val any, field *reflect.StructField)

// Walk validates that v is a struct or pointer to one and starts the recursive traversal.
func Walk(v any, visit VisitFunc) error {
	if visit == nil {
		return errors.New("structwalk: visit callback must not be nil")
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}
	walkValue(nil, rv, nil, visit)
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

func walkValue(path *structpath.PathNode, val reflect.Value, field *reflect.StructField, visit VisitFunc) {
	kind := val.Kind()

	if isScalar(kind) {
		visit(path, val.Interface(), field)
		return
	}

	switch kind {
	case reflect.Pointer:
		walkValue(path, val.Elem(), field, visit)

	case reflect.Struct:
		walkStruct(path, val, visit)

	case reflect.Slice, reflect.Array:
		for i := range val.Len() {
			node := structpath.NewIndex(path, i)
			walkValue(node, val.Index(i), nil, visit)
		}

	case reflect.Map:
		if val.Type().Key().Kind() != reflect.String {
			return // unsupported map key type
		}
		var keys []string
		for _, k := range val.MapKeys() {
			keys = append(keys, k.String())
		}
		slices.Sort(keys)
		for _, ks := range keys {
			v := val.MapIndex(reflect.ValueOf(ks))
			node := structpath.NewBracketString(path, ks)
			walkValue(node, v, nil, visit)
		}

	default:
		// func, chan, interface, invalid, etc. -> ignore
	}
}

func walkStruct(path *structpath.PathNode, s reflect.Value, visit VisitFunc) {
	info := structaccess.GetStructInfo(s.Type())
	forced := info.ForceSendFields(s)

	for _, f := range info.Fields {
		// A copy, so a visitor cannot modify the cached field.
		sf := f.StructField
		fieldVal := s.Field(f.Index)

		// Directly walk into embedded structs without adding the key to the path.
		if f.Flattened {
			walkValue(path, fieldVal, &sf, visit)
			continue
		}

		// Skip zero values with omitempty unless field is explicitly forced.
		if f.OmitEmpty && fieldVal.IsZero() && !slices.Contains(forced, sf.Name) {
			continue
		}

		// EmbeddedSlice: walk directly without adding the field name to the path.
		if f.EmbeddedSlice {
			walkValue(path, fieldVal, &sf, visit)
			continue
		}

		node := structpath.NewDotString(path, f.Name)
		walkValue(node, fieldVal, &sf, visit)
	}
}
