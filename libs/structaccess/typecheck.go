package structaccess

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structdiff/structtag"
)

// ValidateByString reports whether the given path string is valid for the provided type.
// It returns nil if the path resolves fully, or an error indicating where resolution failed.
// This is a convenience function that parses the path string and calls Validate.
func ValidateByString(t reflect.Type, path string) error {
	if path == "" {
		return nil
	}

	p, err := dyn.NewPathFromString(path)
	if err != nil {
		return err
	}

	return Validate(t, p)
}

// Validate reports whether the given path is valid for the provided type.
// It returns nil if the path resolves fully, or an error indicating where resolution failed.
func Validate(t reflect.Type, path dyn.Path) error {
	if len(path) == 0 {
		return nil
	}

	cur := t
	prefix := ""
	for _, c := range path {
		// Always dereference pointers at the type level.
		for cur.Kind() == reflect.Pointer {
			cur = cur.Elem()
		}

		if c.Key() != "" {
			// Key access: struct field (by json tag) or map key.
			newPrefix := prefix
			if newPrefix == "" {
				newPrefix = c.Key()
			} else {
				newPrefix = newPrefix + "." + c.Key()
			}

			switch cur.Kind() {
			case reflect.Struct:
				sf, _, ok := findStructFieldByKeyType(cur, c.Key())
				if !ok {
					return fmt.Errorf("%s: field %q not found in %s", newPrefix, c.Key(), cur.String())
				}
				cur = sf.Type
			case reflect.Map:
				kt := cur.Key()
				if kt.Kind() != reflect.String {
					return fmt.Errorf("%s: map key must be string, got %s", newPrefix, kt)
				}
				cur = cur.Elem()
			default:
				return fmt.Errorf("%s: cannot access key %q on %s", newPrefix, c.Key(), cur.Kind())
			}
			prefix = newPrefix
			continue
		}

		// Index access: slice/array
		idx := c.Index()
		newPrefix := prefix + "[" + strconv.Itoa(idx) + "]"
		kind := cur.Kind()
		if kind != reflect.Slice && kind != reflect.Array {
			return fmt.Errorf("%s: cannot index %s", newPrefix, kind)
		}
		cur = cur.Elem()
		prefix = newPrefix
	}

	return nil
}

// findStructFieldByKeyType searches exported fields of struct type t for a field matching key.
// It matches json tag name (when present and not "-") only.
// It also searches embedded anonymous structs (pointer or value) recursively.
// Returns the StructField, the declaring owner type, and whether it was found.
func findStructFieldByKeyType(t reflect.Type, key string) (reflect.StructField, reflect.Type, bool) {
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
		if osf, owner, ok := findStructFieldByKeyType(ft, key); ok {
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
