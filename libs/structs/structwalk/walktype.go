package structwalk

import (
	"errors"
	"reflect"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

// VisitTypeFunc is invoked for fields encountered while walking typ. This includes both leaf nodes as well as any
// intermediate nodes encountered while walking the struct tree.
//
//   path         PathNode representing the JSON-style path to the field.
//   typ          the field's type – if the field is a pointer to a scalar the pointer type is preserved;
//                the callback receives the actual type (e.g., *string, *int, etc.).
//   field        the struct field if this node represents a struct field, nil otherwise.
//
// The function returns a boolean:
//   continueWalk: if true, the WalkType function will continue recursively walking the current field.
//                 if false, the WalkType function will skip walking the current field and all its children.
//
// NOTE: Fields lacking a json tag or tagged as "-" are ignored entirely.
//       Dynamic types like func, chan, interface, etc. are *not* visited.
//       Only maps with string keys are traversed so that paths stay JSON-like.
//
// The walk is depth-first and deterministic (map keys are sorted lexicographically).
//
// Example:
//   err := structwalk.WalkType(reflect.TypeOf(cfg), func(path *structpath.PathNode, typ reflect.Type, field *reflect.StructField) {
//       fmt.Printf("%s = %v\n", path.String(), typ)
//   })
//
// ******************************************************************************************************

type VisitTypeFunc func(path *structpath.PathNode, typ reflect.Type, field *reflect.StructField) (continueWalk bool)

// WalkType validates that t is a struct or pointer to one and starts the recursive traversal.
func WalkType(t reflect.Type, visit VisitTypeFunc) error {
	if visit == nil {
		return errors.New("structwalk: visit callback must not be nil")
	}
	if t == nil {
		return nil
	}
	visitedCount := make(map[reflect.Type]int)
	walkTypeValue(nil, t, nil, visit, visitedCount)
	return nil
}

func walkTypeValue(path *structpath.PathNode, typ reflect.Type, field *reflect.StructField, visit VisitTypeFunc, visitedCount map[reflect.Type]int) {
	if typ == nil {
		return
	}

	// Call visit on all nodes including the root node. We call visit before
	// dereferencing pointers to ensure that the visit callback receives
	// the actual type of the field.
	continueWalk := visit(path, typ, field)
	if !continueWalk {
		return
	}

	// Dereference pointers.
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	// Return early if we're at a leaf scalar.
	kind := typ.Kind()
	if isScalar(kind) {
		return
	}

	// We're tracking visited and allowing single repeat to support JobSettings.Tasks.ForEachTask.Task
	if visitedCount[typ] >= 2 {
		return
	}

	visitedCount[typ]++

	switch kind {
	case reflect.Struct:
		walkTypeStruct(path, typ, visit, visitedCount)

	case reflect.Slice, reflect.Array:
		walkTypeValue(structpath.NewBracketStar(path), typ.Elem(), nil, visit, visitedCount)

	case reflect.Map:
		if typ.Key().Kind() != reflect.String {
			return // unsupported map key type
		}
		// For maps, we walk the value type directly at the current path
		walkTypeValue(structpath.NewDotStar(path), typ.Elem(), nil, visit, visitedCount)

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

		// Handle embedded structs (anonymous fields without json tags)
		jsonTag := sf.Tag.Get("json")
		if sf.Anonymous && jsonTag == "" {
			// For embedded structs, walk the embedded type at the current path level
			// This flattens the embedded struct's fields into the parent struct
			walkTypeValue(path, sf.Type, &sf, visit, visitedCount)
			continue
		}

		// Skip fields marked as "-" in json tag
		jsonTagName := structtag.JSONTag(jsonTag).Name()
		if jsonTagName == "-" {
			continue
		}

		// Resolve field name from JSON tag or fall back to Go field name
		fieldName := jsonTagName
		if fieldName == "" {
			fieldName = sf.Name
		}
		node := structpath.NewDotString(path, fieldName)
		walkTypeValue(node, sf.Type, &sf, visit, visitedCount)
	}
}
