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
// This is a convenience function that parses the path string and calls ValidatePattern.
// Wildcards are allowed in the path.
func ValidateByString(t reflect.Type, path string) error {
	if path == "" {
		return nil
	}

	patternNode, err := structpath.ParsePattern(path)
	if err != nil {
		return err
	}

	return ValidatePattern(t, patternNode)
}

// ValidatePath reports whether the given path is valid for the provided type.
// It returns nil if the path resolves fully, or an error indicating where resolution failed.
// Paths cannot contain wildcards.
func ValidatePath(t reflect.Type, path *structpath.PathNode) error {
	if path.IsRoot() {
		return nil
	}
	return validateNodeSlice(t, path.AsSlice())
}

// ValidatePattern reports whether the given pattern path is valid for the provided type.
// It returns nil if the path resolves fully, or an error indicating where resolution failed.
// Patterns may include wildcards ([*] and .*).
func ValidatePattern(t reflect.Type, path *structpath.PatternNode) error {
	if path.IsRoot() {
		return nil
	}
	// PatternNode is type definition of PathNode, so we can cast the slice
	patternNodes := path.AsSlice()
	pathNodes := make([]*structpath.PathNode, len(patternNodes))
	for i, n := range patternNodes {
		pathNodes[i] = (*structpath.PathNode)(n)
	}
	return validateNodeSlice(t, pathNodes)
}

// validateNodeSlice is the shared implementation for ValidatePath and ValidatePattern.
func validateNodeSlice(t reflect.Type, nodes []*structpath.PathNode) error {
	cur := t
	for _, node := range nodes {
		for cur.Kind() == reflect.Pointer {
			cur = cur.Elem()
		}

		if _, isIndex := node.Index(); isIndex {
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				return fmt.Errorf("%s: cannot index %s", node.String(), kind)
			}
			cur = cur.Elem()
			continue
		}

		if node.BracketStar() {
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				return fmt.Errorf("%s: cannot use [*] on %s", node.String(), kind)
			}
			cur = cur.Elem()
			continue
		}
		if node.DotStar() {
			if cur.Kind() != reflect.Map {
				return fmt.Errorf("%s: cannot use .* on %s", node.String(), cur.Kind())
			}
			cur = cur.Elem()
			continue
		}

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
