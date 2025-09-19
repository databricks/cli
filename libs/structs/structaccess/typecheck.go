package structaccess

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

// ValidateByString reports whether the given path string is valid for the provided type.
// It returns nil if the path resolves fully, or an error indicating where resolution failed.
// This is a convenience function that parses the path string and calls Validate.
func ValidateByString(t reflect.Type, path string) error {
	if path == "" {
		return nil
	}

	pathNode, err := structpath.Parse(path)
	if err != nil {
		return err
	}

	return Validate(t, pathNode)
}

// Validate reports whether the given path is valid for the provided type.
// It returns nil if the path resolves fully, or an error indicating where resolution failed.
func Validate(t reflect.Type, path *structpath.PathNode) error {
	if path.IsRoot() {
		return nil
	}

	// Convert path to slice for easier iteration
	pathSegments := path.AsSlice()

	cur := t
	prefix := ""
	for _, node := range pathSegments {
		// Always dereference pointers at the type level.
		for cur.Kind() == reflect.Pointer {
			cur = cur.Elem()
		}

		// Handle different node types
		if idx, isIndex := node.Index(); isIndex {
			// Index access: slice/array
			newPrefix := prefix + "[" + strconv.Itoa(idx) + "]"
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				return fmt.Errorf("%s: cannot index %s", newPrefix, kind)
			}
			cur = cur.Elem()
			prefix = newPrefix
			continue
		}

		// Handle wildcards
		if node.DotStar() || node.BracketStar() {
			return fmt.Errorf("wildcards not supported: %s", path.String())
		}

		// Handle field or map key access
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
			newPrefix = prefix + "['" + key + "']"
		} else {
			return errors.New("unsupported path node type")
		}

		switch cur.Kind() {
		case reflect.Struct:
			sf, _, ok := FindStructFieldByKeyType(cur, key)
			if !ok {
				return fmt.Errorf("%s: field %q not found in %s", newPrefix, key, cur.String())
			}
			cur = sf.Type
		case reflect.Map:
			kt := cur.Key()
			if kt.Kind() != reflect.String {
				return fmt.Errorf("%s: map key must be string, got %s", newPrefix, kt)
			}
			cur = cur.Elem()
		default:
			return fmt.Errorf("%s: cannot access key %q on %s", newPrefix, key, cur.Kind())
		}
		prefix = newPrefix
	}

	return nil
}

// FindStructFieldByKeyType searches exported fields of struct type t for a field matching key.
// It matches json tag name (when present and not "-") only.
// It also searches embedded anonymous structs (pointer or value) recursively.
// Returns the StructField, the declaring owner type, and whether it was found.
func FindStructFieldByKeyType(t reflect.Type, key string) (reflect.StructField, reflect.Type, bool) {
	if t.Kind() != reflect.Struct {
		return reflect.StructField{}, reflect.TypeOf(nil), false
	}

	// First pass: direct fields
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		name := structtag.JSONTag(sf.Tag.Get("json")).Name()
		if name == "-" {
			name = ""
		}
		if name != "" && name == key {
			// Skip fields marked as internal/readonly
			btag := structtag.BundleTag(sf.Tag.Get("bundle"))
			if btag.Internal() || btag.ReadOnly() {
				continue
			}
			return sf, t, true
		}

		// Fallback to Go field name when no JSON tag
		if name == "" && sf.Name == key {
			// Skip fields marked as internal/readonly
			btag := structtag.BundleTag(sf.Tag.Get("bundle"))
			if btag.Internal() || btag.ReadOnly() {
				continue
			}
			return sf, t, true
		}
	}

	// Second pass: search embedded anonymous structs recursively (flattening semantics)
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.Anonymous {
			continue
		}
		ft := sf.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() != reflect.Struct {
			continue
		}
		if osf, owner, ok := FindStructFieldByKeyType(ft, key); ok {
			// Skip fields marked as internal/readonly
			btag := structtag.BundleTag(osf.Tag.Get("bundle"))
			if btag.Internal() || btag.ReadOnly() {
				// Treat as not found and continue
				continue
			}
			return osf, owner, true
		}
	}

	return reflect.StructField{}, reflect.TypeOf(nil), false
}
