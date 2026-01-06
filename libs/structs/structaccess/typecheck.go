package structaccess

import (
	"errors"
	"fmt"
	"reflect"

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
	for _, node := range pathSegments {
		// Always dereference pointers at the type level.
		for cur.Kind() == reflect.Pointer {
			cur = cur.Elem()
		}

		if _, isIndex := node.Index(); isIndex {
			// Index access: slice/array
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				return fmt.Errorf("%s: cannot index %s", node.String(), kind)
			}
			cur = cur.Elem()
			continue
		}

		// Handle wildcards
		if node.DotStar() || node.BracketStar() {
			return fmt.Errorf("wildcards not supported: %s", path.String())
		}

		// Handle key-value selector: validates that we can index the slice/array
		if _, _, isKeyValue := node.KeyValue(); isKeyValue {
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				return fmt.Errorf("%s: cannot use key-value syntax on %s", node.String(), kind)
			}
			cur = cur.Elem()
			continue
		}

		key, ok := node.StringKey()

		if !ok {
			return errors.New("unsupported path node type")
		}

		switch cur.Kind() {
		case reflect.Struct:
			sf, _, ok := FindStructFieldByKeyType(cur, key)
			if !ok {
				return fmt.Errorf("%s: field %q not found in %s", node.String(), key, cur.String())
			}
			cur = sf.Type
		case reflect.Map:
			kt := cur.Key()
			if kt.Kind() != reflect.String {
				return fmt.Errorf("%s: map key must be string, got %s", node.String(), kt)
			}
			cur = cur.Elem()
		default:
			return fmt.Errorf("%s: cannot access key %q on %s", node.String(), key, cur.Kind())
		}
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
