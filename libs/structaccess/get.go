package structaccess

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structdiff/structtag"
)

// Get returns the value at the given path inside v.
//
// Path grammar (subset of dyn path):
//   - Struct field names and map keys separated by '.' (e.g., connection.id)
//   - (Note, this prevents maps keys that are not id-like from being referenced, but this general problem with references today.)
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
		var ok bool
		cur, ok = deref(cur)
		if !ok {
			// cannot proceed further due to nil encountered at current location
			return nil, fmt.Errorf("%s: cannot access nil value", prefix)
		}

		if c.Key() != "" {
			// Key access: struct field (by json tag) or map key.
			newPrefix := prefix
			if newPrefix == "" {
				newPrefix = c.Key()
			} else {
				newPrefix = newPrefix + "." + c.Key()
			}
			nv, err := accessKey(cur, c.Key(), newPrefix)
			if err != nil {
				return nil, err
			}
			cur = nv
			prefix = newPrefix
			continue
		}

		// Index access: slice/array
		idx := c.Index()
		newPrefix := prefix + "[" + strconv.Itoa(idx) + "]"
		kind := cur.Kind()
		if kind != reflect.Slice && kind != reflect.Array {
			return nil, fmt.Errorf("%s: cannot index %s", newPrefix, kind)
		}
		if idx < 0 || idx >= cur.Len() {
			return nil, fmt.Errorf("%s: index out of range, length is %d", newPrefix, cur.Len())
		}
		cur = cur.Index(idx)
		prefix = newPrefix
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
func accessKey(v reflect.Value, key, prefix string) (reflect.Value, error) {
	switch v.Kind() {
	case reflect.Struct:
		fv, sf, owner, ok := findStructFieldByKey(v, key)
		if !ok {
			return reflect.Value{}, fmt.Errorf("%s: field %q not found in %s", prefix, key, v.Type())
		}
		// Evaluate ForceSendFields on both the current struct and the declaring owner
		force := containsForceSendField(v, sf.Name) || containsForceSendField(owner, sf.Name)
		// ForceSendFields for nil pointer-to-struct: substitute zero struct.
		if fv.Kind() == reflect.Pointer && fv.IsNil() {
			et := fv.Type().Elem()
			if et.Kind() == reflect.Struct && force {
				return reflect.Zero(et), nil
			}
		}
		// Honor omitempty: if present and value is zero and not forced, treat as omitted (nil).
		jsonTag := structtag.JSONTag(sf.Tag.Get("json"))
		if jsonTag.OmitEmpty() && !force {
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
			return reflect.Value{}, fmt.Errorf("%s: map key must be string, got %s", prefix, kt)
		}
		mk := reflect.ValueOf(key)
		if kt != mk.Type() {
			mk = mk.Convert(kt)
		}
		mv := v.MapIndex(mk)
		if !mv.IsValid() {
			return reflect.Value{}, fmt.Errorf("%s: key %q not found in map", prefix, key)
		}
		return mv, nil
	default:
		return reflect.Value{}, fmt.Errorf("%s: cannot access key %q on %s", prefix, key, v.Kind())
	}
}

// findStructFieldByKey searches exported fields of struct v for a field matching key.
// It matches json tag name (when present and not "-") only.
// It also searches embedded anonymous structs (pointer or value) recursively.
func findStructFieldByKey(v reflect.Value, key string) (reflect.Value, reflect.StructField, reflect.Value, bool) {
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
			return v.Field(i), sf, v, true
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
		if out, osf, owner, ok := findStructFieldByKey(fv, key); ok {
			return out, osf, owner, true
		}
	}

	return reflect.Value{}, reflect.StructField{}, reflect.Value{}, false
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
