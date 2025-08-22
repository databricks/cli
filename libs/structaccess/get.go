package structaccess

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structdiff/structtag"
)

// Get returns the value at the given path inside v.
//
// Path grammar (subset of dyn path):
//   - Keys separated by '.' (e.g., connection.id)
//   - Numeric indices in brackets for arrays/slices (e.g., items[0].name)
//   - Leading '.' is allowed (e.g., .connection.id)
//
// Behavior:
//   - For structs: a key matches a field by its json tag name (if present and not "-").
//     Embedded anonymous structs are searched.
//   - For maps: a key indexes map[string]T (or string alias key types).
//   - For slices/arrays: an index [N] selects the N-th element.
//   - Wildcards ("*" or "[*]") are not supported and return an error.
//
// TODO: embedded struct + FSF needs to be tested and supported
func Get(v any, path string) (any, error) {
	if path == "" {
		return v, nil
	}

	p, err := dyn.NewPathFromString(path)
	if err != nil {
		return nil, err
	}

	cur := reflect.ValueOf(v)
	prefix := ""
	for _, c := range p {
		if c.Key() == "*" {
			return nil, errors.New("wildcard not supported")
		}

		// Dereference pointers and interfaces where possible.
		var ok bool
		cur, ok = deref(cur)
		if !ok {
			loc := prefix
			if loc == "" {
				loc = "(root)"
			}
			return nil, fmt.Errorf("nil found at %s", loc)
		}

		if c.Key() != "" {
			// Key access: struct field (by json tag) or map key.
			nv, err := accessKey(cur, c.Key())
			if err != nil {
				return nil, err
			}
			cur = nv
			if prefix == "" {
				prefix = c.Key()
			} else {
				prefix = prefix + "." + c.Key()
			}
			continue
		}

		// Index access: slice/array
		idx := c.Index()
		kind := cur.Kind()
		if kind != reflect.Slice && kind != reflect.Array {
			return nil, fmt.Errorf("expected slice/array to index [%d], found %s", idx, kind)
		}
		if idx < 0 || idx >= cur.Len() {
			return nil, fmt.Errorf("index out of range [%d] with length %d", idx, cur.Len())
		}
		cur = cur.Index(idx)
		prefix = prefix + "[" + strconv.Itoa(idx) + "]"
	}

	// If the current value is invalid (e.g., omitted due to omitempty), return nil.
	if !cur.IsValid() {
		return nil, nil
	}

	// Return the resulting value as interface{}; do not force dereference of scalars.
	return cur.Interface(), nil
}

// accessKey returns the field or map entry value selected by key from v.
// v must be non-pointer, non-interface reflect.Value.
func accessKey(v reflect.Value, key string) (reflect.Value, error) {
	switch v.Kind() {
	case reflect.Struct:
		fv, sf, ok := findStructFieldByKey(v, key)
		if !ok {
			return reflect.Value{}, fmt.Errorf("field \"%s\" not found in struct %s", key, v.Type())
		}
		// ForceSendFields for nil pointer-to-struct: substitute zero struct.
		if fv.Kind() == reflect.Pointer && fv.IsNil() {
			et := fv.Type().Elem()
			if et.Kind() == reflect.Struct && containsForceSendField(v, sf.Name) {
				return reflect.Zero(et), nil
			}
		}
		// Honor omitempty: if present and value is zero and not forced, treat as omitted (nil).
		jsonTag := structtag.JSONTag(sf.Tag.Get("json"))
		if jsonTag.OmitEmpty() && !containsForceSendField(v, sf.Name) {
			if fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					return reflect.Value{}, nil
				}
				// Non-nil pointer: check the element zero-ness for pointers to scalars/structs.
				if fv.Elem().IsZero() {
					return reflect.Value{}, nil
				}
			} else if fv.IsZero() {
				return reflect.Value{}, nil
			}
		}
		return fv, nil
	case reflect.Map:
		kt := v.Type().Key()
		if kt.Kind() != reflect.String {
			return reflect.Value{}, fmt.Errorf("map key must be string, got %s", kt)
		}
		mk := reflect.ValueOf(key)
		if kt != mk.Type() {
			mk = mk.Convert(kt)
		}
		mv := v.MapIndex(mk)
		if !mv.IsValid() {
			return reflect.Value{}, fmt.Errorf("key \"%s\" not found in map", key)
		}
		return mv, nil
	default:
		return reflect.Value{}, fmt.Errorf("key \"%s\" cannot be applied to %s", key, v.Kind())
	}
}

// findStructFieldByKey searches exported fields of struct v for a field matching key.
// It matches json tag name (when present and not "-") only.
// It also searches embedded anonymous structs (pointer or value) recursively.
func findStructFieldByKey(v reflect.Value, key string) (reflect.Value, reflect.StructField, bool) {
	t := v.Type()

	// First pass: direct fields
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}

		// Read JSON tag using structtag helper
		name := structtag.JSONTag(sf.Tag.Get("json")).Name()
		if name == "-" {
			name = ""
		}

		if name != "" && name == key {
			return v.Field(i), sf, true
		}
	}

	// Second pass: search embedded anonymous structs recursively (flattening semantics)
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.Anonymous {
			continue
		}
		fv := v.Field(i)
		// Dereference pointer anonymous structs
		for fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				// Not initialized; can't descend
				break
			}
			fv = fv.Elem()
		}
		if fv.Kind() != reflect.Struct {
			continue
		}
		if out, osf, ok := findStructFieldByKey(fv, key); ok {
			return out, osf, true
		}
	}

	return reflect.Value{}, reflect.StructField{}, false
}

// containsForceSendField reports whether struct v has a ForceSendFields slice containing goFieldName.
func containsForceSendField(v reflect.Value, goFieldName string) bool {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return false
	}
	fsField := v.FieldByName("ForceSendFields")
	if !fsField.IsValid() || fsField.Kind() != reflect.Slice {
		return false
	}
	for i := range fsField.Len() {
		el := fsField.Index(i)
		if el.Kind() == reflect.String && el.String() == goFieldName {
			return true
		}
	}
	return false
}

// deref dereferences pointers and interfaces until it reaches a non-pointer, non-interface value.
// Returns ok=false if it encounters a nil pointer/interface.
func deref(v reflect.Value) (reflect.Value, bool) {
	for {
		switch v.Kind() {
		case reflect.Pointer:
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		case reflect.Interface:
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		default:
			return v, true
		}
	}
}
