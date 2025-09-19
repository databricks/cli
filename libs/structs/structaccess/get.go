package structaccess

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

// GetByString returns the value at the given path inside v.
// This is a convenience function that parses the path string and calls Get.
func GetByString(v any, path string) (any, error) {
	if path == "" {
		return v, nil
	}

	pathNode, err := structpath.Parse(path)
	if err != nil {
		return nil, err
	}

	return Get(v, pathNode)
}

// Get returns the value at the given path inside v.
// - For structs: supports both .field and ['field'] notation
// - For maps: supports both ['key'] and .key notation
// - For slices/arrays: an index [N] selects the N-th element.
// - Wildcards ("*" or "[*]") are not supported and return an error.
func Get(v any, path *structpath.PathNode) (any, error) {
	if path.IsRoot() {
		return v, nil
	}

	// Convert path to slice for easier iteration
	pathSegments := path.AsSlice()

	cur := reflect.ValueOf(v)
	prefix := ""
	for _, node := range pathSegments {
		var ok bool
		cur, ok = deref(cur)
		if !ok {
			// cannot proceed further due to nil encountered at current location
			return nil, fmt.Errorf("%s: cannot access nil value", prefix)
		}

		if idx, isIndex := node.Index(); isIndex {
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
			continue
		}

		if node.DotStar() || node.BracketStar() {
			return nil, fmt.Errorf("wildcards not supported: %s", path.String())
		}

		var key string
		var newPrefix string

		if field, isField := node.Field(); isField {
			key = field
			newPrefix = prefix
			if newPrefix == "" {
				newPrefix = key
			} else {
				newPrefix = newPrefix + "." + key
			}
		} else if mapKey, isMapKey := node.MapKey(); isMapKey {
			key = mapKey
			newPrefix = prefix + "[" + structpath.EncodeMapKey(key) + "]"
		} else {
			return nil, errors.New("unsupported path node type")
		}

		nv, err := accessKey(cur, key, newPrefix)
		if err != nil {
			return nil, err
		}
		cur = nv
		prefix = newPrefix
	}

	// If the current value is invalid (e.g., omitted due to omitempty), return nil.
	if !cur.IsValid() {
		return nil, nil
	}

	// If the final value is a nil pointer or nil interface, return nil.
	if (cur.Kind() == reflect.Pointer || cur.Kind() == reflect.Interface) && cur.IsNil() {
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
		return accessStructField(v, key, prefix)
	case reflect.Map:
		return accessMapKey(v, key, prefix)
	default:
		return reflect.Value{}, fmt.Errorf("%s: cannot access key %q on %s", prefix, key, v.Kind())
	}
}

// accessStructField returns the struct field value selected by key from v.
func accessStructField(v reflect.Value, key, prefix string) (reflect.Value, error) {
	fv, sf, owner, ok := findStructFieldByKey(v, key)
	if !ok {
		return reflect.Value{}, fmt.Errorf("%s: field %q not found in %s", prefix, key, v.Type())
	}
	// Evaluate ForceSendFields on both the current struct and the declaring owner
	force := containsForceSendField(v, sf.Name) || containsForceSendField(owner, sf.Name)

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
}

// accessMapKey returns the map entry value selected by key from v.
func accessMapKey(v reflect.Value, key, prefix string) (reflect.Value, error) {
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
			// Skip fields marked as internal or readonly via bundle tag
			btag := structtag.BundleTag(sf.Tag.Get("bundle"))
			if btag.Internal() || btag.ReadOnly() {
				continue
			}
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
			// Skip fields marked as internal or readonly via bundle tag
			btag := structtag.BundleTag(osf.Tag.Get("bundle"))
			if btag.Internal() || btag.ReadOnly() {
				// Treat as not found and continue searching other anonymous fields
				continue
			}
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
