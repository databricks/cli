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
	// PathNode is type definition of PatternNode, so we can cast directly
	return ValidatePattern(t, (*structpath.PatternNode)(path))
}

// ValidatePattern reports whether the given pattern path is valid for the provided type.
// It returns nil if the path resolves fully, or an error indicating where resolution failed.
// Patterns may include wildcards ([*] and .*).
func ValidatePattern(t reflect.Type, path *structpath.PatternNode) error {
	if path.IsRoot() {
		return nil
	}
	return validateNodeSlice(t, path.AsSlice())
}

// validateNodeSlice is the implementation for ValidatePattern.
func validateNodeSlice(t reflect.Type, nodes []*structpath.PatternNode) error {
	cur := t
	for _, node := range nodes {
		// Always dereference pointers at the type level.
		for cur.Kind() == reflect.Pointer {
			cur = cur.Elem()
		}

		// Index access: slice/array
		if idx, isIndex := node.Index(); isIndex {
			if cur.Kind() == reflect.Struct {
				if embedType := findEmbedFieldType(cur); embedType != nil {
					cur = embedType
				}
			}
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				// Terraform represents single-block fields as lists and uses [0] to access them.
				// Treat [0] on a struct as a no-op so TF-style paths work against DABs structs.
				if idx == 0 && kind == reflect.Struct {
					continue
				}
				return fmt.Errorf("%s: cannot index %s", node.String(), kind)
			}
			cur = cur.Elem()
			continue
		}

		// Handle wildcards - treat like index/key access
		if node.BracketStar() {
			if cur.Kind() == reflect.Struct {
				if embedType := findEmbedFieldType(cur); embedType != nil {
					cur = embedType
				}
			}
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

		// Handle key-value selector: validates that we can index the slice/array
		if _, _, isKeyValue := node.KeyValue(); isKeyValue {
			if cur.Kind() == reflect.Struct {
				if embedType := findEmbedFieldType(cur); embedType != nil {
					cur = embedType
				}
			}
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
	if sf, ok := findDirectFieldByKeyType(t, key); ok {
		return sf, t, true
	}

	// Second pass: search embedded anonymous structs breadth-first, mirroring findStructFieldByKey
	// (get.go) so a path validates against the same field Get and Set resolve it to, which is
	// the one encoding/json serializes: the shallower of two same-named fields.
	// See findStructFieldByKey: a cyclic embedding must not be walked twice.
	seen := map[reflect.Type]bool{t: true}
	level := embeddedStructTypes(t)
	for len(level) > 0 {
		var next []reflect.Type
		var found []struct {
			field reflect.StructField
			owner reflect.Type
		}
		for _, ft := range level {
			if sf, ok := findDirectFieldByKeyType(ft, key); ok {
				found = append(found, struct {
					field reflect.StructField
					owner reflect.Type
				}{sf, ft})
				continue
			}
			for _, deeper := range embeddedStructTypes(ft) {
				if seen[deeper] {
					continue
				}
				seen[deeper] = true
				next = append(next, deeper)
			}
		}
		if len(found) == 1 {
			return found[0].field, found[0].owner, true
		}
		if len(found) > 1 {
			// Ambiguous at this depth, which encoding/json resolves by omitting the field; see
			// findStructFieldByKey in get.go.
			return reflect.StructField{}, reflect.TypeOf(nil), false
		}
		level = next
	}

	return reflect.StructField{}, reflect.TypeOf(nil), false
}

// findDirectFieldByKeyType matches key against the struct's own fields, by json tag name.
func findDirectFieldByKeyType(t reflect.Type, key string) (reflect.StructField, bool) {
	for sf := range t.Fields() {
		if sf.PkgPath != "" { // unexported
			continue
		}
		name := structtag.JSONTag(sf.Tag.Get("json")).Name()
		if name == "-" || sf.Name == EmbeddedSliceFieldName {
			continue
		}
		if name != key {
			continue
		}
		// Skip fields marked as internal/readonly
		btag := structtag.BundleTag(sf.Tag.Get("bundle"))
		if btag.Internal() || btag.ReadOnly() {
			continue
		}
		return sf, true
	}
	return reflect.StructField{}, false
}

// embeddedStructTypes returns the anonymous struct fields of t, dereferenced.
func embeddedStructTypes(t reflect.Type) []reflect.Type {
	var out []reflect.Type
	for sf := range t.Fields() {
		if !sf.Anonymous {
			continue
		}
		ft := sf.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			out = append(out, ft)
		}
	}
	return out
}
